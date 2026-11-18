# Dogfood run 9, tick 1 — 2026-11-18 (guard-dogfood-2026-11-18-00-54-47)

Verdict: SHIPPABLE. One docs-census row filed (`GUARD-DF-023`); no other finding.

## Promise

"A user can filter messages for injection risk through one core from three
consumption shapes — the Go library, the guardd CLI, and guardd over HTTP
(with a Python connector) — following README Quick Start alone."

## What was actually used this fire

Every step below ran from the repo at `7c08590` (= origin/main at fire time).

1. **Build.** `make build` → `bin/guardd v0.2.0` in 0.47 s. `-version` prints
   the stamped build version (injected via Makefile ldflags).
2. **Fail-closed CLI, documented reproduction.** The README's scratch-HOME
   command (`env -u OPENROUTER_API_KEY HOME=$TD … ./bin/guardd`) reproduced
   the exact documented verdict — `decision: block, route: quarantine,
   errored: true`, reason naming the missing key — and exited **rc=2**, which
   is what the README promises for "config broken". Cold wall time < 10 ms.
3. **Real classification (the leg prior fires left UNVERIFIED).** With the
   documented key-discovery order intact (`~/.hermes/.env`), stdin classify
   returned a real Jev verdict: `allow`, score 0.2, `attack_class:
   masquerade`, `model: typesafe/jev-1.13-20260917`, `duration_ms: 720`,
   `cost_usd: 0.0000278`. Warm CLI classify over 5 runs: 0.40–0.58 s wall
   (dominated by the classifier round-trip, i.e. the network, not guard).
4. **Serve mode + policy overrides (documented Quick Start fixture).**
   `GUARD_TOKEN … GUARD_POLICY=policy.json ./bin/guardd -serve :8791`; the
   per-source override moved the bands live: `source: webhook` verdicts carry
   `override_source: "webhook"`, `effective_block_at: 0.55`,
   `effective_review_at: 0.4`, while `github_pr` falls to the base
   0.85/0.50. Both `/healthz` and the `/health` alias answer 200 with
   `{"status":"ok","egress_enabled":false}` in < 2 ms.
5. **HTTP auth.** `/check` without a token answers 401 with the message
   `unauthorized: missing or invalid X-Operator-Token` — the error names the
   exact header, so a first-time consumer self-corrects. Auth'd requests
   pass. Unknown JSON fields (`{"bogus":1}`) are silently ignored (200) —
   already owned by `DOC-31`, no new row.
6. **Egress-ON HTTP classification.** A second server with
   `GUARD_EGRESS_ENABLED=true` classified through `/check` end-to-end:
   `allow`, `duration_ms: 580`, `cost_usd: 0.0000276`. This closes the
   classifier-latency leg the 2026-11-14 run recorded as unverified.
7. **Python connector as a real consumer.** Fresh venv
   (`python3 -m venv`; pip 26.2.2), `pip install connectors/python` →
   `guard-agent-client 0.2.0`. Following `connectors/README.md`:
   - `Guard()` + `check(content, source="webhook")` → verdict with
     `override_source: "webhook"`, `route: quarantine` (egress-off
     fail-closed), `deliver: False`.
   - Console script: `guard --source webhook "…"` and the stdin form both
     print the verdict JSON and exit **2** (quarantine) per the documented
     exit-code contract.
   - `Verdict.from_dict` without a `route` key → route `review` with the
     malformed-verdict reason (fail-closed) — a proxy error page can never
     read as "deliver".
   - `Guard(fail_mode="open")` against an unreachable endpoint → route
     `review`, `errored: true`, capabilities flagged `True` — gated, not
     trusted.
8. **Error paths (documented startup failures).** Invalid policy JSON:
   startup refused, rc=1, `not valid guard.Policy JSON: invalid character…`.
   Tokenfile mode 0640: refused with the exact remedy in the message
   (`mode 0640 is too open …; chmod 600 it first`).
9. **Inert-override warning (corroborates closed `GUARD-DF-022`).** Serving
   the fixture policy without any webhook request logs the startup warning
   naming the unselected key and `ValidateOverrideKeys`.

## Performance (Step 2b) — measured, no row

- Warm CLI classify: 0.40–0.58 s (n=5); fail-closed cold CLI: < 10 ms (n=1).
- `/check` egress-off, median of 10: **4 KiB 0.76 ms, 64 KiB 0.90 ms,
  1 MiB 7.81 ms** (p90 9.50 ms).
- The 2026-11-14 record published 6.4–8.0 ms @ 64 KiB and 96–99 ms @ 1 MiB.
  Live today is ~7× and ~12× faster. This is the `GUARD-DF-020` /
  `GUARD-DF-020-R2` normalization work landing (both rows complete;
  benchmark doc claims 551 MB/s @ 8 MiB). Confirmed live with
  `bytes_seen: 1013637` — a real 1 MiB body.
- Nothing here is user-felt slowness, so **no PERF row was filed**; the
  numbers corroborate the closed rows instead.

## Findings

- `GUARD-DF-023` (P4): the `-version` flag is absent from README,
  `skills/guard-usage/SKILL.md`, and `docs/operations.md` — grep census 0/0/0
  — while `-h` lists it and CHANGELOG documents it. Acceptance: census ≥ 1 on
  README and the usage skill, mentioning the `dev` vs stamped-version
  difference.

## Boundary / left unverified

- Battery-based install leg: launched on `bunker-mvp` (`agent=7a98d021`,
  launched 01:06 UTC with `--keep`); result recorded in the dogfood-log
  entry and the tick report. The pre-existing agent `f6a5f076` (created
  00:08, before this tick) is not this fire's and was left untouched.
- No classifier accuracy claim is made (three benign messages classify
  allow; that is a liveness check, not an accuracy audit).
