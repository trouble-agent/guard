  "route": "quarantine",            // deliver | review | quarantine
  "errored": false,                 // json:"errored,omitempty": emitted only
                                    // when true; an absent key means false
                                    // (see "Error verdicts" below)
  "reason": "injection signals above block threshold",
  "matched_patterns": ["instruction_injection"],
  "attack_class": "instruction_injection",
  "score": 0.98,
  "policy": "default",              // the decision table that judged this message
  "override_source": "webhook",     // the Policy.Overrides row that decided it — ABSENT
                                    // when the policy's own bands did (GUARD-DF-011)
  "effective_block_at": 0.55,       // the bands actually compared: the policy's own, or
  "effective_review_at": 0.40,      // the selected Override's (never omitted)
  "provider": "jev",
  "model": "typesafe/jev-1.13-20260917",
  "constraints": {                  // what the handling agent may hold
    "allow_tools": false,
    "allow_network": false,
    "allow_secrets": false
  },
  "normalizations": ["rot13"],               // what RAN before classification (rot13 always)
  "load_bearing_normalizations": ["rot13"],  // the decoded variant DROVE the verdict (absent otherwise)
  "duration_ms": 439,
  "cost_usd": 0.000027468,
  "bytes_seen": 1234,
  "classified_bytes": 1234,
  "truncated": false
}
```

**`route` is the primary output, not `decision`.** A verdict that only chooses a
destination can afford to be wrong far more often than one that gates content:
a false negative gets a reader holding no tools and no secrets, a false positive
gets a slightly dumber reader. That is what makes the filter safe to deploy at
90% accuracy instead of 99%.

### Which row of the table decided (`override_source`, `effective_*`)

`policy` names the table; it does not say which row of that table produced this
verdict. Once a per-source Override is in play the two can diverge: a tightened
`webhook` and an unoverridden source route the same score differently while
`policy`, `decision`, `risk_level` and the `reason` wording all stay identical.
So the verdict also reports the SELECTION that was applied:

- `override_source` — the `Policy.Overrides` key this message selected (the
  `source` the caller sent). Absent when the policy's own bands decided, so an
  old consumer never sees the key and a present key always means "an Override
  row decided this".
- `effective_block_at` / `effective_review_at` — the bands that were actually
  compared: the policy's own, or the selected Override's. Never omitted, so an
  absent key can never be misread as a zero band.

Two verdicts produced under different bands are therefore distinguishable from
the JSON alone: the 0.60 webhook message that quarantined carries
`override_source: "webhook"` with `effective_block_at: 0.55`, while a control
source on the base bands carries neither key and reports 0.85/0.50. Both fields
appear on error verdicts too — the selected row is what resolved the fail-closed
stance — and the Python connector mirrors them (`Verdict.override_source`,
`Verdict.effective_block_at`, `Verdict.effective_review_at`).

### normalizations: what ran vs what mattered

`normalizations` lists the deterministic transforms that ran before
classification. ROT13 runs on **every** message by design — it is the measured
fix for the bare-ROT13 payload that once scored 0.32 and was delivered — so
its presence says nothing about your payload. Read
`load_bearing_normalizations` for the audit answer: on a non-deliver verdict
the guard re-classifies the raw input, and names the decode layer there only
if the verdict differs without the decoded text. A caught ROT13 payload
reports both fields; benign plain text reports
`normalizations: ["rot13"]` (the transform ran) and no load-bearing
normalizations. Attribution costs one extra classifier call on caught
messages only — deliver verdicts skip it.

