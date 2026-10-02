// Command guardd is the lightweight guard service.
//
// Three shapes, one core:
//
//	guardd                          # CLI: reads content on stdin, prints the Result JSON
//	guardd -content "..."           # CLI: one-shot, argument form
//	guardd -serve :8768 -token s3cr3t # HTTP: POST /check {input} -> Result JSON
//
// The HTTP shape is authenticated: /check requires the shared secret in the
// X-Operator-Token header (the fleet's operator-token pattern), and -token is
// mandatory — guardd refuses to start serving without it rather than run
// unauthenticated. /healthz stays open so probes keep working. The bind
// address defaults to loopback: ":8768" becomes "127.0.0.1:8768" unless the
// operator names an explicit host.
//
// Zero dependencies, one static binary. task-router (Python) calls the HTTP
// form; crier (Go) imports the package directly; shell scripts pipe through the
// CLI. Same verdict everywhere.
package main

import (
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
	"time"

	"github.com/trouble-agent/guard"
)

// OperatorTokenHeader is the header carrying the shared secret on /check.
// Same name as the fleet's operator-token convention so one credential works
// across surfaces.
const OperatorTokenHeader = "X-Operator-Token"

func main() {
	var (
		serve   = flag.String("serve", "", "address to serve HTTP on, e.g. :8768 (empty = CLI mode)")
		token   = flag.String("token", "", "shared secret required in the "+OperatorTokenHeader+" header on /check (REQUIRED with -serve)")
		content = flag.String("content", "", "content to classify (empty = read stdin)")
		source  = flag.String("source", "", "source label, e.g. github_pr (drives policy overrides)")
		channel = flag.String("channel", "", "channel label, e.g. crier|task-router")
		fail    = flag.Bool("fail-closed", true, "block on guard error (false = deliver flagged)")
		pretty  = flag.Bool("pretty", false, "indent the JSON output")
	)
	flag.Parse()

	// Refuse to serve unauthenticated. A guardd reachable on the wire with no
	// token is an open classifier and an egress proxy for anyone who can reach
	// the port, so this is a startup failure, never a warning.
	if *serve != "" && *token == "" {
		log.Fatalf("guardd: refusing to serve HTTP without -token: /check would be unauthenticated. "+
			"Pass -token <secret> (clients send it as the %s header); /healthz stays open for probes.", OperatorTokenHeader)
	}

	g := &guard.Guard{Classifier: guard.NewJev(), Policy: guard.DefaultPolicy()}
	g.Policy.FailClosed = *fail

	if *serve != "" {
		serveHTTP(*serve, g, *token)
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
func newHandler(g *guard.Guard, token string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok\n")
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
		json.NewEncoder(w).Encode(g.Check(ctx, in))
	})
	return mux
}

func serveHTTP(addr string, g *guard.Guard, token string) {
	if token == "" {
		log.Fatalf("guardd: refusing to serve HTTP without -token (unauthenticated /check)")
	}
	addr = normalizeBindAddr(addr)
	log.Printf("guardd listening on %s (POST /check [%s required], GET /healthz)", addr, OperatorTokenHeader)
	if err := http.ListenAndServe(addr, newHandler(g, token)); err != nil {
		log.Fatalf("serve: %v", err)
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
