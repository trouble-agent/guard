# DOC-9 — documentation parity audit (2026-10-13)

Re-audit after DOC-6 (exit codes), DOC-7 (SKILL.md refresh), DOC-8 (GUARD_EGRESS_ENABLED),
DOC-2 (/healthz + anchors) — wave guard-foreman-2026-10-13-15-29-04.
Verdict: **PASS**, with one anchor regression found and fixed in this commit.

## 1. Source inventory (DOC-6)

`grep -n 'Exit code 3\|exits 0' cmd/guardd/main.go` → one match (line 31): the
corrective statement itself — "Exit code 3 is not a guardd code at all: it belongs to
the Python connector (…0 deliver, 1 review, 2 quarantine, 3 no-verdict)". The header
comment now states: exit 0 = any verdict served (block included), exit 2 = guard_error
(no key / egress disabled / classifier failure), `-fail-closed=false` reshapes the
JSON only (exit stays 2), HTTP /check unaffected. No stale claims.

## 2. No-key fail-closed probe (rebuilt binary)

- `go build -o /tmp/guardd-audit ./cmd/guardd` — OK; `--help` renders all flags.
- `printf hello | env -u OPENROUTER_API_KEY HOME=/tmp/guard-empty guardd` →
  **rc=2**, JSON: `decision=block, risk_level=medium, route=quarantine,
  reason="guard_error (fail-closed): no openrouter api key available",
  errored=true, constraints all false, normalizations=[rot13]`. Exact match.

## 3. Live HTTP probes (`-serve :18769 -token test123`)

| Probe | Result |
|---|---|
| GET /healthz | 200 `{"status":"ok","egress_enabled":true}` |
| POST /check + token | 200 verdict: allow/deliver, provider jev-1.13, score 0.17 |
| POST /check, no token | 401 `unauthorized: missing or invalid X-Operator-Token` |
| GET /check + token | 405 `POST only` |
| bad JSON body | 400 `bad request: invalid character 'b' …` |
| empty content | 400 `content is required` |

Field-name note: the task brief expected a `classifier_egress` healthz field; the
server and every doc (ADOPTION.md:201, connectors/README.md:65) agree on
`egress_enabled`. The brief's field name was the slip — parity holds.

## 4. Connector test suites

- `connectors/python`: pytest 21/21 passed (6.06s), including the outage
  exit-code branch tests (exit 3 fail-closed outage, exit 1 fail-open outage).
- `go test ./client/...`: 6/6 PASS (0.286s).

## 5. Audience coverage

- `GUARD_EGRESS_ENABLED`: README.md §"Classifier egress" (lines 203–216, default
  enabled, kill switch, exact guard_error reason string), connectors/README.md:67,
  docs/ADOPTION.md:161–204. Absent from skills/guard-usage/SKILL.md (operator env
  knob, not an agent-facing capability) — listed under residual gaps.
- Go ≥1.26.6: skills/guard-usage/SKILL.md:14 and README.md:12.
- `errored`: README.md ×6 (incl. the "DIFFERENT shape on classifier failure"
  contract) and SKILL.md ×6 (GUARD-DF-001 gating).

## 6. Stale-reference scan

- `Go 1.24|1.25`: one hit — docs/dogfood/2026-10-02-integration.md:85, a *dated
  dogfood finding* ("Go ≥1.24 … neither is stated") that predates this wave and
  motivated the fixes; it is the audit record, not a living claim. Same class:
  main.go anchors inside docs/dogfood/2026-10-13-run2-findings.md are same-day
  findings. No living doc carries stale references.
- `constraints={}` in SKILL.md: none.

## REGRESSION FOUND AND FIXED: connectors/README.md line anchors (+2)

DOC-6's exit-code comment (commit 8ef7ffe) added two header lines to
cmd/guardd/main.go **after** DOC-2 re-verified the anchors, shifting every
`main.go:` line reference in connectors/README.md by +2. Stale → fixed, each
re-verified against live main.go:

| Ref | Was | Now (verified) |
|---|---|---|
| 401 unauthorized | 183 | 185 |
| 405 POST only | 187 | 189 |
| 400 bad request | 192 | 194 |
| 400 content required | 196 | 198 |
| LimitReader 8 MiB | 191 | 193 |
| HTTP classification timeout | 199 | 201 |
| CLI timeout budget | 128 | 130 |
| /healthz handler | 172-178 | 174-180 |

All documented error bodies/states live-probed this tick (table above).
Provenance note: this connectors/README.md edit was found **uncommitted in the
worktree** at tick start (author unknown); every corrected value was
independently re-derived from main.go before adopting it.

## Residual gaps (report only, not fixed here)

1. **No anchor-drift guard**: `make docs-check` verifies the go directive and
   README command replay, but greps no `main.go:N` references — DOC-2-style
   drift recurs on any main.go edit. Suggest extending docs-check to assert
   every `path:N` anchor in README/connectors/README/docs resolves.
2. `GUARD_EGRESS_ENABLED` not in skills/guard-usage/SKILL.md (operator knob).
3. Brief-vs-reality field name: `classifier_egress` (brief) vs
   `egress_enabled` (docs + server) — brief slip only.

ch:trace row=DOC-9 spec=guard-docs-parity-audit evidence=probes-green+tests-pass+no-stale-refs witness=none:local-verification verdict=none:docs-only commit=HEAD
