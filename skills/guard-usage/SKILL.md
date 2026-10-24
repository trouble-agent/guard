---
name: guard-usage
description: Use when integrating or running the guard message-security filter (CLI, guardd HTTP service, or Python connector) — entry points, prerequisites, and the pitfalls that cost time.
---

# guard — usage for agents

**What it is:** a lightweight prompt-injection filter for untrusted text.
One Go core, three consumption shapes, one `Result` contract. Route on
`result.route` + `result.constraints`, never on `decision` alone.

## Prerequisites

- Go ≥1.26.9 to build (`go build ./...` — go.mod declares `go 1.26.9`); zero external deps.
  A newer toolchain downloads itself from the `go` directive's toolchain floor.
- An OpenRouter API key for ANY classification (Jev decisions model at
  `openrouter.ai/api/alpha/decisions`). Key load order:
  `$OPENROUTER_API_KEY` → `~/.hermes/.env` → `~/9router-deploy/.env.shared`
  (regex `sk-or-v1-…`, tried in order on 401/402/429). An expired key in the
  first file yields opaque `guard_error (fail-closed): key rejected: http 401`
  block verdicts — check `errored` in the Result when a block looks wrong.
- **`jq` for `make docs-check`** (the README-drift gate probes for it at
  `Makefile:31-32` and shells out to it at `Makefile:86,88,91,104`). The gate
  checks for the tool before parsing: without jq, `make docs-check` exits 2
  with `docs-check FAIL: jq was not found on PATH`; with jq present, it ends
  with `docs-check: PASS` (GUARD-DF-004).
- Full prerequisite detail, key discovery, and a first-run walkthrough:
  README.md "Quick Start". This file is the agent-facing summary.

## Entry points

```bash
go build -o bin/guardd ./cmd/guardd
printf '%s' "$msg" | ./bin/guardd                    # CLI, stdin
./bin/guardd -content "$msg" -source github_pr       # CLI, one-shot
# -channel <label> — currently an inert annotation: it is passed to
# Input.Channel and no non-test code reads it, so it does not reach the
# classifier or the Result (GUARD-DF-008d). Do not rely on it.
./bin/guardd -content "$msg" -source task-router -channel router
./bin/guardd -serve 127.0.0.1:8768 -tokenfile <0600-file>  # HTTP (preferred; or GUARD_TOKEN env)
./bin/guardd -serve 127.0.0.1:8768 -token <secret>         # dev: works but warns (token visible in ps)
```

```python
import sys; sys.path.insert(0, 'connectors/python')
from guard_client import Guard
g = Guard(endpoint='http://127.0.0.1:8768', fail_mode='open')
v = g.check(content, source='task-router', channel='router')
if v.errored: ...        # outage path — check BEFORE honouring constraints (GUARD-DF-001)
if not v.deliver: ...    # quarantine/review
```

## Policy and per-source overrides — read this before promising a tightening

- `Policy` is a Go struct (`policy.go`): `BlockAt`, `ReviewAt`,
  `QuotedDowngrade`, `FailClosed`, `Overrides map[string]Override`.
- `Overrides` is keyed on **`Input.Source`** — never on the policy id. A key
  that matches no source the caller passes is **silently ignored**: no error,
  no warning, the base bands simply apply (GUARD-DF-007). A typo'd source label
  (`"webook"`) disables the tightening invisibly.
- **The served/CLI shapes CAN supply a Policy** (GUARD-DF-005 landed):
  `-policy <file>` or `GUARD_POLICY=<file>` loads a policy file (JSON matching
  `guard.Policy`), validated strictly at startup — bands in (0,1],
  `review_at <= block_at`, omitted global fields inheriting the shipped
  defaults — and guardd refuses to run on a bad file instead of falling back to
  looser bands (`cmd/guardd/main.go`, `resolvePolicy`/`loadPolicyFile`). With
  no file named, guardd runs `guard.DefaultPolicy()`. An in-process Go consumer
  (crier) still builds its own `&guard.Guard{...}`.
- **A verdict reports which row of the table decided it** (GUARD-DF-011):
  `override_source` names the `Overrides` key the message selected (absent =
  the policy's own bands decided) and `effective_block_at` /
  `effective_review_at` are the bands that were actually compared — on error
  verdicts too, since the selected row is what resolves the fail-closed stance.
  That is how you verify a tightening fired from the audit trail instead of
  inferring it from the policy file.
- `QuotedDowngrade` (default 0.80) is live and verified: a score above
  `BlockAt` on text the classifier reads as a quotation/example becomes
  `route=review`, never deliver and never with capabilities restored. Three
  independent phrasings (explicit "DO NOT FOLLOW THIS" wrapper, a fenced test
  case, a canary string) scored 0.95–0.97 and all downgraded to review.

## Verdict semantics that matter

- `route`: deliver | review | quarantine — the primary output.
- `constraints`: allow_tools/network/secrets for the handling agent.
- `policy` / `override_source` / `effective_block_at` / `effective_review_at`:
  the table that judged the message, the `Overrides` row INSIDE it that decided
  this verdict (absent when the policy's own bands did), and the bands actually
  compared (GUARD-DF-011). Two verdicts produced under different bands are
  distinguishable from the JSON alone — a tightened source no longer looks like
  the base bands.
- `errored: true` ⇒ every other field describes the ERROR verdict, not a
  classification (attack_class is absent, score is 0). `policy`,
  `override_source` and the `effective_*` bands are still meaningful: they name
  the row that resolved the fail-closed/fail-open stance.
- `normalizations` includes `rot13` on ~every result: ROT13 is applied
  unconditionally by design; the field means "transform ran", NOT "this
  payload was encoded". "Caught by an encoding" reads from
  `load_bearing_normalizations` (populated only when the decoded variant
  changed the verdict — GUARD-DF-002 fix).
- `attack_class` is noisy near the threshold — benign text often carries
  masquerade at 0.1–0.4. Trust `route`, not the label, at the margin.
- **`duration_ms` and `cost_usd` cover ALL classifier calls the verdict
  caused** (accumulated accounting since e858041, the GUARD-DF-006 fix). On a
  non-deliver verdict the guard re-classifies the raw input to attribute
  normalisations (skipped when the verdict errored or normalisation changed
  nothing), so a caught message costs **two** classifier calls, about twice
  the wall time, and reports ≈2x the cost — the fields are the full bill for
  the message, not the first call. Measured: caught verdicts ~0.48–0.85 s wall
  vs deliver ~0.24–0.40 s (load-dependent; 47 samples over 2026-10-23/24 at
  karaHermes-mde-7840hs, HEADs c9f913b–06085ab). Don't read the fields as per-call
  values: to time or count an individual call, measure it yourself or use a
  stub endpoint.

## Run commands

```bash
go build ./... && go vet ./... && go test ./... -count=1   # about 4–6s real (4.04s on 2026-10-24 at karaHermes-mde-7840hs, HEAD 74f9d1b)
# Offline when testing.Short, no resolved classifier key, or HTTP_PROXY/HTTPS_PROXY is set (the live tests skip).
# Keyed runs make six live Jev classifier calls in TestLiveJevRoutesKnownCases (real network/cost).
make docs-check                                            # needs jq (see Prerequisites)
```

## Performance (measured 2026-10-23–24, HEADs c9f913b–06085ab, karaHermes-mde-7840hs)

- deliver verdict, HTTP /check warm: **0.24–0.40 s** wall, `duration_ms` 240–389
  (47 samples across 2026-10-23/24 under mixed host load; five-run sets land
  240–294 ms on an idle host and 299–389 ms on a loaded one — the spread is
  host load, not code).
- caught verdict (block/review): **0.48–0.85 s** wall, `duration_ms` 477–807
  (13 samples 2026-10-23/24: 576–603 ms measured 10-23, 477–807 ms measured
  10-24) — two classifier calls by design.
- local floor with `GUARD_EGRESS_ENABLED=false`: 0.00–0.01 s, rc=2.
- cold first call ~0.85 s; cost ≈ $0.000027 per classifier call (measured
  0.0000268–0.0000271) — a caught message's `cost_usd` reports both calls
  (≈2x) since e858041.

≈ All of it is the Jev API round trip (user+sys ≈ 20–25 ms locally).
Don't profile the Go code first — the network is the latency.

## Pitfalls

- HTTP /check without `X-Operator-Token` → 401 by design; /healthz is open and
  returns `{"status":"ok","egress_enabled":<bool>}`.
- The `cmd/guardd/main.go` file-header usage comment says `POST /check {input}`;
  the handler decodes `{content}`. Following the comment gives
  `400 content is required` with no hint of the right key (GUARD-DF-008a).
- Fail-open (`fail_mode='open'`) does NOT deliver on outage: it returns
  route=review, decision='error', errored=True, and
  constraints=CONSTRAINTS_FAIL_OPEN (allow_tools/allow_secrets still granted
  — GUARD-DF-001). The outage degrades to "a human looks" without silently
  stripping tools/secrets from every task; `errored` ALWAYS gates unattended
  delivery, so branch on `errored` and never auto-deliver, whatever
  constraints say. (`fail_mode='closed'` raises instead.) The Go client's
  `FailOpen` takes the opposite stance: route=review, errored=true, **every
  capability withheld** — both are documented, do not "reconcile" them.
- The Python kwarg is `endpoint=`, not `base_url=`; default endpoint from
  env `GUARD_ENDPOINT`, else `http://127.0.0.1:8768`. The module default
  **snapshots `GUARD_ENDPOINT` at import time** — set the env before importing,
  or pass `endpoint=` explicitly.
- Malformed JSON to /check → HTTP 400 with a Go json parse message (readable).
- A body over 8 MiB is truncated by `io.LimitReader` and surfaces as
  `400 bad request: unexpected EOF` — the 8 MiB limit is not named in the
  error (GUARD-DF-008c). The CLI's stdin path (`io.ReadAll(os.Stdin)`,
  `main.go:123`) has NO cap at all (GUARD-DF-008b).
- go.mod declares go 1.26.9; newer toolchains are fine.
