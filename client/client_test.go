package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		json.NewEncoder(w).Encode(guard.Result{
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
