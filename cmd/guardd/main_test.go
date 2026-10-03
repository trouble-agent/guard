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
	srv := httptest.NewServer(newHandler(stubGuard(), token))
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
			defer resp.Body.Close()
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
	srv := httptest.NewServer(newHandler(stubGuard(), "s3cr3t-token"))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("get /healthz: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %q)", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if got := strings.TrimSpace(string(raw)); got != "ok" {
		t.Errorf("body = %q, want %q", got, "ok")
	}
}

// A handler that is somehow built without a token must fail closed: an unset
// secret authorizes nothing (startup refuses this case, this is the backstop).
func TestCheckFailsClosedWithoutConfiguredToken(t *testing.T) {
	srv := httptest.NewServer(newHandler(stubGuard(), ""))
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
	defer resp.Body.Close()
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
	defer resp.Body.Close()
	return resp.StatusCode
}

// AC1 (REVIEW-GUARD-001): a token resolved from the GUARD_TOKEN env var
// authenticates /check exactly like the argv form — no argv secret involved.
func TestServeWithTokenFromEnv(t *testing.T) {
	const token = "env-secret-token"
	srv := httptest.NewServer(newHandler(stubGuard(), token))
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
	srv := httptest.NewServer(newHandler(stubGuard(), token))
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
	w.Close()

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
		w.Close()
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
