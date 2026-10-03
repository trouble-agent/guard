// Package client is the thin HTTP connector for the guard service.
//
// This is the connector for Go consumers that talk to a shared guardd instance
// (crier, hermes-dagger). For in-process use with no network hop, import the
// parent package github.com/trouble-agent/guard directly instead — same Result
// shape either way, so the two are interchangeable.
//
// The whole point is that adopting this is one line, the way using a
// parameterised query is one line:
//
//	c := client.New("http://127.0.0.1:8768")
//	c.Token = os.Getenv("GUARD_TOKEN") // /check requires X-Operator-Token
//	res, err := c.Check(ctx, guard.Input{Source: "webhook", Content: raw})
//	if res.Route != guard.RouteDeliver {
//		// honour res.Route and res.Constraints — do not deliver as-is
//	}
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/trouble-agent/guard"
)

// Client calls a guardd HTTP service.
type Client struct {
	// Endpoint is the base URL of the service, e.g. http://127.0.0.1:8768.
	Endpoint string
	// Token is the shared secret sent as the X-Operator-Token header on
	// /check (guardd's operator-token auth). When empty, GUARD_TOKEN from
	// the environment is used; guardd answers 401 to /check without it.
	Token string
	// Timeout bounds one classification. Default 120s (Jev can be slow).
	Timeout time.Duration
	// FailMode says what a transport/HTTP error means to the CALLER. It mirrors
	// the server's policy so the decision is never implicit:
	//   FailClosed (default) -> return the error; the caller must not deliver.
	//   FailOpen             -> synthesise a REVIEW verdict and return it.
	FailMode FailMode
	HTTP     *http.Client
}

// FailMode is the connector-side error stance.
type FailMode int

const (
	// FailClosed returns the error. The caller decides; nothing is delivered.
	FailClosed FailMode = iota
	// FailOpen returns a REVIEW verdict with all capabilities withheld, so a
	// service outage degrades to "a human looks" rather than "no protection"
	// or "work stops".
	FailOpen
)

// New returns a client for endpoint with the safe defaults.
func New(endpoint string) *Client {
	return &Client{Endpoint: endpoint, Timeout: 120 * time.Second, FailMode: FailClosed}
}

// token resolves the operator token: the Token field, else GUARD_TOKEN from
// the environment. An empty result means no header is sent (a token-gated
// guardd then answers 401, which is the fail-closed outcome).
func (c *Client) token() string {
	if c.Token != "" {
		return c.Token
	}
	return os.Getenv("GUARD_TOKEN")
}

// Check classifies one message. On error it honours FailMode: FailClosed
// returns the error, FailOpen returns a REVIEW verdict that withholds every
// capability. It never returns a zero Result with a nil error.
func (c *Client) Check(ctx context.Context, in guard.Input) (guard.Result, error) {
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 120 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	body, err := json.Marshal(in)
	if err != nil {
		return c.onError(fmt.Errorf("encode input: %w", err))
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "http://127.0.0.1:8768"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/check", bytes.NewReader(body))
	if err != nil {
		return c.onError(fmt.Errorf("build request: %w", err))
	}
	req.Header.Set("Content-Type", "application/json")
	if token := c.token(); token != "" {
		req.Header.Set("X-Operator-Token", token)
	}

	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return c.onError(fmt.Errorf("guard unreachable: %w", err))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return c.onError(fmt.Errorf("guard http %d: %s", resp.StatusCode, string(bytes.TrimSpace(b))))
	}
	var res guard.Result
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return c.onError(fmt.Errorf("decode verdict: %w", err))
	}
	return res, nil
}

func (c *Client) onError(err error) (guard.Result, error) {
	if c.FailMode == FailOpen {
		return guard.Result{
			Decision:    guard.DecisionAllow,
			RiskLevel:   guard.RiskMedium,
			Route:       guard.RouteReview,
			Reason:      "guard_error (connector fail-open): " + err.Error(),
			Errored:     true,
			Constraints: guard.Constraints{}, // nothing granted
		}, nil
	}
	return guard.Result{}, err
}
