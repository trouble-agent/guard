// Command guardd is the lightweight guard service.
//
// Three shapes, one core:
//
//	guardd                          # CLI: reads content on stdin, prints the Result JSON
//	guardd -content "..."           # CLI: one-shot, argument form
//	GUARD_TOKEN=s3cr3t guardd -serve :8768  # HTTP: POST /check {input} -> Result JSON
//
// The HTTP shape is authenticated: /check requires the shared secret in the
// X-Operator-Token header (the fleet's operator-token pattern), and a token is
// mandatory — guardd refuses to start serving without one rather than run
// unauthenticated. /healthz stays open so probes keep working. The bind
// address defaults to loopback: ":8768" becomes "127.0.0.1:8768" unless the
// operator names an explicit host.
//
// The token is resolved by precedence env > tokenfile > argv:
//
//	GUARD_TOKEN=... guardd -serve :8768      # env (no argv secret)
//	guardd -serve :8768 -tokenfile f         # 0600 file (group/world-readable refused)
//	guardd -serve :8768 -token s3cr3t        # argv (works, but warned: visible in ps)
//
// CLI exit codes: 0 for any verdict the guard served — including a normal
// block (blocking is the guard working, not failing). Exit code 2 fires when
// the guard could not judge at all (guard_error: no OpenRouter API key
// configured, egress disabled, classifier failure): the error verdict is
// still emitted with Result.Errored set, and the CLI exits 2 whichever way
// the policy resolved it, so callers keying on the exit status can
// distinguish "classified, blocked" from "never classified". Turning
// fail-closed off (-fail-closed=false) only reshapes that error verdict in
// the JSON — allow flagged for review instead of block+quarantine; the exit
// code stays 2. Serving mode (/check) is unaffected: HTTP answers in-band
// and the process keeps running.
//
// Zero dependencies, one static binary. task-router (Python) calls the HTTP
// form; crier (Go) imports the package directly; shell scripts pipe through the
// CLI. Same verdict everywhere.
package main

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/trouble-agent/guard"
)

// OperatorTokenHeader is the header carrying the shared secret on /check.
// Same name as the fleet's operator-token convention so one credential works
// across surfaces.
const OperatorTokenHeader = "X-Operator-Token"

// GuardTokenEnv is the environment variable carrying the shared secret so the
// token never has to appear in argv (ps and /proc/<pid>/cmdline are readable by
// every user on the host).
const GuardTokenEnv = "GUARD_TOKEN"

// GuardEgressEnv controls whether the hosted Jev classifier may receive
// untrusted content. Egress is enabled by default for backwards compatibility;
// set GUARD_EGRESS_ENABLED=false to refuse classification rather than send data.
const GuardEgressEnv = "GUARD_EGRESS_ENABLED"

type egressDisabledClassifier struct{}

func (egressDisabledClassifier) Name() string { return "disabled" }
func (egressDisabledClassifier) Classify(context.Context, string) (guard.Signals, error) {
	return guard.Signals{}, fmt.Errorf("hosted classifier egress is disabled (%s=false); no local classifier is configured", GuardEgressEnv)
}

func classifierEgressEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "no", "off":
		return false
	default:
		return true
	}
}

func main() {
	var (
		serve     = flag.String("serve", "", "address to serve HTTP on, e.g. :8768 (empty = CLI mode)")
		token     = flag.String("token", "", "shared secret required in the "+OperatorTokenHeader+" header on /check (REQUIRED with -serve; prefer GUARD_TOKEN or -tokenfile)")
		tokenfile = flag.String("tokenfile", "", "path to a file holding the shared secret; must be 0600 (recommended: keeps the token out of argv)")
		content   = flag.String("content", "", "content to classify (empty = read stdin)")
		source    = flag.String("source", "", "source label, e.g. github_pr (drives policy overrides)")
		channel   = flag.String("channel", "", "channel label, e.g. crier|task-router")
		fail      = flag.Bool("fail-closed", true, "block on guard error (false = deliver flagged)")
		pretty    = flag.Bool("pretty", false, "indent the JSON output")
	)
	flag.Parse()

	egressEnabled := classifierEgressEnabled(os.Getenv(GuardEgressEnv))
	var classifier guard.Classifier = egressDisabledClassifier{}
	if egressEnabled {
		classifier = guard.NewJev()
	}
	g := &guard.Guard{Classifier: classifier, Policy: guard.DefaultPolicy()}
	g.Policy.FailClosed = *fail

	if *serve != "" {
		resolved, prov, err := resolveToken(os.Getenv(GuardTokenEnv), *tokenfile, *token)
		if err != nil {
			log.Fatalf("guardd: %v", err)
		}
		warnTokenProvenance(prov, *token)
		serveHTTP(*serve, g, resolved, egressEnabled)
		return
	}

	text := *content
	if text == "" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("read stdin: %v", err)
		}
		text = string(b)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	res := g.Check(ctx, guard.Input{Source: *source, Channel: *channel, Content: text})
	emit(res, *pretty)
	// QA-GUARD-002: the CLI errorpath contract — a broken (config/egress-failed)
	// run must not look successful to rc-keying shell/CI callers. The fail-closed
	// DECISION in the JSON is the product; the exit code distinguishes
	// "classified" from "guard_error fired". Serving mode (/check) is unaffected:
	// there the block verdict is the legitimate answer to a request.
	if res.Errored {
		os.Exit(2)
	}
}

// normalizeBindAddr keeps the operator's address but defaults an omitted host to
// loopback, so the bare fleet spelling ":8768" does not bind every interface.
// An explicit host ("0.0.0.0:8768", "10.0.0.5:8768", "localhost:9000") passes
// through unchanged. Anything SplitHostPort cannot parse is returned as-is.
func normalizeBindAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// authorized reports whether the request carries the configured shared secret.
// An unset token authorizes nothing: the handler fails closed even if it is
// somehow reached without -token. Comparison is constant-time.
func authorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	got := r.Header.Get(OperatorTokenHeader)
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

// newHandler builds the HTTP surface: /healthz open, /check token-gated.
func newHandler(g *guard.Guard, token string, egressEnabled bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Status        string `json:"status"`
			EgressEnabled bool   `json:"egress_enabled"`
		}{Status: "ok", EgressEnabled: egressEnabled})
	})
	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		// Auth before anything else: an unauthenticated request learns nothing,
		// not even whether its method or body would have been accepted.
		if !authorized(r, token) {
			http.Error(w, "unauthorized: missing or invalid "+OperatorTokenHeader, http.StatusUnauthorized)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var in guard.Input
		if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&in); err != nil {
			http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if in.Content == "" {
			http.Error(w, "content is required", http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(g.Check(ctx, in))
	})
	return mux
}

func serveHTTP(addr string, g *guard.Guard, token string, egressEnabled bool) {
	if err := requireToken(token); err != nil {
		log.Fatalf("guardd: %v", err)
	}
	addr = normalizeBindAddr(addr)
	if egressEnabled {
		log.Printf("WARNING: classifier egress is enabled; untrusted content sent to /check is transmitted to the hosted Jev API outside this host. Set %s=false to disable egress (classification will fail closed without a local classifier).", GuardEgressEnv)
	} else {
		log.Printf("WARNING: classifier egress is DISABLED; /check will refuse classification because no local classifier is configured")
	}
	log.Printf("guardd listening on %s (POST /check [%s required], GET /healthz)", addr, OperatorTokenHeader)
	if err := http.ListenAndServe(addr, newHandler(g, token, egressEnabled)); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// tokenProvenance says where the serving token came from, so the launcher can
// surface the trade-offs of each form.
type tokenProvenance int

const (
	tokenFromTokenFile tokenProvenance = iota // 0600 -tokenfile
	tokenFromEnv                              // GUARD_TOKEN
	tokenFromArgv                             // -token: visible in ps / /proc cmdline
)

// resolveToken resolves the shared secret by precedence env > tokenfile > argv,
// the raw flag strings it was given, or a refusal error (guardd never serves
// unauthenticated).
//
// Precedence env > tokenfile lets an operator stage a tokenfile (e.g. in a
// unit file) and still override it ad hoc without editing the file. argv is
// last and deprecated: it works, but the secret is world-readable in the
// process list, so serving with it is loudly warned about on stderr.
func resolveToken(envVal, tokenfilePath, argvToken string) (token string, prov tokenProvenance, err error) {
	switch {
	case envVal != "":
		return envVal, tokenFromEnv, nil
	case tokenfilePath != "":
		b, err := readTokenFile(tokenfilePath)
		if err != nil {
			return "", 0, err
		}
		return string(b), tokenFromTokenFile, nil
	case argvToken != "":
		return argvToken, tokenFromArgv, nil
	default:
		return "", 0, fmt.Errorf("no token provided: refusing to serve an unauthenticated /check; "+
			"set %s, or pass -tokenfile <path> (0600)", GuardTokenEnv)
	}
}

// readTokenFile reads a shared-secret file. The token is the file contents
// with surrounding ASCII whitespace trimmed (editors leave trailing newlines;
// tokens themselves never contain spaces). The file must be owner-only:
// 0400 or 0600. Anything group- or world-accessible is refused — a secret any
// other account can read is not a secret.
func readTokenFile(path string) ([]byte, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("tokenfile: %w", err)
	}
	perm := fi.Mode().Perm()
	if perm&0o077 != 0 {
		return nil, fmt.Errorf("tokenfile %s: mode %04o is too open (group/world bits set); chmod 600 it first", path, perm)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("tokenfile: %w", err)
	}
	return bytes.Trim(b, " 	\n\r\v\f"), nil
}

// requireToken refuses an unset token: a guardd reachable on the wire with no
// secret is an open classifier, so this is a startup failure, never a warning.
func requireToken(token string) error {
	if token == "" {
		return fmt.Errorf("no token provided: refusing to serve an unauthenticated /check; "+
			"set %s, or pass -tokenfile <path> (0600)", GuardTokenEnv)
	}
	return nil
}

// warnTokenProvenance tells the operator how the token was supplied, and
// warns loudly when it came from argv — ps and /proc/<pid>/cmdline expose the
// full command line to every user on the host, so an argv token is readable
// process-wide. Preferred: GUARD_TOKEN env or a 0600 -tokenfile.
func warnTokenProvenance(prov tokenProvenance, argvToken string) {
	switch prov {
	case tokenFromArgv:
		fmt.Fprintf(os.Stderr, "WARNING: guardd -token used: the shared secret is visible in the process list (ps, /proc/<pid>/cmdline) to every user on this host. Prefer %s or -tokenfile <path> (0600).\n", GuardTokenEnv)
	case tokenFromEnv:
		fmt.Fprintf(os.Stderr, "guardd: token loaded from %s\n", GuardTokenEnv)
	case tokenFromTokenFile:
		fmt.Fprintf(os.Stderr, "guardd: token loaded from tokenfile\n")
	}
}

func emit(v any, pretty bool) {
	enc := json.NewEncoder(os.Stdout)
	if pretty {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
}
