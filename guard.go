// Package guard is a lightweight, language-neutral message-security filter.
//
// One core, many callers: crier (Go, in-process), task-router (Python, over
// HTTP), or anything else. The contract is deliberately the SAME shape crier's
// internal/guard already speaks (Decision allow|block|sanitize, RiskLevel
// low|medium|high), so adopting this is not a second dialect.
//
// Design notes (each one is load-bearing):
//
//   - Stdlib only. No dependencies, no cgo: a static binary small enough to
//     drop anywhere, including where nothing else can be installed.
//   - Deterministic normalisation runs BEFORE the classifier. A payload the
//     classifier cannot internally invert (ROT13 was the measured miss) is
//     decoded here instead. The model judges plaintext; the parser does parsing.
//   - Fail-closed is a per-policy switch, never a silent default. A guard that
//     cannot produce a verdict either (a) blocks, or (b) delivers-flagged, and
//     it records which choice it made.
//   - The verdict carries a ROUTE, not just an allow/deny. The guard's job is to
//     tell the caller WHERE the message should go (deliver | review |
//     quarantine) and what CAPABILITIES the handling agent may hold. A verdict
//     that only picks a destination can afford to be wrong more often than one
//     that gates content — which is why the route is the primary output.
package guard

import (
	"context"
	"time"
)

// Decision is the verdict action (crier contract).
type Decision string

const (
	DecisionAllow    Decision = "allow"
	DecisionBlock    Decision = "block"
	DecisionSanitize Decision = "sanitize"
)

// RiskLevel is the verdict risk tier (crier contract).
type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

// Route is the message-routing surface: where the caller should send the
// message, independent of the raw decision. This is the part integrations bind
// to, so it is stable and extensible (add a route, not a flag).
type Route string

const (
	RouteDeliver    Route = "deliver"    // straight through to the normal handler
	RouteReview     Route = "review"     // a human/strong model looks before anyone acts
	RouteQuarantine Route = "quarantine" // held: never delivered as-is
)

// Constraints say what capabilities the agent handling this message may hold.
// This is what makes a wrong verdict survivable: a false negative gets a
// tool-less reader instead of a breach.
type Constraints struct {
	AllowTools   bool `json:"allow_tools"`
	AllowNetwork bool `json:"allow_network"`
	AllowSecrets bool `json:"allow_secrets"`
}

// Input is one message to classify. Content is untrusted.
type Input struct {
	ID      string            `json:"id,omitempty"`
	Source  string            `json:"source,omitempty"`  // github_pr | inbox | webhook | task | ...
	Channel string            `json:"channel,omitempty"` // crier | task-router | ...
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// Result is the full guard outcome for one message. Field names and value
// spaces match crier's Result so one consumer shape serves every integration.
type Result struct {
	Decision    Decision    `json:"decision"`
	RiskLevel   RiskLevel   `json:"risk_level"`
	Route       Route       `json:"route"`
	Reason      string      `json:"reason,omitempty"`
	Patterns    []string    `json:"matched_patterns,omitempty"`
	Policy      string      `json:"policy,omitempty"`
	Provider    string      `json:"provider,omitempty"`
	Model       string      `json:"model,omitempty"`
	AttackClass string      `json:"attack_class,omitempty"`
	Score       float64     `json:"score"`
	Errored     bool        `json:"errored,omitempty"`
	Constraints Constraints `json:"constraints"`
	DurationMs  int64       `json:"duration_ms,omitempty"`
	CostUSD     float64     `json:"cost_usd,omitempty"`
	// Normalizations lists the deterministic transforms applied before
	// classification (e.g. "base64", "rot13"). Visible so a caller can see
	// WHY a decode-and-follow payload was caught.
	Normalizations []string `json:"normalizations,omitempty"`
}

// Signals is what a classifier returns: calibrated numbers, not prose.
type Signals struct {
	Injection float64 // P(content tries to manipulate instructions/secrets)
	Jailbreak float64 // P(content tries to remove safety restrictions)
	Severity  float64 // ordered rubric position 0..N
	Class     string  // instruction_injection | jailbreak | masquerade | structured_object | none
	Quoted    float64 // P(the injection appears as a quotation/example, not a directive)
	Sample    string  // representative excerpt (never the whole payload)
	Model     string
	CostUSD   float64
}

// Score collapses the two detectors: either firing is enough.
func (s Signals) Score() float64 {
	if s.Jailbreak > s.Injection {
		return s.Jailbreak
	}
	return s.Injection
}

// Classifier is the pluggable judgement layer. Jev today; a chat-LLM, a local
// model, or a deterministic ruleset tomorrow — the interface is the expansion
// point, not the implementation.
type Classifier interface {
	Name() string
	Classify(ctx context.Context, content string) (Signals, error)
}

// Guard wires normalizers, a classifier and a policy together.
type Guard struct {
	Classifier  Classifier
	Policy      Policy
	Normalizers []Normalizer
}

// Check classifies one message. It never returns an error: an error becomes a
// verdict that records the failure, so callers have exactly one code path.
func (g *Guard) Check(ctx context.Context, in Input) Result {
	start := time.Now()
	res := Result{Provider: g.Classifier.Name(), Policy: g.Policy.ID, Constraints: Constraints{}}

	text, applied := NormalizeAll(in.Content, g.Normalizers)
	res.Normalizations = applied

	sig, err := g.Classifier.Classify(ctx, text)
	res.DurationMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Errored = true
		res.RiskLevel = RiskMedium
		if g.Policy.FailClosed {
			res.Decision = DecisionBlock
			res.Route = RouteQuarantine
			res.Reason = "guard_error (fail-closed): " + err.Error()
		} else {
			res.Decision = DecisionAllow
			res.Route = RouteReview
			res.Reason = "guard_error (fail-open): " + err.Error()
		}
		return res
	}

	res.Score = sig.Score()
	res.Model = sig.Model
	res.CostUSD = sig.CostUSD
	if sig.Class != "" && sig.Class != "none" {
		res.AttackClass = sig.Class
		res.Patterns = []string{sig.Class}
	}
	g.Policy.apply(&res, sig)
	return res
}
