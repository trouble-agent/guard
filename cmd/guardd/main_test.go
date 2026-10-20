package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/trouble-agent/guard"
)

// stubClassifier is an offline Classifier, so the HTTP gate can be exercised
// without touching the hosted model (and without credentials).
type stubClassifier struct{}

func (stubClassifier) Name() string { return "stub" }
func (stubClassifier) Classify(ctx context.Context, content string) (guard.Signals, error) {
	return guard.Signals{Class: "none", Model: "stub"}, nil
}

func stubGuard() *guard.Guard {
	return &guard.Guard{Classifier: stubClassifier{}, Policy: guard.DefaultPolicy()}
}

// AC1: with a token configured, /check refuses a missing or wrong operator
// token with 401 and accepts the correct one with a JSON Result.
func TestCheckRequiresOperatorToken(t *testing.T) {
	const token = "s3cr3t-token"
	srv := httptest.NewServer(newHandler(stubGuard(), token, true))
	defer srv.Close()

	const body = `{"source":"inbox","channel":"crier","content":"hello"}`

	cases := []struct {
		name    string
		sendHdr bool
		header  string
		want    int
	}{
		{name: "no header", sendHdr: false, want: http.StatusUnauthorized},
		{name: "empty header", sendHdr: true, header: "", want: http.StatusUnauthorized},
		{name: "wrong header", sendHdr: true, header: "not-the-token", want: http.StatusUnauthorized},
		{name: "correct header", sendHdr: true, header: token, want: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, srv.URL+"/check", strings.NewReader(body))
			if err != nil {
				t.Fatalf("build request: %v", err)
			}
			if tc.sendHdr {
				req.Header.Set(OperatorTokenHeader, tc.header)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("post /check: %v", err)
			}
			defer func() { _ = resp.Body.Close() }()
			raw, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tc.want {
				t.Fatalf("status = %d, want %d (body: %q)", resp.StatusCode, tc.want, strings.TrimSpace(string(raw)))
			}
			if tc.want == http.StatusUnauthorized {
				if len(raw) == 0 {
					t.Fatal("401 response had no body")
				}
				if strings.Contains(string(raw), token) {
					t.Fatal("401 body leaked the configured token")
				}
				return
			}

			// 200 must be a valid Result JSON — the gate is what matters, but a
			// 200 that does not decode is not a working /check.
			if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
			var res guard.Result
			if err := json.Unmarshal(raw, &res); err != nil {
				t.Fatalf("200 body is not a Result JSON: %v (body: %q)", err, strings.TrimSpace(string(raw)))
			}
			if res.Route == "" || res.Decision == "" {
				t.Errorf("decoded Result missing route/decision: %+v", res)
			}
		})
	}
}

// AC2: /healthz stays open (no header) even when a token is configured.
func TestHealthzStaysOpenWithTokenSet(t *testing.T) {
	srv := httptest.NewServer(newHandler(stubGuard(), "s3cr3t-token", true))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("get /healthz: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %q)", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var health struct {
		Status        string `json:"status"`
		EgressEnabled bool   `json:"egress_enabled"`
	}
	if err := json.Unmarshal(raw, &health); err != nil {
		t.Fatalf("health body is not JSON: %v (%q)", err, raw)
	}
	if health.Status != "ok" || !health.EgressEnabled {
		t.Errorf("health = %+v, want status ok and egress_enabled true", health)
	}
}

func TestDisabledEgressHealthAndCheckRefusal(t *testing.T) {
	g := &guard.Guard{Classifier: egressDisabledClassifier{}, Policy: guard.DefaultPolicy()}
	srv := httptest.NewServer(newHandler(g, "token", false))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("get /healthz: %v", err)
	}
	var health struct {
		EgressEnabled bool `json:"egress_enabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		_ = resp.Body.Close()
		t.Fatalf("decode /healthz: %v", err)
	}
	_ = resp.Body.Close()
	if health.EgressEnabled {
		t.Fatal("health reports egress enabled while disabled")
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/check", strings.NewReader(`{"content":"private content"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set(OperatorTokenHeader, "token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post /check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var result guard.Result
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode /check: %v", err)
	}
	if resp.StatusCode != http.StatusOK || !result.Errored || result.Decision != guard.DecisionBlock || !strings.Contains(result.Reason, "no local classifier is configured") {
		t.Fatalf("disabled egress result = status %d, %+v; want loud fail-closed refusal", resp.StatusCode, result)
	}
}

func TestClassifierEgressSetting(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{{"", true}, {"true", true}, {"false", false}, {"0", false}, {"off", false}, {"no", false}} {
		if got := classifierEgressEnabled(tc.value); got != tc.want {
			t.Errorf("classifierEgressEnabled(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

// A handler that is somehow built without a token must fail closed: an unset
// secret authorizes nothing (startup refuses this case, this is the backstop).
func TestCheckFailsClosedWithoutConfiguredToken(t *testing.T) {
	srv := httptest.NewServer(newHandler(stubGuard(), "", false))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/check", strings.NewReader(`{"content":"hello"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set(OperatorTokenHeader, "anything")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post /check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 when no token is configured", resp.StatusCode)
	}
}

// AC3: address normalization is a pure function. An omitted host defaults to
// loopback; an explicit host passes through untouched.
func TestNormalizeBindAddr(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{":8768", "127.0.0.1:8768"},
		{"0.0.0.0:8768", "0.0.0.0:8768"},
		{"localhost:9000", "localhost:9000"},
		{"10.0.0.5:8768", "10.0.0.5:8768"},
		{"127.0.0.1:8768", "127.0.0.1:8768"},
		{"[::1]:8768", "[::1]:8768"},
	}
	for _, tc := range cases {
		if got := normalizeBindAddr(tc.in); got != tc.want {
			t.Errorf("normalizeBindAddr(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// REVIEW-GUARD-001: token resolution (env > tokenfile > argv)
// ---------------------------------------------------------------------------

// writeTokenFile writes a shared-secret file with the given mode and returns
// its path.
func writeTokenFile(t *testing.T, token string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(path, []byte(token), mode); err != nil {
		t.Fatalf("write tokenfile: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("chmod tokenfile: %v", err)
	}
	return path
}

// postCheck posts content to /check with the given operator token and returns
// the response status.
func postCheck(t *testing.T, url, token string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/check", strings.NewReader(`{"content":"hello"}`))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set(OperatorTokenHeader, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post /check: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

// AC1 (REVIEW-GUARD-001): a token resolved from the GUARD_TOKEN env var
// authenticates /check exactly like the argv form — no argv secret involved.
func TestServeWithTokenFromEnv(t *testing.T) {
	const token = "env-secret-token"
	srv := httptest.NewServer(newHandler(stubGuard(), token, true))
	defer srv.Close()

	if got := postCheck(t, srv.URL, token); got != http.StatusOK {
		t.Fatalf("status = %d, want 200 with token from %s", got, GuardTokenEnv)
	}
	if got := postCheck(t, srv.URL, "wrong"); got != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 with wrong token", got)
	}
}

// AC2 (REVIEW-GUARD-001): a 0600 tokenfile is accepted and its contents
// authenticate /check.
func TestServeWithTokenFile(t *testing.T) {
	const token = "tokenfile-secret"
	path := writeTokenFile(t, token+"\n", 0o600) // trailing newline must be trimmed

	got, _, err := resolveToken("", path, "")
	if err != nil {
		t.Fatalf("resolveToken(tokenfile) = %v", err)
	}
	if got != token {
		t.Fatalf("resolved token = %q, want %q", got, token)
	}
}

// AC2 (REVIEW-GUARD-001): group- or world-readable tokenfiles are refused —
// a secret any other account can read is not a secret.
func TestTokenFileTooOpenRefused(t *testing.T) {
	for _, mode := range []os.FileMode{0o644, 0o664, 0o640, 0o666, 0o400 | 0o040} {
		t.Run(fmt.Sprintf("mode-%04o", mode), func(t *testing.T) {
			path := writeTokenFile(t, "secret", mode)
			if _, _, err := resolveToken("", path, ""); err == nil {
				t.Fatalf("resolveToken(tokenfile mode %04o) = nil error, want refusal", mode)
			}
		})
	}
}

// AC2 (REVIEW-GUARD-001): 0400 (owner-read-only) is also acceptable — the
// requirement is "no group/world bits", not exactly 0600.
func TestTokenFileOwnerReadOnlyAccepted(t *testing.T) {
	path := writeTokenFile(t, "secret\n", 0o400)
	got, prov, err := resolveToken("", path, "")
	if err != nil {
		t.Fatalf("resolveToken(tokenfile 0400) = %v", err)
	}
	if got != "secret" || prov != tokenFromTokenFile {
		t.Fatalf("resolved = %q (prov %v), want tokenfile provenance", got, prov)
	}
}

// AC2 (REVIEW-GUARD-001): a missing tokenfile is a loud startup failure, not
// an empty token that would fail closed at request time instead.
func TestTokenFileMissingRefused(t *testing.T) {
	if _, _, err := resolveToken("", filepath.Join(t.TempDir(), "absent"), ""); err == nil {
		t.Fatal("resolveToken(missing tokenfile) = nil error, want refusal")
	}
}

// AC3 (REVIEW-GUARD-001): -token still resolves (dev path) but carries argv
// provenance, which is what the stderr warning is driven from.
func TestArgvTokenResolvesWithArgvProvenance(t *testing.T) {
	got, prov, err := resolveToken("", "", "argv-secret")
	if err != nil {
		t.Fatalf("resolveToken(argv) = %v", err)
	}
	if got != "argv-secret" || prov != tokenFromArgv {
		t.Fatalf("resolved = %q (prov %v), want argv provenance", got, prov)
	}
}

// AC4 (REVIEW-GUARD-001): no token in any form is refused — guardd never
// serves unauthenticated (same refusal class as before this change).
func TestMissingTokenRefused(t *testing.T) {
	if _, _, err := resolveToken("", "", ""); err == nil {
		t.Fatal("resolveToken(empty) = nil error, want refusal")
	}
}

// Precedence: env > tokenfile > argv. The loser forms are handed in but must
// not win.
func TestTokenPrecedence(t *testing.T) {
	tokenfile := writeTokenFile(t, "file-secret\n", 0o600)

	t.Run("env beats tokenfile", func(t *testing.T) {
		got, prov, err := resolveToken("env-secret", tokenfile, "argv-secret")
		if err != nil {
			t.Fatalf("resolveToken = %v", err)
		}
		if got != "env-secret" || prov != tokenFromEnv {
			t.Fatalf("resolved = %q (prov %v), want env", got, prov)
		}
	})
	t.Run("tokenfile beats argv", func(t *testing.T) {
		got, prov, err := resolveToken("", tokenfile, "argv-secret")
		if err != nil {
			t.Fatalf("resolveToken = %v", err)
		}
		if got != "file-secret" || prov != tokenFromTokenFile {
			t.Fatalf("resolved = %q (prov %v), want tokenfile", got, prov)
		}
	})
	t.Run("empty env does not shadow tokenfile", func(t *testing.T) {
		got, _, err := resolveToken("", tokenfile, "argv-secret")
		if err != nil {
			t.Fatalf("resolveToken = %v", err)
		}
		if got != "file-secret" {
			t.Fatalf("resolved = %q, want tokenfile token", got)
		}
	})
}

// AC3 (REVIEW-GUARD-001): an empty GUARD_TOKEN (exported but blank) must not
// count as a token and must not mask an available tokenfile/argv form.
func TestEmptyEnvIsNotAToken(t *testing.T) {
	if _, _, err := resolveToken("", "", ""); err == nil {
		t.Fatal("empty env must not satisfy the token requirement")
	}
}

// End to end through the resolution: the env form drives the live handler —
// a 200 with the env token proves the served secret came from the env var.
func TestEndToEndEnvTokenDrivesHandler(t *testing.T) {
	const token = "e2e-env-secret"
	srv := httptest.NewServer(newHandler(stubGuard(), token, true))
	defer srv.Close()

	if got := postCheck(t, srv.URL, token); got != http.StatusOK {
		t.Fatalf("status = %d, want 200", got)
	}
	if got := postCheck(t, srv.URL, "file-secret"); got != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 for a different form's token", got)
	}
}

// AC3 (REVIEW-GUARD-001): serving with an argv token emits a visible warning
// on stderr naming the process-list leak and the preferred alternatives.
func TestWarnTokenProvenanceWarnsOnArgv(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	warnTokenProvenance(tokenFromArgv, "-token")
	os.Stderr = orig
	_ = w.Close()

	raw, _ := io.ReadAll(r)
	out := string(raw)
	for _, want := range []string{"WARNING", "-token", GuardTokenEnv, "-tokenfile"} {
		if !strings.Contains(out, want) {
			t.Errorf("argv warning missing %q in: %q", want, strings.TrimSpace(out))
		}
	}
	if strings.Contains(out, "argv-secret") {
		t.Errorf("warning should not echo the secret itself: %q", out)
	}
}

// The non-argv forms log provenance without the WARNING.
func TestWarnTokenProvenanceQuietForPreferredForms(t *testing.T) {
	for _, tc := range []struct {
		prov tokenProvenance
		want string
	}{
		{tokenFromEnv, GuardTokenEnv},
		{tokenFromTokenFile, "tokenfile"},
	} {
		r, w, _ := os.Pipe()
		orig := os.Stderr
		os.Stderr = w
		warnTokenProvenance(tc.prov, "")
		os.Stderr = orig
		_ = w.Close()
		raw, _ := io.ReadAll(r)
		out := string(raw)
		if !strings.Contains(out, tc.want) {
			t.Errorf("provenance %v line missing %q: %q", tc.prov, tc.want, out)
		}
		if strings.Contains(out, "WARNING") {
			t.Errorf("preferred form must not WARN: %q", out)
		}
	}
}

// --- GUARD-DF-005: the policy file surface (-policy / GUARD_POLICY) ---

// scoreClassifier returns a fixed score from every Classify call, so a policy
// change can be observed as a route change without the hosted model.
type scoreClassifier struct {
	score float64
}

func (c scoreClassifier) Name() string { return "score-stub" }
func (c scoreClassifier) Classify(context.Context, string) (guard.Signals, error) {
	return guard.Signals{Injection: c.score, Class: "instruction_injection", Model: "score-stub"}, nil
}

// writePolicyFile is the policy analogue of writeTokenFile.
func writePolicyFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write policy file: %v", err)
	}
	return path
}

const webhookOverridePolicy = `{
  "id": "tight-webhook",
  "block_at": 0.85,
  "review_at": 0.50,
  "quoted_downgrade": 0.80,
  "fail_closed": true,
  "overrides": {
    "webhook": {"block_at": 0.55, "review_at": 0.40}
  }
}`

// AC: no file named anywhere -> the shipped default policy, provenance
// policyDefault.
func TestResolvePolicyDefaultWhenNothingNamed(t *testing.T) {
	pol, prov, err := resolvePolicy("", "")
	if err != nil {
		t.Fatalf("resolvePolicy(empty) = %v", err)
	}
	if prov != policyDefault {
		t.Fatalf("provenance = %v, want policyDefault", prov)
	}
	if pol.ID != "default" || pol.BlockAt != 0.85 || pol.ReviewAt != 0.50 {
		t.Fatalf("policy = %+v, want the shipped default bands", pol)
	}
}

// AC: GUARD_POLICY env loads the file; -policy wins when both are set (the
// ad-hoc override shape, mirroring the token precedence).
func TestResolvePolicyPrecedenceFlagOverEnv(t *testing.T) {
	envPath := writePolicyFile(t, webhookOverridePolicy)
	flagPath := writePolicyFile(t, `{"id":"flag-table","block_at":0.9,"review_at":0.5}`)

	pol, prov, err := resolvePolicy(envPath, "")
	if err != nil || prov != policyFromEnv || pol.ID != "tight-webhook" {
		t.Fatalf("resolvePolicy(env) = (%+v, %v, %v), want env table", pol, prov, err)
	}
	pol, prov, err = resolvePolicy(envPath, flagPath)
	if err != nil || prov != policyFromFlag || pol.ID != "flag-table" {
		t.Fatalf("resolvePolicy(flag) = (%+v, %v, %v), want flag table", pol, prov, err)
	}
}

// AC: a missing file is a startup failure, never a silent fall-back to the
// default bands — an operator tightening a source must not run with the
// looser table while believing the override is live.
func TestResolvePolicyMissingFileRefused(t *testing.T) {
	if _, _, err := resolvePolicy("", filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("resolvePolicy(missing file) = nil error, want refusal")
	}
}

// AC: the loader accepts exactly the documented guard.Policy JSON shape and
// an overrides entry that survives the round trip.
func TestLoadPolicyFileRoundTrip(t *testing.T) {
	path := writePolicyFile(t, webhookOverridePolicy)
	pol, err := loadPolicyFile(path)
	if err != nil {
		t.Fatalf("loadPolicyFile = %v", err)
	}
	o, ok := pol.Overrides["webhook"]
	if !ok {
		t.Fatalf("overrides missing the webhook entry: %+v", pol.Overrides)
	}
	if o.BlockAt != 0.55 || o.ReviewAt != 0.40 {
		t.Fatalf("webhook override = %+v, want block 0.55 / review 0.40", o)
	}
}

// Validation: bands are probabilities, so a table that can never fire —
// out-of-range SET bands, an inverted review>block pair, on the policy itself
// or on any override's effective pair — is refused at load, not discovered in
// production.
func TestLoadPolicyFileValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{"block_at over 1", `{"block_at":1.5,"review_at":0.5}`, "out of range"},
		{"review_at negative", `{"block_at":0.85,"review_at":-1}`, "out of range"},
		{"inverted bands", `{"block_at":0.5,"review_at":0.85}`, "review_at=0.85 > block_at=0.5"},
		{"override inverted", `{"block_at":0.85,"review_at":0.5,"overrides":{"webhook":{"block_at":0.30,"review_at":0.40}}}`, "override webhook"},
		{"override block over 1", `{"block_at":0.85,"review_at":0.5,"overrides":{"webhook":{"block_at":1.7}}}`, "out of range"},
		{"not JSON", `{"block_at":`, "not valid guard.Policy JSON"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadPolicyFile(writePolicyFile(t, tc.body))
			if err == nil {
				t.Fatalf("loadPolicyFile(%s) = nil error, want refusal naming %q", tc.body, tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %q, want it to name %q", err, tc.want)
			}
		})
	}
}

// A minimal file — global bands and quoted_downgrade omitted, one override
// naming only block_at — is VALID: the global fields inherit the shipped
// defaults (a zero block_at would quarantine every score; a zero
// quoted_downgrade would downgrade every quarantine), and an override band ≤0
// means "not set" exactly as Policy.selectFor reads it. This is the README
// "github_pr" example shape.
func TestLoadPolicyFileMinimalShapeInheritsDefaults(t *testing.T) {
	pol, err := loadPolicyFile(writePolicyFile(t, `{"id":"minimal","overrides":{"github_pr":{"block_at":0.75}}}`))
	if err != nil {
		t.Fatalf("loadPolicyFile(minimal) = %v", err)
	}
	def := guard.DefaultPolicy()
	if pol.BlockAt != def.BlockAt || pol.ReviewAt != def.ReviewAt || pol.QuotedDowngrade != def.QuotedDowngrade {
		t.Fatalf("policy = %+v, want inherited defaults %+v", pol, def)
	}

	// And the tightening still works from the minimal file: 0.76 from
	// github_pr clears the override's 0.75 but not the inherited 0.85, so it
	// quarantines — while the same score from another source stays review.
	g := &guard.Guard{Classifier: scoreClassifier{score: 0.76}, Policy: pol}
	if res := g.Check(context.Background(), guard.Input{Source: "github_pr", Content: "x"}); res.Route != guard.RouteQuarantine {
		t.Fatalf("github_pr @0.76 = %s, want quarantine", res.Route)
	}
	if res := g.Check(context.Background(), guard.Input{Source: "inbox", Content: "x"}); res.Route != guard.RouteReview {
		t.Fatalf("inbox @0.76 = %s, want review (inherited default bands)", res.Route)
	}
}

// AC6: a loaded policy with an Overrides entry actually takes effect —
// a webhook-source input scoring 0.60 is QUARANTINED by the tightened
// override while the default table would only review it. This is the
// behavioural proof that REVIEW-GUARD-003's per-source tightening is
// reachable from guardd now, not just from an in-process Go consumer.
func TestLoadedPolicyOverrideTightensWebhookSource(t *testing.T) {
	path := writePolicyFile(t, webhookOverridePolicy)
	pol, err := loadPolicyFile(path)
	if err != nil {
		t.Fatalf("loadPolicyFile = %v", err)
	}
	g := &guard.Guard{Classifier: scoreClassifier{score: 0.60}, Policy: pol}

	res := g.Check(context.Background(), guard.Input{Source: "webhook", Content: "payload"})
	if res.Route != guard.RouteQuarantine || res.Decision != guard.DecisionBlock {
		t.Fatalf("webhook @0.60 with override = %s/%s, want quarantine/block", res.Decision, res.Route)
	}
	if res.RiskLevel != guard.RiskHigh {
		t.Fatalf("risk = %s, want high on the quarantine route", res.RiskLevel)
	}
	if res.Policy != "tight-webhook" {
		t.Fatalf("verdict policy = %q, want the loaded file's id", res.Policy)
	}

	// Same score, other source: the override must NOT leak across sources.
	res = g.Check(context.Background(), guard.Input{Source: "inbox", Content: "payload"})
	if res.Route != guard.RouteReview {
		t.Fatalf("inbox @0.60 with override = %s, want review (default bands)", res.Route)
	}
}

// AC6 through the served surface: a guardd whose policy came from a file
// quarantines the webhook hit and reviews the same score from another
// source, on the live /check handler.
func TestLoadedPolicyOverrideDrivesServedCheck(t *testing.T) {
	pol, err := loadPolicyFile(writePolicyFile(t, webhookOverridePolicy))
	if err != nil {
		t.Fatalf("loadPolicyFile = %v", err)
	}
	srv := httptest.NewServer(newHandler(&guard.Guard{Classifier: scoreClassifier{score: 0.60}, Policy: pol}, "tok", false))
	defer srv.Close()

	post := func(source string) guard.Result {
		t.Helper()
		body := fmt.Sprintf(`{"source":%q,"content":"payload"}`, source)
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/check", strings.NewReader(body))
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		req.Header.Set(OperatorTokenHeader, "tok")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("post /check: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d (body %q)", resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		var res guard.Result
		if err := json.Unmarshal(raw, &res); err != nil {
			t.Fatalf("decode Result: %v (body %q)", err, raw)
		}
		return res
	}

	if res := post("webhook"); res.Route != guard.RouteQuarantine {
		t.Fatalf("served webhook @0.60 = %s, want quarantine from the loaded override", res.Route)
	}
	if res := post("inbox"); res.Route != guard.RouteReview {
		t.Fatalf("served inbox @0.60 = %s, want review (default bands)", res.Route)
	}
}

// A file that leaves quoted_downgrade out inherits the default 0.80: the
// zero value would re-route EVERY quarantine to review (sig.Quoted >= 0 is
// always true), so omission must not read as "disable".
func TestLoadPolicyFileQuotedDowngradeInheritsDefault(t *testing.T) {
	pol, err := loadPolicyFile(writePolicyFile(t, `{"id":"qd","block_at":0.85,"review_at":0.5}`))
	if err != nil {
		t.Fatalf("loadPolicyFile = %v", err)
	}
	if pol.QuotedDowngrade != guard.DefaultPolicy().QuotedDowngrade {
		t.Fatalf("quoted_downgrade = %v, want the shipped %v", pol.QuotedDowngrade, guard.DefaultPolicy().QuotedDowngrade)
	}

	// Behavioural half: with Quoted below the (inherited) threshold, an
	// over-band score stays quarantined; at/above it, quarantine downgrades
	// to review — never to deliver.
	g := &guard.Guard{Classifier: scoreClassifier{score: 0.90}, Policy: pol}
	g.Classifier = quotedClassifier{inner: scoreClassifier{score: 0.90}, quoted: 0.10}
	if res := g.Check(context.Background(), guard.Input{Content: "x"}); res.Route != guard.RouteQuarantine {
		t.Fatalf("quoted 0.10 @0.90 = %s, want quarantine (below the inherited downgrade)", res.Route)
	}
	g.Classifier = quotedClassifier{inner: scoreClassifier{score: 0.90}, quoted: 0.90}
	if res := g.Check(context.Background(), guard.Input{Content: "x"}); res.Route != guard.RouteReview {
		t.Fatalf("quoted 0.90 @0.90 = %s, want review (downgrade fired), got %s", res.Route, res.Route)
	}
}

// quotedClassifier wraps a classifier and stamps a Quoted probability, so the
// downgrade path can be exercised without the hosted model.
type quotedClassifier struct {
	inner  scoreClassifier
	quoted float64
}

func (c quotedClassifier) Name() string { return "quoted-stub" }
func (c quotedClassifier) Classify(ctx context.Context, content string) (guard.Signals, error) {
	sig, err := c.inner.Classify(ctx, content)
	sig.Quoted = c.quoted
	return sig, err
}

// The provenance line lands on stderr with the table id and override count,
// so a mispointed -policy is visible in unit/service logs next to the token
// provenance.
func TestWarnPolicyProvenanceLines(t *testing.T) {
	cases := []struct {
		prov policyProvenance
		pol  guard.Policy
		want string
	}{
		{policyFromFlag, guard.Policy{ID: "tight-webhook", Overrides: map[string]guard.Override{"webhook": {}}}, `"tight-webhook", 1 override(s)`},
		{policyFromEnv, guard.Policy{ID: "env-table"}, GuardPolicyFileEnv},
		{policyDefault, guard.Policy{ID: "default"}, "policy loaded from default"},
	}
	for _, tc := range cases {
		r, w, _ := os.Pipe()
		orig := os.Stderr
		os.Stderr = w
		warnPolicyProvenance(tc.prov, tc.pol)
		os.Stderr = orig
		_ = w.Close()
		out := string(mustReadAll(t, r))
		if !strings.Contains(out, tc.want) {
			t.Errorf("provenance %v output %q missing %q", tc.prov, out, tc.want)
		}
	}
}

func mustReadAll(t *testing.T, r io.Reader) []byte {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return b
}
