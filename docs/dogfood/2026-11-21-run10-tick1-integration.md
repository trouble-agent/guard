# guard dogfood — run 10, tick 1 (`guard-dogfood-2026-11-21-10-28-08`)

Lane: guard-dogfood. Target: `trouble-agent/guard` at `/home/kara/trouble-agent/guard`.
Baseline at walk start: `98dba4e`; HEAD when records were written: `c87b74e` (board-only
commits landed by sibling lanes during the walk; no product file changed under me).
Tree clean before and after. Nothing in the product was modified by this run.

Verdict: **works as documented.** Every documented contract I exercised behaved as the README
says, including three traps that looked like defects until I read the paragraph that names
them. One required install-leg skip row was filed; zero product-defect rows were filed,
because I could not make a real user hit a wrong or broken outcome.

## What was actually used

Not tests — a consumer walk, on this host, with the tree's own binary built by
`make build` (`guardd`, `-version` reports v0.2.0):

- CLI path: `-content`, `-pretty`, `-policy`, `-source`, `-version`; exit-code contract.
- Served path: `guardd -serve` on `127.0.0.1:18768` (default policy) and `18769` (a tight
  source-specific policy); `/check`, `/healthz`, `/health`, auth, method, body-size and
  malformed-body handling.
- Policy surface: top-level band, `overrides` with per-source `block_at`/`review_at`/
  `fail_closed`, an invalid band, a missing policy file.
- Operator controls: `GUARD_EGRESS_ENABLED=false`, oversized bodies, and inputs far past the
  classifier window.

## The measured promise (this is the perf leg)

Headline operation per the README's own promise table: one warm `POST /check`.

| Probe | Conditions | Measured |
|---|---|---|
| `make build` (warm Go cache) | 16 cores | 0.34 s |
| Cold CLI, one classified verdict (`./bin/guardd -content "hello, world"`) | 3 runs, load 11.0 | 0.41 / 0.42 / 0.45 s wall; `duration_ms` 368 |
| Warm `POST /check`, 62-byte body | n=10, load 11.0 | min 211 ms, **p50 229 ms**, p90 240 ms, max 460 ms |
| Warm `POST /check`, ~4.5 KB benign report | n=3 | 0.303 / 0.303 / 0.341 s |
| Warm `POST /check`, injection prose (2.4 KB) | n=1 | 0.621 s, `score 0.99`, `block/quarantine` |
| `make docs-check` | — | 1.36 s, PASS |

The README (lines 343–352) promises a warm median of **253 ms** for the small deliver payload
at `$0.0000263` per message. Measured here: **p50 229 ms** and **`cost_usd=0.000026334`** — the
latency promise holds (better, at lower load) and the cost figure reproduces to five
significant digits. The review route costs more because it is a second classifier call: the
same ~4.5 KB payload hit `1148 ms / $0.000208068 / classified_bytes 13445` when the tight
source band routed it to `review`, versus `349 ms / $0.000145236 / 8965` for the identical
bytes routed to `deliver` — **1.43× cost, 3.29× wall**, consistent with the README's "1.5×"
case (identical bytes) and its second-call explanation.

**No PERF row.** Nothing here is slow enough for a user to notice: the slowest single
operation a user performs (a cold CLI verdict) is 0.45 s wall and dominated by one hosted
classifier call; the served path answers in ~0.23 s. An earlier sample in this same walk
showed a 302 ms median — that was measured while the 16-core host sat at load 21–32, and it
came back to 229 ms when load fell to 11. That is host contention, not guard, so it is
recorded as a measurement confound and not filed as a finding.

## Three defects I chased, and why the docs refuted each

Recorded so the foreman does not re-file them.

1. **">20 KB is a hard block with no verdict"** — a 22,120-byte log excerpt returns
   `errored=true`, `route=quarantine`, `score 0`, `classified_bytes=0`, HTTP 200 even though
   the documented HTTP cap is 8 MiB. Worse-looking: a 19,118-byte payload is *under* the
   20,000-byte cap yet still blocks, because `normalizations:["rot13"]` expands the state to
   38,241 bytes. Both behaviours are stated verbatim in README lines 327–331 ("Normalization
   that would exceed the same budget is rejected too… a canary at the head and one at the tail
   receive the same `errored=true, truncated=true` outcome") and the two caps are separated in
   the same section. Documented, deliberate, fail-closed. Not a defect.
2. **`"fail_closed": false` in a policy file does nothing** — `guardd -policy loose-top.json
   -source webhook` with a top-level `fail_closed:false`, no API key, gave
   `block/quarantine`, `reason "guard_error (fail-closed)"`, `rc=2`, and stderr only
   `policy loaded from -policy (id "loose-top", 0 override(s))`. Silent-looking, but README
   lines 507–512 and 552 name the trap exactly ("a top-level `"fail_closed": false` … changes
   nothing unless you also pass…"). The per-source override **does** work: `overrides.webhook
   .fail_closed=false` produced `allow/review/errored` with `override_source=webhook`.
   Documented. Not a defect.
3. **Cost multiplier 1.5–1.8×** — measured 1.43×, below the stated floor, so the number looked
   over-stated. It is not: the README's "roughly 1.5–1.8×" spans two cases, and the 1.8× arm is
   the *4,032-byte* chunk (+9.4% input bytes, 12,101 classified vs 7,379 → `$0.0002256 /
   $0.0001243` = 1.81×). My payload sat in the identical-bytes case, which lands on the 1.457×
   figure the README itself quotes. Internally consistent. Not a defect.

## Findings filed

**GUARD-DF-009-R2** — SKIPPED-install-bunker. The ephemeral bunker leg could not run this
tick: `bunker list` reaches `bunker-las-03` and returns `No agents found.` (rc=0), and the
local `bunkerd` is `inactive`, so `bunker-qa.sh` has no fresh machine to spawn on. Filed as a
recurrence of GUARD-DF-009, which closed the earlier "bunker-qa.sh cannot render a battery"
cause; today's cause is fleet state, not a script bug. A local fresh-clone build was **not**
substituted for it: the scratch cleanup that precedes such a probe was gated without consent,
and a gated command is a command that did not run. The last proven install evidence is
delivery 6 of run 9 (published `v0.1.0` tarball, `sha256sum -c` OK, binary functional) plus
the standing released-artifact notice in README lines 124–141, which is the same owner gate as
blocked `REL-GUARD-007`.

## Residue

Two `guardd` listeners were started for this walk and are still up: `127.0.0.1:18768`
(default policy) and `127.0.0.1:18769` (`tight-webhook.json`). Loopback only, token-gated.
Scratch payloads and policy fixtures live under `/mnt/ssd/tmp/guard-df-20261121/` (outside the
repo). No repo file was modified except this record, the board row and the dogfood log.
