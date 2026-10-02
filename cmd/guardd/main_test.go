package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
