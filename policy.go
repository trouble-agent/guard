package guard

// Policy is the deterministic decision table. The classifier produces numbers;
// the policy turns numbers into a route. Keeping this in CODE (not in a prompt)
// is what lets thresholds be tuned, tested and diffed without touching a model.
type Policy struct {
	ID string `json:"id"`

	// BlockAt / ReviewAt are the score bands on max(injection, jailbreak).
	BlockAt  float64 `json:"block_at"`
	ReviewAt float64 `json:"review_at"`

	// QuotedDowngrade: when the classifier is confident the injection is a
	// quotation/example rather than a directive, BLOCK becomes REVIEW. It can
	// only ever loosen to REVIEW — never to allow — so a crafted "this is just
	// a test case" wrapper buys a human look, not a bypass.
	QuotedDowngrade float64 `json:"quoted_downgrade"`

	// FailClosed decides what an ERROR means. true = block on guard error,
	// false = deliver flagged. Never a silent default: the result records which.
	FailClosed bool `json:"fail_closed"`

	// Overrides lets a high-sensitivity source tighten the bands without a new
	// policy. Absent entry = this policy's bands.
	Overrides map[string]Override `json:"overrides,omitempty"`
}

// Override is a per-source tightening.
type Override struct {
	BlockAt    float64 `json:"block_at,omitempty"`
	ReviewAt   float64 `json:"review_at,omitempty"`
	FailClosed *bool   `json:"fail_closed,omitempty"`
}

// DefaultPolicy is the balanced stance: block the clear cases, review the
// ambiguous ones, fail-closed (a guard that cannot judge must not wave through).
func DefaultPolicy() Policy {
	return Policy{
		ID:              "default",
		BlockAt:         0.85,
		ReviewAt:        0.50,
		QuotedDowngrade: 0.80,
		FailClosed:      true,
	}
}

func (p Policy) bandsFor(source string) (blockAt, reviewAt float64, failClosed bool) {
	blockAt, reviewAt, failClosed = p.BlockAt, p.ReviewAt, p.FailClosed
	if o, ok := p.Overrides[source]; ok {
		if o.BlockAt > 0 {
			blockAt = o.BlockAt
		}
		if o.ReviewAt > 0 {
			reviewAt = o.ReviewAt
		}
		if o.FailClosed != nil {
			failClosed = *o.FailClosed
		}
	}
	return
}

// apply fills the verdict fields from the signals. This is the whole decision
// surface: numbers in, route + capabilities out.
func (p Policy) apply(res *Result, sig Signals) {
	blockAt, reviewAt, _ := p.bandsFor(res.Policy)
	score := sig.Score()

	switch {
	case score >= blockAt:
		res.Decision = DecisionBlock
		res.Route = RouteQuarantine
		res.RiskLevel = RiskHigh
		res.Reason = "injection signals above block threshold"
	case score >= reviewAt:
		res.Decision = DecisionAllow
		res.Route = RouteReview
		res.RiskLevel = RiskMedium
		res.Reason = "injection signals in the review band"
	default:
		res.Decision = DecisionAllow
		res.Route = RouteDeliver
		res.RiskLevel = RiskLow
		res.Reason = "no injection signals above threshold"
	}

	// Meta-context downgrade: a high-confidence quotation is a review, not a
	// block. Loosens only to REVIEW — never to deliver.
	if res.Route == RouteQuarantine && sig.Quoted >= p.QuotedDowngrade {
		res.Route = RouteReview
		res.RiskLevel = RiskMedium
		res.Reason = "injection signals above block threshold, but classified as a quotation/example — escalated for review"
	}

	// Capability constraints follow the route. This is what makes a wrong
	// verdict survivable: review means "a reader with no tools and no secrets".
	switch res.Route {
	case RouteDeliver:
		res.Constraints = Constraints{AllowTools: true, AllowNetwork: true, AllowSecrets: false}
	case RouteReview:
		res.Constraints = Constraints{AllowTools: false, AllowNetwork: false, AllowSecrets: false}
	case RouteQuarantine:
		res.Constraints = Constraints{AllowTools: false, AllowNetwork: false, AllowSecrets: false}
	}
}
