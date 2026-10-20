package guard

import (
	"fmt"
	"sort"
	"sync"
)

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
	//
	// Keys are Input.Source values ("github_pr", "inbox", "webhook", ...).
	// They are NOT policy IDs: Overrides{"default": ...} never matches a
	// message, not even under a policy whose ID is "default". A key no source
	// ever sends is inert and silent at this layer — ValidateOverrideKeys
	// checks it up front, and a Guard with Warnf set warns (once) the first
	// time a Check fails to select a key.
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

// failClosedFor is the error-path stance for one source: the policy's
// FailClosed unless that source's Override sets its own. It resolves through
// bandsFor, so the error path and the score path always agree on what an
// override means (GUARD-DF-010: apply() discarded the per-source value and
// Guard.Check keyed on the policy-level switch only).
func (p Policy) failClosedFor(source string) bool {
	_, _, fc := p.bandsFor(source)
	return fc
}

// overrideWatches caches, per policy+key-set, which override keys real traffic
// has already contradicted. Map key is the policy fingerprint (ID + sorted
// key set, sortedOverrides); the value is a set of dead keys already warned
// about. Guards across processes start empty: the first Check that sees
// traffic selects keys, anything still unwarned and unmatched gets exactly
// one warning.
var overrideWatches sync.Map

// sortedOverrides fingerprints a policy's override key set so the warn-once
// cache cannot leak between different policies that share an ID.
func sortedOverrides(p Policy) []string {
	keys := make([]string, 0, len(p.Overrides))
	for k := range p.Overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// unknownOverrideKeys returns the sorted subset of override keys that no
// source in sources selects. sources is the caller's declared source set
// (ValidateOverrideKeys); an empty sources set means every key is unproven.
// The runtime warn-once path does its own live bookkeeping (see
// WarnUnknownOverrideKeys) instead of calling this.
func unknownOverrideKeys(p Policy, sources []string) []string {
	if len(p.Overrides) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(sources))
	for _, s := range sources {
		seen[s] = true
	}
	var unknown []string
	for _, k := range sortedOverrides(p) {
		if !seen[k] {
			unknown = append(unknown, k)
		}
	}
	return unknown
}

// WarnUnknownOverrideKeys checks p's override keys against the sources seen so
// far and warns (via warn, when non-nil) once, naming every key that no
// source has ever selected. Advisory only: it never changes a verdict.
//
// The watch record is the once-guard and it accumulates positively: a key is
// marked seen the moment any call's sources select it (proven live), and
// marked handled the moment a warning names it — so each key is named at most
// once per policy+key-set, no matter how the calls interleave (the claim is
// stored before warn runs). Later calls naming other sources cannot resurrect
// an already-selected key; the unknown set only ever shrinks. Empty Overrides
// is always a no-op.
func WarnUnknownOverrideKeys(p Policy, warn func(format string, args ...any), sources ...string) {
	if len(p.Overrides) == 0 || warn == nil {
		return
	}
	fingerprint := fmt.Sprintf("%s|%v", p.ID, sortedOverrides(p))
	w, _ := overrideWatches.LoadOrStore(fingerprint, &sync.Map{})
	watched := w.(*sync.Map)

	// Prove keys live: this call's sources that actually select an override
	// are permanently good, so later traffic on other sources cannot
	// re-flag them.
	for _, s := range sources {
		if _, isKey := p.Overrides[s]; isKey {
			watched.Store(s, true)
		}
	}

	var unknown []string
	for _, k := range sortedOverrides(p) {
		if _, handled := watched.Load(k); !handled {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) == 0 {
		return
	}
	// Claim-then-warn: mark every dead key handled before invoking warn, so
	// interleaved calls yield exactly one warning per policy+key-set.
	for _, k := range unknown {
		watched.Store(k, true)
	}
	warn("GUARD: policy %q: override key(s) %v not selected by any source seen so far — "+
		"the override is inert (a typo, or the old policy-ID spelling). "+
		"Keys are Input.Source values, never policy IDs. Use ValidateOverrideKeys to fail fast.",
		p.ID, unknown)
}

// ValidateOverrideKeys is the construction-time check: it returns the sorted
// override keys that none of the given sources will ever select. An empty
// result means every key is matched by a declared source. Feed it the full set
// of sources your consumer sends and treat a non-empty result as a config
// error — GUARD-DF-007: an Overrides key matching no Input.Source is inert,
// and this makes the inertness nameable before the first message.
func (p Policy) ValidateOverrideKeys(sources ...string) []string {
	return unknownOverrideKeys(p, sources)
}

// apply fills the verdict fields from the signals. This is the whole decision
// surface: numbers in, route + capabilities out. source selects the per-source
// override (Input.Source); empty means no override.
func (p Policy) apply(res *Result, sig Signals, source string) {
	blockAt, reviewAt, _ := p.bandsFor(source)
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
