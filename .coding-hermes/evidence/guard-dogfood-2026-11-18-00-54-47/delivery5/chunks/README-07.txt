
### Classifier egress (`GUARD_EGRESS_ENABLED`)

The hosted Jev classifier is a network call: content handed to `/check` or the
CLI is transmitted to the hosted Jev API outside this host.
`GUARD_EGRESS_ENABLED` is the operator kill switch. Default is **enabled**
(serving with egress enabled prints a one-time warning to that effect). Set it
to `false`, `0`, `no`, or `off` — case-insensitive, surrounding whitespace
ignored; any other value, including unset/empty, means enabled — and guardd
replaces the classifier with a disabled stub: nothing leaves the host. Serve
mode then logs
`WARNING: classifier egress is DISABLED; /check will refuse classification because no local classifier is configured`,
and every `/check` request and CLI classification fails closed with the
standard error verdict — `errored=true`, `route=quarantine`,
`reason="guard_error (fail-closed): hosted classifier egress is disabled (GUARD_EGRESS_ENABLED=false); no local classifier is configured"`
(see "Error verdicts" above). With `-fail-closed=false` the same error routes
to `review` instead of `quarantine`; it is still an error verdict, never a
classification. There is no local classifier: disabling egress always disables
classification.

## Policy is data, in code

```go
guard.Policy{
    ID: "default", BlockAt: 0.85, ReviewAt: 0.50,
    QuotedDowngrade: 0.80,   // BLOCK -> REVIEW when it is a quotation. Never -> deliver.
    FailClosed: true,        // an error blocks; false = route=review, flagged (recorded either way)
    Overrides: map[string]guard.Override{
        "webhook": {BlockAt: 0.55},   // tighten a sensitive source
    },
}
```

A verdict decided by that `webhook` row says so: `override_source: "webhook"`
with `effective_block_at: 0.55` — and an errored verdict carries the same two
fields, since the selected row is what resolved its fail-closed stance.

Override keys are **`Input.Source` values** — `"webhook"`, `"inbox"`,
`"github_pr"`, whatever your callers send in the `source` field. They are never
policy IDs: `Overrides{"default": ...}` matches no message even under a policy
whose ID is `"default"`. A key no source ever selects is inert, and the library
will not invent a signal for it at verdict time. Two ways to catch a typo
(`"webook"`) before it costs you a tightening:

```go
// Construction time: name every source your consumer sends; a non-empty
// result is a config error.
if dead := policy.ValidateOverrideKeys("webhook", "inbox", "github_pr"); len(dead) > 0 {
    return fmt.Errorf("policy %s: override keys %v match no source", policy.ID, dead)
}

// Runtime: a Guard with Warnf set warns once, naming the dead keys, the
// first time a Check fails to select a key.
g := &guard.Guard{Classifier: cls, Policy: policy, Warnf: log.Printf}
```

## Extending it (the point of the design)

| Layer | Interface | Add today |
|---|---|---|
| Normalisers | `Normalizer.Expand(string) ([]string, bool)` | a new decoder; chain order is the only config |
| Classifiers | `Classifier.Classify(ctx, string) (Signals, error)` | a local model, a rules engine, a second guard for consensus |
| Routes | add a `Route` constant | a new destination (e.g. `sandbox`) |
| Policies | `Policy` + `Override` | per-source bands without a new code path |

Nothing in the pipeline requires touching another layer to grow.

## Honest limitations

- **The classifier is a hosted third-party API.** Untrusted content sent for
  classification leaves the host. Redact before guarding, or guard only content
  that is already safe to share.
- **Availability is a policy decision, not a default.** `FailClosed: true` means
  a provider outage stops delivery; `false` means delivery continues flagged. Pick
  per surface.
- **Long content is rejected above the classification window.** The guard
  refuses inputs over 20,000 UTF-8 bytes (or normalized state over that budget)
  before provider submission. Raw overflow reports raw `bytes_seen`; normalized
  overflow reports the complete normalized-state length in `bytes_seen`, with
