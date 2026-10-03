package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/trouble-agent/guard"
)

func TestCheckAgainstStub(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/check" {
			http.NotFound(w, r)
			return
		}
		var in guard.Input
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(guard.Result{
			Decision:  guard.DecisionBlock,
			RiskLevel: guard.RiskHigh,
			Route:     guard.RouteQuarantine,
			Policy:    "default",
			Provider:  "jev",
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	res, err := c.Check(context.Background(), guard.Input{Source: "webhook", Content: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Route != guard.RouteQuarantine || res.Decision != guard.DecisionBlock {
		t.Fatalf("got %s/%s", res.Route, res.Decision)
	}
}

func TestFailClosedReturnsError(t *testing.T) {
	c := New("http://127.0.0.1:1") // nothing listening
	c.Timeout = 2_000_000_000      // 2s
	if _, err := c.Check(context.Background(), guard.Input{Content: "x"}); err == nil {
		t.Fatal("FailClosed must surface the transport error")
	}
}

func TestFailOpenReturnsReviewWithNoCapabilities(t *testing.T) {
	c := New("http://127.0.0.1:1")
	c.FailMode = FailOpen
	res, err := c.Check(context.Background(), guard.Input{Content: "x"})
	if err != nil {
		t.Fatalf("FailOpen must not return an error: %v", err)
	}
	if res.Route != guard.RouteReview {
		t.Fatalf("FailOpen must degrade to review, got %s", res.Route)
	}
	if !res.Errored {
		t.Fatal("the failure must be recorded")
	}
	if res.Constraints.AllowTools || res.Constraints.AllowSecrets {
		t.Fatal("a failed classification must grant nothing")
	}
}

// tokenStub mirrors guardd's /check: 401 unless X-Operator-Token matches.
func newTokenServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/check" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("X-Operator-Token") != token {
			http.Error(w, "unauthorized: missing or invalid X-Operator-Token", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(guard.Result{
			Decision: guard.DecisionAllow,
			Route:    guard.RouteDeliver,
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// Without a token the request must hit guardd's 401 — the defect this row
// fixes was every client call failing exactly this way.
func TestCheckWithoutTokenFails401(t *testing.T) {
	t.Setenv("GUARD_TOKEN", "") // the ambient env must not satisfy the stub
	srv := newTokenServer(t, "s3cret")
	c := New(srv.URL)
	_, err := c.Check(context.Background(), guard.Input{Content: "x"})
	if err == nil {
		t.Fatal("a token-gated guardd must reject a headerless client")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected the server's 401 in the error, got: %v", err)
	}
}

func TestCheckWithTokenFieldSendsHeader(t *testing.T) {
	srv := newTokenServer(t, "s3cret")
	c := New(srv.URL)
	c.Token = "s3cret"
	res, err := c.Check(context.Background(), guard.Input{Content: "x"})
	if err != nil {
		t.Fatalf("the token must satisfy guardd's /check: %v", err)
	}
	if res.Route != guard.RouteDeliver {
		t.Fatalf("got %s", res.Route)
	}
}

func TestCheckWithTokenEnvFallback(t *testing.T) {
	t.Setenv("GUARD_TOKEN", "env-secret")
	srv := newTokenServer(t, "env-secret")
	c := New(srv.URL)
	res, err := c.Check(context.Background(), guard.Input{Content: "x"})
	if err != nil {
		t.Fatalf("GUARD_TOKEN env fallback must send the header: %v", err)
	}
	if res.Route != guard.RouteDeliver {
		t.Fatalf("got %s", res.Route)
	}
}
