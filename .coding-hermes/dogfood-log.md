# Guard dogfood log

## 2026-10-02 — first dogfood run (tick guard-dogfood-2026-10-02-02-20-45)

- **Verdict: PROMISING-BUT-ROUGH** — core classification works end to end in
  all three consumption shapes; trust/usability edges need work.
- **Promise:** classify untrusted messages for prompt injection via CLI /
  HTTP / Python connector with one shared verdict contract. **Reality:** held
  up — same Result from all three; direct, ROT13 and base64 injection all
  blocked at 0.98, benign delivered.
- **Time-to-first-success:** ~15 min (binary builds in seconds; the wait was
  credential loading, which is undocumented — GUARD-DF-003).
- **Top findings:** GUARD-DF-001 (P1: fail-open outage strips tools/secrets
  via constraints={}), GUARD-DF-002 (P2: normalizations field misdocumented,
  rot13 on every result), GUARD-DF-003 (P2: no quickstart, prereqs
  undocumented). GUARD-001 corroborated live pre-fix, fix verified post-fix
  on the fresh box.
- **Install leg:** ran on ephemeral bunker agent (las-bunker-03, destroyed
  after). Fresh Debian 13: go build 15.5s cold, vet clean, tests pass,
  headline feature identical to dev box. Install friction = undocumented
  prereqs, not broken code.
- **Perf:** CLI 293–327ms warm / 260ms HTTP / 0.25–0.9s cold — provider-bound
  (Jev API); no PERF row (nothing locally slow enough to feel).
- **Left behind:** docs/dogfood/2026-10-02-integration.md,
  docs/dogfood/diagnostics.md, skills/guard-usage/SKILL.md, 3 new board rows
  + GUARD-001 cross-evidence note.

## 2026-10-13 — dogfood run 2 (tick guard-dogfood-2026-10-13-14-52-46, READ-ONLY lane)

- **Verdict: SOUND** — all three consumption shapes re-proven at HEAD b23aad1;
  new surface this run = HTTP auth/error paths, Go library consumer, Python
  connector on a fresh box.
- **Install leg:** PASS on bunker-mvp agent guard-dogfood (las-bunker-03 +
  bunker2/3/4 unreachable over tailscale; single live host suffices per skill).
  Clone→build 33s (go 1.26.6 auto-download)→fail-closed first run rc=2→tests
  green→pip connectors/python→guard CLI works. ~3min cold; agent destroyed,
  residue re-checked (none).
- **Findings (SUGGESTED, not filed — lane is read-only):** DF-G1 {input} vs
  {content} doc bug in main.go:7; DF-G2 raw decode-error leak; DF-G3 CLI stdin
  unbounded ReadAll vs 8MiB HTTP cap; DF-G4 >8MiB 400 message lacks the limit;
  DF-G5 truncated mq-01 example in threshold-calibration.md. Full evidence:
  docs/dogfood/2026-10-13-run2-findings.md (commit c1fa6eb).
- **Perf:** HTTP 394ms±13 warm (provider-bound); CLI 638ms±112 (same Jev floor);
  local path 4-8ms. No PERF row — nothing locally slow enough to feel.
- **Board note:** tasks.jsonl tail shows DOC-2..DOC-10 'done' rows duplicating
  earlier 'pending' rows for the same ids (52 lines, dup-id pairs DOC-2/6/7/8/10,
  README-2..6) — status-by-id is ambiguous; foreman should reconcile.
## 2026-10-16 — dogfood run 3 (tick guard-dogfood-2026-10-16-15-45-07, READ-ONLY lane)

- **Verdict: SOUND** — angle = the `-channel` flag surface (documented 10-14
  by DOC-13, never driven) + released-artifact check (v0.1.0) + fresh-box
  install re-proof. Delta vs run 2 (b23aad1→aeb2f80): docs DOC-12/13/14,
  README-3/4/5, SEC-GO-1269 go 1.26.9 bump — no runtime code changed.
- **Fresh-box install leg (bunker-mvp agent guard-df-1016, destroyed,
  residue recheck clean):** clone from PUBLIC origin → aeb2f80 (main tip) →
  `go build ./cmd/guardd` 41s cold (go1.26.9 auto-download) → fail-closed
  first-run rc=2 exact documented shape → bogus key rc=2 "key rejected:
  http 401". INSTALL_SECONDS=44. pip connector FAILED verbatim (PEP 668
  externally-managed-environment, Python 3.12.3); rescue
  `--break-system-packages` green; connector CLI rc=3 connection-refused on
  a serverless box = correct documented failure.
- **Findings (SUGGESTED, lane read-only):** DF-G6 (P2, -channel/-source
  accepted then silently discarded — classifier never sees the labels,
  Result echoes neither, README:212 says "classifier label"; proven by a
  5x5 matrix, verdicts identical); DF-G7 (P3, Jev model noise 0.46-0.53
  straddles the 0.5 review threshold on a borderline payload — route flips
  deliver↔review across identical replays); DF-G8 (P1, released v0.1.0
  README lacks every post-tag user-facing contract: rc=2 exit codes, egress
  kill-switch, -channel, go 1.26.9 security fix — fix direction = cut
  v0.1.1, not doc edits); DF-G9 (P2, fresh-box pip install needs the PEP 668
  footnote). DF-G4 re-proven fresh; DF-G1 premise holds (main.go:7).
  Full evidence: docs/dogfood/2026-10-16-run3-findings.md (this commit).
- **Perf:** headline CLI classification warm 391.0ms ± 39.0 (provider-bound,
  same envelope as run 2's 394±13); cold ~0.85s; startup 2.8ms; local floor
  3.0ms. **No PERF row** — nothing locally slow enough to feel; go 1.26.9
  bump moved nothing.
- **Board:** run 2's duplicate-status caveat RESOLVED by foreman (588a5dd,
  10-14); last-occurrence census = 45 unique ids, 0 open rows.


## 2026-10-16 — dogfood run 3b (tick guard-dogfood-2026-10-16-15-45-07, READ-ONLY lane; SECOND run under the same tick)

- **Verdict: SOUND** (consistent with runs 1-3). Sibling run 3 (7a0aca3, committed
  16:09:39Z mid-tick, agent guard-df-1016 DESTROYED) covered the -channel/-source
  no-op surface (DF-G6), threshold jitter (DF-G7), released-artifact staleness
  (DF-G8), PEP 668 connector friction (DF-G9), and a hand-walked fresh-box install
  (44s). This run changed the ANGLE per the same-tick-sibling rule: the surfaces
  run 3 did NOT touch.
- **New surface 1 — `make docs-check` (the repo's own README-replay drift gate,
  never driven by runs 1-2):** gate PASSES its replay legs live (README build +
  stdin example replayed verbatim, empty-stdin rc=2 OK). New finding DF-G10 [P2]:
  the gate's `print-go-directive` Makefile target is a hardcoded constant
  (Makefile:106-107, `@echo 1.26.6`) while go.mod declares `go 1.26.9` since the
  SEC-GO-1269 bump (a1b8ff3) — the gate built to catch stale Go-version claims now
  rubber-stamps the stale claim; README.md:12 + skills/guard-usage/SKILL.md:14,84
  still say 1.26.6 in-tree. Fix: parse go.mod (source of truth) instead of the
  echo, then correct the three stale claims.
- **New surface 2 — standard bunker-qa.sh battery (bunker-mvp agent 102dffc5,
  TTL 4h, DESTROYED by collect, bunker list clean):** fresh-install OK (agent
  distro go 1.26.5 < 1.26.9 floor; go.mod's toolchain directive auto-downloaded
  go1.26.9 itself — the security bump works on a fresh box, no hand-install),
  ci-pass OK (act 1 job green), upgrade OK (v0.1.0 -> HEAD clean), chaos-resource
  PASS (3G cap), chaos-errorpath OK (rc=2 clean), chaos-disconnect UNVERIFIED
  (zero output under network cut — battery cell gap, not a product verdict).
  19 evidence rows. Full battery ~12min wall; install_seconds≈720 battery-side.
- **Re-proofs at HEAD:** DF-G1 (main.go:7 still {input}; body {"input":...} ->
  400 "content is required"), DF-G4 (9MiB -> 400 "bad request: unexpected EOF"),
  DF-G5 (threshold-calibration.md:20-21 truncated aUdub3Jl example unchanged).
  DF-G2 NARROWED: the raw decoder leak is HTTP-body-only; CLI stdin "not json"
  classifies as plain text rc=0 — run 2's "CLI path identical" premise corrected
  (no new row; scope of the old row's fix = http decode arm only).
- **Category quality (real use):** direct injection block 0.98-0.99 quarantine;
  ROT13-wrapped injection block w/ rot13 normalization (GUARD-DF-002 holds);
  benign quarterly-report text allow->review 0.51 (matches the documented
  11-of-33 benign-review band); base64 corpus payload mq-01 block 0.99.
- **Perf:** headline CLI classification warm 405.2ms ± 68.7 (hyperfine 10 runs,
  provider-bound Jev round-trip; user 15.1ms sys 9.0ms); fail-closed no-key 5.1ms;
  no-key cold first run 193.8ms. **No PERF row** — only felt cost is the designed
  classifier round-trip; matches runs 1-3's envelope.
- **Full evidence:** docs/dogfood/2026-10-16-run3b-findings.md (this commit).

## 2026-10-19 — dogfood run 4 (tick guard-dogfood-2026-10-20-01-29-06, HEAD bcb6002)

- **Verdict: SHIPPABLE (core)** — every documented contract of all four consumption
  shapes was driven live and matched the docs. Angle = the surface no run had touched:
  the policy layer as an integrator configures it (per-source `Overrides` — the
  REVIEW-GUARD-003 fix, landed 10-19 and never dogfooded — plus `QuotedDowngrade`),
  the connector contract DOC-18/17-R3 documented on 10-18, and a fresh-machine install
  leg. Delta vs run 3b (0b9db86): docs DOC-16/17/18/19 + README-8, code `d71b80c`.
- **Fix re-proved from the consumer side:** `webhook` override BlockAt 0.55 at score
  0.60 → quarantine/block; inbox and empty source keep the base review route.
  `QuotedDowngrade` holds live on three independent quotation phrasings (0.97/0.97/0.95
  above the 0.85 block band → review, never deliver, never quarantine).
- **Contract matrix, all exact:** /healthz body, 401/405/400 bodies, 8 MiB truncation
  behaviour, Python CLI exit codes 0/1/2/3, `--fail-open` outage verdict (decision
  'error', constraints preserved), argparse usage rc=2 with no stdout JSON,
  `--endpoint` vs GUARD_ENDPOINT precedence, the import-time env snapshot trap, Go
  client FailOpen withholding capabilities. 5-way concurrency: all 200, no bleed.
- **NEW findings (rows filed this tick, family GUARD-DF):** GUARD-DF-004 [P2] fresh-box
  `make docs-check` fails on an undocumented `jq` dependency and misdiagnoses it as a
  README JSON parse error (root-caused; rescue with jq in $HOME → rc=0); GUARD-DF-005
  [P2] per-source Overrides unreachable from CLI/HTTP (main.go:108 hardcodes
  DefaultPolicy) while docs/threshold-calibration.md:111,156 prescribes them;
  GUARD-DF-006 [P3] the verdict's own `duration_ms`/`cost_usd` exclude the attribution
  call (stub: 2 calls, cost 0.0010 reported vs 0.0020; live: caught wall 0.53-0.61s vs
  duration_ms 263-362, overlapping the deliver range 248-290); GUARD-DF-007 [P4] an
  Override key matching no source is silently ignored; GUARD-DF-008 [P3] roll-up of
  run-2/3 findings still live and never filed (DF-G1, DF-G3, DF-G4, DF-G6 re-proved);
  GUARD-DF-009 [P2] SKIPPED-install-bunker — `bunker-qa.sh` aborts its render
  (`UI_PORTS: unbound variable` → 0-byte qa-run.sh), installability proved instead by a
  hand-walked fresh-agent leg (INSTALL_SECONDS ~= 60, agent ae4ac4b7 destroyed).
- **Perf:** deliver verdict 0.25-0.29s warm / caught 0.53-0.61s (2 classifier calls by
  design); local floor 0.00-0.01s; docs-check 1-2s; tests 13s on a fresh box.
  **No PERF row** — the only felt cost is the designed remote round trip; the 2x is
  filed as the accounting gap (GUARD-DF-006), not as a slowdown.
- **Full evidence:** docs/dogfood/2026-10-19-run4-findings.md (this commit). Scratch
  consumer (outside the repo): ~/guard-integrator-1019/.

## 2026-10-19b — dogfood run 4b (tick guard-dogfood-2026-10-20-01-29-06, SECOND executor of the same tick: CLI wire shape + override internals)

- **Verdict: SOUND** (no new product-blocking finding). Run 4a = commit `b05b8da` by this tick's
  first executor (consumer + HTTP + Python connector + `make docs-check` + install leg). This run
  changed the ANGLE per the same-tick-sibling rule, and re-proved run 4a's rows on ITS OWN
  artifacts (~/guard-df-run4, HEAD build bcb6002).
- **GUARD-DF-005 re-proved behaviourally on the CLI shape** (row byte-untouched; fresh evidence in
  events.jsonl): in-band payloads scored live ABOVE the documented override's BlockAt 0.55 --
  `guardd -content "<B2>"` 9 runs: score 0.76-0.84, `route=review` with `-source webhook` AND
  `-source inbox` AND no flag (identical; a reachable per-source override at BlockAt 0.55 must
  quarantine anything 0.55 <= score < 0.85). Same build, in-process consumer with the README
  literal at score 0.60 + source=webhook -> `quarantine/block`. HTTP A/B identical
  (`POST /check {content,source:webhook}` -> review). `-fail-closed` DOES reach the policy, so the
  gap is the missing `Overrides` loader, not the flag plumbing.
- **`-channel` inert, independently re-proved** (3-variant live matrix -> identical verdicts; only
  non-test references are the flag decl and the `Input` assignment) -- sub-evidence of GUARD-DF-008,
  not a new row.
- **New findings (rows filed this tick, family GUARD-DF, ids minted via board_next_id.py):**
  GUARD-DF-010 [P3] `Override.FailClosed` is exported + JSON-tagged but never read -- inert in BOTH
  directions (deterministic stub consumer: global true + override false -> still fail-closed
  quarantine; global false + override true -> still fail-open review; each identical to its control
  source); GUARD-DF-011 [P4] the verdict cannot report that a per-source override was applied
  (`policy` stays "default", reason wording unchanged).
- **GUARD-DF-009 re-proved independently** (render break, no agent needed):
  `bunker-qa.sh __gen-remote /home/kara/trouble-agent/guard` -> rc=1, 0-byte output,
  `bunker-qa.sh: line 1061: UI_PORTS: unbound variable`. Line-level root cause added to the
  evidence: the unquoted render heredoc (line 1061 `cat <<EOF`, body 1062-2757) expands on the
  controller under `set -uo pipefail` (line 44), and lines 2397-2407 lost the escape fence the
  surrounding lines keep (`UI_PORTS="${BUNKER_QA_UI_PORTS:-}"`, `[ -z "$UI_PORTS" ]`, ...); line
  2407's unescaped `$(...)` is the same gap's second hazard (it would execute on the CONTROLLER if
  the abort at 2398 did not fire first). Event appended to the EVENTS file; GUARD-DF-009 stays the
  single SKIPPED-install-bunker row.
- **Install leg re-run by hand on its OWN fresh agent** (skill's sanctioned substitute; agent
  `701e4fac` @ bunker-mvp, TTL 90m, DESTROYED, key removed, list re-checked clean): public-origin
  clone -> `b05b8da` -> build rc=0 45.94s (image go1.22.2; the toolchain directive auto-downloads
  go1.26.9) -> fail-closed first run rc=2 byte-exact documented JSON -> serve: /healthz 200,
  no token 401, token 200 -> `make docs-check` rc=2 "could not parse README fail-closed JSON block"
  with `jq` MISSING on the box (GUARD-DF-004 independently reproduced) -> `go test ./... -count=1`
  rc=0 12.97s. INSTALL_SECONDS ~= 46 (build dominates). Battery cells remain unproven (driver break).
- **Perf (Step 2b):** warm CLI deliver 343.9ms +- 40.1 (hyperfine, 8 runs, 1 classifier call);
  warm caught verdict 590.8ms +- 29.7 (5 runs, 2 calls by design); local floor (egress disabled)
  4.9-7.8ms; process startup floor 3.7ms +- 1.1; in-process policy evaluation (whole 11-case
  consumer, stub classifier) 3.4ms +- 0.5. **No PERF row** -- every local path is single-digit ms
  and the only felt wait is the designed remote round trip (accounted by GUARD-DF-006).
- **Not exercised (stated, not passed off):** the bunker-qa.sh battery cells (ci-pass/upgrade/
  chaos-*) -- driver break above; a real classification on the fresh box (no key there; credentials
  never minted or copied into a bunker); the Python connector and Go client (covered by run 4a,
  deliberately not repeated).
- **Full evidence:** docs/dogfood/2026-10-19-run4b-second-angle.md (this commit). Scratch:
  ~/guard-df-run4/ (consumer, scans, ab-cli, http-battery, install-leg, gen.*).

## 2026-10-19c — dogfood run 4c (tick guard-dogfood-2026-10-20-01-29-06, THIRD executor of the same tick: the published calibration numbers + all six normalisers + the battery ran for real)

- **Verdict: SOUND** (no product defect found; one documentation-tool honesty gap and one battery-cell
  gap filed). Run 4a = `b05b8da`, run 4b = `444446d`; this executor took the two surfaces neither had
  driven: **the documented calibration replay** (docs/threshold-calibration.md:40-54) and **the six
  documented normalisers as an evasion ladder** - plus the first REAL battery run for this repo.
- **Published numbers reproduce EXACTLY at HEAD** (first live re-measure since 2026-10-04): confusion
  attack 31 block/1 review, quoted 3 block/2 review, benign 21 allow/1 block/11 review; block band
  tp=34 fp=1 fn=3 P=0.971 R=0.919; review-or-above tp=37 fp=12 fn=0 P=0.755 R=1.000; route agreement
  attack 31/32, quoted 5/5, benign 21/33 - identical to docs/threshold-calibration.md. Named samples
  moved within +-0.04 (so-08 0.78 vs 0.80, bn-06 0.87 vs 0.85, bn-18 0.84 vs 0.83; bn-30 0.47 vs 0.51
  flipped band) while the band counts stayed identical, so the doc's aggregate claim holds and its
  per-sample prose is a snapshot - recorded in the log only, no row (the doc's own Residuals already
  expect drift).
- **Evasion ladder, 14 payloads, no evasion:** direct, base64, hex, ROT13, URL-encoded, zero-width,
  homoglyph and a homoglyph+zero-width mix all landed quarantine (0.98-0.99) where before
  (0.98-0.99) - and the same eight again as BARE blobs with no English lead-in (0.98-0.99,
  `attack_class=instruction_injection`). Only ROT13 reports `load_bearing_normalizations=["rot13"]`,
  which is the field's documented meaning (the decode DROVE the verdict): for base64/hex/url/zero-width/
  homoglyph the raw arm is caught too, so the decoder is not what carried those verdicts. First time
  hex / URL-encoding / zero-width / homoglyph are driven by any recorded run.
- **NEW findings (minted via board_next_id.py):**
  GUARD-DF-012 [P3] the documented calibration replay prints a full precision/recall block for a run that
  classified nothing - all-errored run: rc=0, `Block-band tp=0 fp=0 fn=37 precision=0.000 recall=0.000`,
  the review arm contradicting it (fn=0 for the same run), `Route-agreement attack 32/32` reading as
  success (fail-closed quarantine satisfies the attack route), zero warning text; the repo's own Go gate
  (calibration_test.go:167-170) does fail loudly, so the script the doc points at is the weaker surface.
  QA-GUARD-005 [P4] the battery's chaos-disconnect arm cannot run guard's suite at all: it runs the
  native command under `GOPROXY=off GOTOOLCHAIN=local` (bunker-qa.sh:2482) and go answers `go.mod
  requires go >= 1.26.9 (running go 1.26.5; GOTOOLCHAIN=local)` (reproduced locally, rc=2, 124-byte log),
  so the cell grades UNVERIFIED (2nd consecutive run) and the vacuity branch prints no rc/log bytes.
- **Install leg: the driver is FIXED and the battery RAN** (`bunker-qa.sh` mtime 20:59, the unescaped
  `UI_PORTS` fragment now escaped; `__gen-remote` -> rc=0, 119,363 bytes). Full battery on fresh agent
  `990d3b37` @ bunker-mvp: fresh-install OK, ci-pass OK (act 1 job green), upgrade OK (v0.1.0 tree ->
  HEAD clean), chaos-resource PASS (3G cap), chaos-errorpath OK (rc=2), 4 N/A (no compose/UI/db),
  chaos-disconnect UNVERIFIED (QA-GUARD-005), collect OK - agent destroyed by collect, no local key
  left, list re-checked. So GUARD-DF-009 (SKIPPED-install-bunker) is FALSIFIED: the battery ships and
  runs; event appended to that row for the foreman to close.
- **Perf (Step 2b):** documented calibration replay = **37.3 s for 70 rows** (0.53 s/row, live Jev); the
  all-errored negative leg = instant (local fail-closed, no network); battery wall ~4-5 min
  (launch 02:00:22Z -> collect 02:06Z). No PERF row - every local path remains single-digit ms.
- **Not exercised:** docker/UI/state chaos cells (N/A for this repo - no compose file, no web UI, no
  db/state files), and a real classification on a bunker agent (no key there, by design).
- **Full evidence:** docs/dogfood/2026-10-19-run4c-findings.md (this commit). Scratch:
  ~/guard-df-run4/ (ladder*.py, disc-local.sh, out/{calibration*,ladder*,disc-*,gen-4c.*,qa-launch-4c.txt}).

## 2026-10-19d — dogfood run 4d (tick guard-dogfood-2026-10-20-01-29-06, FOURTH executor of the same tick: both landed fixes re-proved + the operator/parity surface)

- **Verdict: SOUND.** This executor was delta-driven: the tick's own rows were FIXED while it ran
  (`2a2fc30` GUARD-DF-010, `e858041` GUARD-DF-006, `6836cc8` trace carrier), so the real-use pass
  became the independent re-proof of both fixes on a clean `git archive` build of HEAD 6836cc8
  (never the dirty working tree -- a sibling lane's in-flight edits were present in the checkout at
  the start of this run and were gone by the end, so they are neither measured nor judged here).
- **GUARD-DF-010 (per-source `Override.FailClosed`) VERIFIED** from the consumer side, 10 cases on
  a determinant stub classifier: fail-closed policy + `webhook:{FailClosed:false}` -> webhook
  `review/allow` reason "(fail-open)" while the inbox control and the empty source stay
  `quarantine/block`; the mirror (fail-open policy + `webhook:{FailClosed:true}`) -> webhook
  quarantines, control fails open; both fields on one override coexist (error -> fail-open review,
  score 0.60 -> quarantine); the score path is untouched; a source with no override keeps the
  policy stance. Event appended to the row.
- **GUARD-DF-006 (attribution accounting) RE-PROVED with a before/after pair:** the shipped
  `JevClassifier` pointed at a local stub (cost 0.0010/call, 120 ms attribution lag) now reports
  deliver 1 call cost 0.0010 == true 0.0010, and quarantine 2 calls cost 0.0020 == true 0.0020
  (run 4b measured 0.0010 reported vs 0.0020 true on the pre-fix build) with `duration_ms` 240
  covering the attribution lag. Event appended to the row.
- **NEW findings (minted):** GUARD-DF-013 [P4] `skills/guard-usage/SKILL.md` still documents the
  pre-fix semantics ('duration_ms and cost_usd describe the FIRST classifier call only') -- a
  paragraph dogfood run 4b wrote and `e858041` invalidated; the fix commit itself flags it as out
  of scope. GUARD-DF-014 [P4] `guardd` logs `listening on <addr>` BEFORE the bind, so a port
  conflict prints the success banner and then `serve: listen tcp ...: bind: address already in use`
  with rc=1 -- a readiness-looking line for a server that never listened.
- **Operator ladder, all documented legs hold:** no token -> rc=1 with the exact refusal text;
  `-tokenfile` mode 0644 -> rc=1 "mode 0644 is too open (group/world bits set); chmod 600 it first";
  0600 accepted ("token loaded from tokenfile"); `GUARD_TOKEN` beats a staged tokenfile (env token
  200, file token 401 on the same server); bare `:port` binds 127.0.0.1 only, explicit `0.0.0.0:port`
  binds `*`; `-token` on argv prints the loud ps-visibility warning; `GUARD_EGRESS_ENABLED=false`
  serves with the documented WARNING, `/healthz` reports `egress_enabled:false`, `/check` answers
  the in-band error verdict with HTTP 200 and the process keeps serving. `/healthz` accepts every
  method (GET/POST/PUT/DELETE/OPTIONS/PATCH/HEAD -> 200, as documented); `/check` is 401-before-405
  (auth first); `/nope` and `/check/` 404; empty body -> 400 `bad request: EOF`; `{"content":null}`
  -> 400 `content is required`.
- **In-process vs HTTP parity: the documented "one-line change" holds.** Same input through the
  in-process core and through `client.New(...).Check` against a live `guardd`: route, decision,
  risk, errored, policy, provider, attack_class and constraints agree on all three payloads
  (benign deliver / injection quarantine / quoted review). The HTTP hop cost is in the noise
  (-88 ms, -9 ms, +5 ms against a 240-555 ms remote classifier), so the documented zero-hop
  in-process advantage is unmeasurable at this latency -- no PERF row.
- **Install leg:** `bunker-qa.sh` renders (rc=0, 119,363 bytes) and the FULL battery ran on a fresh
  agent `3ea7d415` @ bunker-mvp at HEAD `6836cc8`: repo-head OK, toolchain-bootstrap OK,
  fresh-install OK, ci-pass OK (act 1 job green), upgrade OK (v0.1.0 tree -> HEAD reinstall clean),
  chaos-resource PASS (3G cap), chaos-errorpath OK (rc=2), 4 cells N/A by repo shape,
  chaos-disconnect UNVERIFIED (third consecutive observation -- QA-GUARD-005), collect OK, agent
  destroyed by collect. Event appended to QA-GUARD-005.
- **Perf (Step 2b):** startup-to-first-200-/healthz **13-15 ms** (5 fresh processes; the operator's
  "can I send traffic" number); CLI local floor 3.8 ms +- 0.4 (hyperfine, 10 runs); clean-copy
  suite 4.0 s; battery wall ~4 min. **No PERF row** -- nothing a user would feel.
- **Not exercised:** the docker/UI/state chaos cells (N/A by repo shape), a real classification on
  a bunker agent (no key there by design), and the sibling lane's in-flight edits (uncommitted and
  then withdrawn -- explicitly NOT measured, as they were never part of HEAD).
- **Full evidence:** docs/dogfood/2026-10-19-run4d-findings.md (this commit). Scratch:
  ~/guard-df-run4/{consumer-4d,parity-4d,stubprobe-4d,ops-ladder.sh,parity.sh,startup-4d.sh,clean-copy*.sh}.

## 2026-10-23 — dogfood run 5 (tick guard-dogfood-2026-10-23-04-13-22, HEAD f0e79c1)

- **NEW ground vs runs 1–4d:** live-key real classification (run 4d's named gap),
  policy-override audit trail over HTTP, connector outage path, README fail-closed
  example reproduction, and the first battery with the QA-GUARD-005-R2 harness fix.
- **Live-key classification (OpenRouter via the product's own key discovery):
  injection -> quarantine 0.98, attack_class instruction_injection, 529 ms,
  cost 0.000054264 (2 calls, post-e858041 accounting verified with real money);
  benign -> deliver 0.23 / 287 ms / 1 call; quoted example attack -> review 0.95
  (QuotedDowngrade). Repeat call -> quarantine 0.98 again (deterministic route).
- **Override audit trail:** -policy with slack_digest override (0.6/0.3) quarantines
  a 0.63-score body under base bands' deliver, and the verdict NAMES
  override_source=slack_digest + effective_block_at 0.6. Typo'd source row still
  silently ignored (GUARD-DF-007 re-confirmed at HEAD).
- **Connector + outage:** python connector fail-open on a dead endpoint -> route
  review, errored=True, deliver=False, tools/secrets still granted (GUARD-DF-001 at
  HEAD), 0 ms fail, no hang.
- **README no-key example:** NOT literally producible where ~/.hermes/.env carries a
  live key (the documented discovery path reads it by design) — reproduced via
  scratch-HOME substitution (rc=2, fail-closed block, exact documented reason).
  Filed **GUARD-DF-015** [P4].
- **Perf:** deliver /check warm 254.3 ms ± 20.2 (hyperfine 10 runs); cold first call
  286–343 ms on 3 fresh servers; /healthz 6.4 ms; egress-off floor 9.6 ms → the
  whole user-visible wait is the Jev round trip. **No PERF row** (nothing a user
  would feel; decomposed per the two-stage rule).
- **Install leg:** FULL battery, fresh agent 2a27aadc @ bunker-mvp (04:22:11Z,
  destroyed by collect + TEARDOWN-RECOVERED). 19 rows: repo-head OK, fresh-install
  OK, ci-pass OK (act 1 job green), upgrade OK (v0.1.0 → HEAD), chaos-resource PASS,
  chaos-errorpath OK, 3 N/A by repo shape, ui-probe INFO. **chaos-disconnect
  UNVERIFIED a 4th consecutive time — now with the toolchain-floor fix in place
  (516423c ran in this battery) the leg grades vacuous-native-suite (disc.log 0
  bytes) rather than toolchain-floor refusal; the suite still produced zero output.**
  The battery evidence is archived outside the repo (stand-in workdir) because the
  lane contract keeps repo writes to rows/docs only.
- **Not exercised:** a multi-tenant concurrency soak; docker/UI/state cells stay N/A
  by repo shape.
- **Verdict: SHIPPABLE.** Real users integrate via the three shapes and the contract
  (route + constraints + errored) holds live at HEAD.
- **Run record parked:** DuckBrain namespace `guard`,
  key `/dogfood/2026-10-23/run5-guard-dogfood-2026-10-23-04-13-22` (id 2de4a648,
  content read-back verified, distinctive-phrases verified).
- **Evidence:** docs/dogfood/2026-10-23-run5-findings.md; scratch /tmp/guard-df5/
  (scripts + responses) — ephemeral; battery evidence JSONL in
  ~/.hermes/stand-in/dogfood/guard/evidence/.
- ch:trace row=GUARD-DF-015 spec=README.md "First run with no key" evidence=docs/dogfood/2026-10-23-run5-findings.md witness=path@scratch:/tmp/guard-df5 verdict=manual-live-use commit=f0e79c1

## 2026-10-29 — dogfood run 6 (tick guard-dogfood-2026-10-29-10-20-47, HEAD 6cfd37f)

- **Verdict: SHIPPABLE.** Go CLI, authenticated HTTP, `/healthz` plus `/health`, Python connector install, documented fail-closed paths, and the fresh-machine install leg all completed at current HEAD. The run-owned bunker agent `1d6f29c2` at `bunker-mvp` produced fresh-install OK (artifact 11,086,664 bytes), ci-pass OK, upgrade OK, chaos-disconnect OK (`DISCONNECT_SUITE_SUMMARY rc=0 outcomes=13`), chaos-resource PASS, and chaos-errorpath OK; teardown was independently verified with `bunker info`=`not_found` and zero list matches. Host capacity (53.4% CPU, 49 containers, 2 existing agents, 70% disk, 20 orphan keys) is context only, not a product failure.
- **Promise:** provide one message-security filter through Go, HTTP, and Python surfaces with a shared Result contract and fail-closed behavior. **Reality:** held on all exercised source-build/user surfaces and the fresh-agent install leg.
- **Real use:** detached HEAD build 0.61s; `go test ./... -count=1` all four packages PASS; `go vet ./...` PASS; `make docs-check` PASS. Owned live server on `127.0.0.1:18947`: `/healthz` and `/health` HTTP 200 in 37ms; `/check` 401 no/wrong token, 400 empty/malformed, 405 GET, and 200 in-band fail-closed quarantine with egress disabled. Live-key CLI classification returned rc=0/deliver; cold process samples 0.86s, 0.30s, 0.29s; five additional runs 319–493ms with service durations 281–457ms. Python venv install rc=0 and installed `guard` console script reached the live service, returning the expected rc=2 fail-closed JSON.
- **Perf:** Jev round trip dominated the measured 0.29–0.49s steady CLI wait; the first process was 0.86s. Nothing was slow enough to justify a PERF row; no profile was run.
- **Findings:** no new product or PERF row filed by this read-only lane. The README no-key example now includes an isolated-HOME reproduction hint; prior GUARD-DF-015 remains historical context. This lane did not mint ids or modify board state.
- **Evidence:** docs/dogfood/2026-10-29-run6-findings.md; /tmp/guard-dogfood-20261029-102047-Bz0g/ (detached build, live probes, timings, Python consumer).
- ch:trace tick=guard-dogfood-2026-10-29-10-20-47 spec=README.md+skills/guard-usage/SKILL.md evidence=docs/dogfood/2026-10-29-run6-findings.md witness=path@/tmp/guard-dogfood-20261029-102047-Bz0g verdict=SHIPPABLE install=PASS agent=1d6f29c2@bunker-mvp commit=verified-local-HEAD

## 2026-11-11 — dogfood run 7 (tick guard-dogfood-2026-11-11-22-36-17)

- Verdict: SHIPPABLE core, with two P4 documentation findings filed as GUARD-DF-016 and GUARD-DF-017.
- HEAD: 92c3e07178f2dd267a1512318029c06e50922c5e (canonical /home/kara/trouble-agent/guard; origin/main matched at discovery).
- Promise: provide one message-security filter through Go CLI, authenticated HTTP, and Python connector, with fail-closed behavior and one Result contract. Real use held: live benign CLI deliver rc=0 (score 0.19, duration_ms=843), live injection CLI quarantine rc=0 (score 0.98, duration_ms=811), no-key isolated-HOME CLI block rc=2, local HTTP health 200, unauthenticated /check 401, malformed/empty content 400, egress-disabled authenticated /check 200 with fail-closed quarantine, and pip-installed Python CLI installation rc=0.
- Performance: direct user-path timings were 0.84s deliver and 0.81s injection; a five-run warmup-2 hyperfine remeasurement of the deliver command was mean 406.0ms ±17.3ms, range 378.8–421.8ms, all exit 0. The wait is dominated by the designed Jev network round trip; no PERF row filed because no local code path was slow enough to justify profiling. GUARD-DF-016 records stale precision in skills/guard-usage/SKILL.md.
- Install leg: bunker-qa.sh launch/collect ran on fresh agent 1867ca04 at bunker-mvp. Terminal evidence contains 23 collected rows: fresh-install OK, lint OK, tests OK, CI act 1 job green, upgrade v0.1.0 to HEAD OK, chaos-disconnect INFO (passes with network cut; no network dependency), chaos-resource PASS, chaos-errorpath OK; Docker/compose/UI/state cells N/A by repository shape. Agent was destroyed by collect; post-collect bunker info returned not_found and bunker list had no 1867ca04. Host census was not a fresh tenant-isolation proof: bunker-mvp has shared network namespaces and four unrelated running agents; that boundary is disclosed, not graded as product failure.
- Findings: GUARD-DF-016 (current performance documentation band no longer reproduces at HEAD); GUARD-DF-017 (usage skill source anchor/CLI stdin-cap statement is stale against current main.go; no oversized-input behavior claimed because this tick did not run a multi-gigabyte probe). Board readback: each id exactly once, paired task_created events present, boardctl validate rc=0 with the pre-existing 73 warnings unchanged.
- Left behind in canonical repo: board rows/events for GUARD-DF-016 and GUARD-DF-017. Controller evidence: /home/kara/.hermes/cache/scratch/guard-dogfood-20261111-223617/ (probe, hyperfine JSON, bunker evidence, validation output). No code or product files modified. Existing unowned files .coding-hermes/audits/guard-readme-2026-10-30-21-53-18.md and .coding-hermes/board/tasks.jsonl.pre-resolve were left untouched.

## 2026-11-11 — duplicate fire of dogfood run 7 (records-only; no new delivery)

This fire carried the tick id `guard-dogfood-2026-11-11-22-36-17`, which already has a committed run record (heading count for that id in this file remains **1**, the run-7 delivery above). Per the dogfood skill's duplicate-fire gate this fire is records-only: no product use, benchmark, install battery, or new row was run or filed. The heading above deliberately does not repeat the tick id so the heading count keeps measuring real deliveries.

Verification of the committed run-7 evidence, re-derived from committed blobs and independent paths rather than from the prior executor's prose:

- Canonical repo resolved from the scheduler row: `/home/kara/trouble-agent/guard`; remote `origin` default branch `main` (`git ls-remote --symref origin HEAD` -> `ref: refs/heads/main`). Checkout was 0 ahead / 2 behind at fire start; fast-forwarded with `git merge --ff-only origin/main` (adds no commits) to `2c822c43`.
- Tested HEAD was `92c3e07178f2dd267a1512318029c06e50922c5e`; current HEAD `2c822c43` is 3 commits later and the whole delta is bookkeeping only (`.coding-hermes/board/*`, `.coding-hermes/dogfood-log.md`, `docs/dogfood/*`, `.coding-hermes/audits/*`). **No product code changed** between tested and current HEAD, so the run-7 verdict still describes the shipped tree.
- Board rows `GUARD-DF-016` and `GUARD-DF-017` are present in the committed blob at HEAD, both `pending`/`P4`/`source=dogfood`, each with one paired `task_created` event (events 204 and 205) carrying the tick id. Whole-row scan of the committed `tasks.jsonl` finds 3 rows citing the tick: those two filings plus `REL-GUARD-003`, whose `dogfood` attribute block records run-7 status/head — a reference, not a duplicate filing.
- Id uniqueness holds for both rows and `boardctl -C /home/kara/trouble-agent/guard validate` -> `RESULT: OK (73 warning(s))`, rc=0, warnings unchanged from before the run.
- Measurement paths cited by the report still exist and reproduce their numbers: `live-benign-hyperfine.json` gives mean 0.40596817 s (406.0 ms), stddev 0.01728595 (17.3 ms), min 0.37882, max 0.42176, five runs all exit 0 — exactly the band the log states. `probe.out` shows rc=0/route=deliver/score=0.19/duration_ms=843 and rc=0/route=quarantine/score=0.98/duration_ms=811, plus the no-key fail-closed rc=2.
- Install-leg ledger reconciled: `bunker-evidence.jsonl` now holds **25** rows against the "23 collected rows" claim. This is not a contradiction — `synced_rows=23` at 22:57:27Z is the battery sync, and the two extra rows are the `launch OK` and `collect OK` control rows appended afterwards. The per-cell ledger is duplicate-free (9 of the 25 are per-tool `toolchain-bootstrap OK`): fresh-install OK, lint OK, tests OK, ci-pass OK, upgrade OK, chaos-disconnect INFO, chaos-resource PASS, chaos-errorpath OK; Docker/compose/UI/shutdown/corruption N/A by repository shape. Teardown was re-verified at this fire: `bunker info 1867ca04 --server bunker-mvp` -> `not_found: agent "1867ca04" not found`, and `bunker list --server bunker-mvp` matched `1867ca04` zero times.
- The two findings' premises were re-checked against current HEAD rather than trusted: `cmd/guardd/main.go:168-174` still reads stdin with an uncapped `io.ReadAll` while the HTTP path keeps `io.LimitReader(r.Body, 8<<20)` at `main.go:244`, and `skills/guard-usage/SKILL.md:165` still cites the stale anchor `main.go:123`; lines 110 and 126 still state the 0.24–0.40 s deliver band. Both P4 rows therefore remain live, not stale.

Ownership at fire start: no live worker, collector, guard, or judge in the checkout (only this lane's own shell processes matched the repo cwd), no Git operation markers. Untracked residue `.coding-hermes/audits/guard-readme-2026-10-30-21-53-18.md` and `.coding-hermes/board/tasks.jsonl.pre-resolve` was left byte-untouched; no sibling board row was adopted, reverted, or closed by this fire.

Left behind: this log record only. No rows, events, reports, code, visibility, or permissions changed.

## 2026-11-14 — dogfood run 8 (tick guard-dogfood-2026-11-14-23-17-36)

- Verdict: SHIPPABLE WITH A SECURITY-SHAPED ROUGH EDGE — core use holds, but the verdict silently ignores all content past ~19 KB, filed as GUARD-DF-018 (P2); plus GUARD-DF-019 (P4) connector/source parity.
- HEAD: 4d4a2954affaceeb6dbc4d22824424e32ab6e900 (canonical /home/kara/trouble-agent/guard; equal to origin/main, clean tree at fire start; no code touched by this run).
- Duplicate-fire gate: no `2026-11-14-23-17-36` / `20261114-231736` anywhere in the repo and no `## 2026-11-14` heading before the write; this is a first delivery.
- Real use, five surfaces: CLI `-content`/`-source`/`-pretty`/stdin; authenticated HTTP (`/healthz`, `/check`, 401 on bad token, egress-disabled fail-closed 200); Go library via `&guard.Guard{Classifier: guard.NewJev(), Policy: guard.DefaultPolicy()}` + `g.Check(ctx, Input{Source, Channel, Content})`; pip-installed Python connector (`guard` console script, 7 cases + 26 package tests); real documents (README.md 28,158 B, CHANGELOG.md 3,967 B, jev.go 7,367 B) and task-shaped content (PR-style body, rot13-obfuscated injection, 64 KB body). Benign deliver, plain injection quarantine, rot13 injection still quarantined, per-source override flips 0.9-block vs 0.6-review, missing key fails closed rc=2 with a visible reason, bad/missing policy file fails loudly, policy reload applied without restart.
- Headline finding (GUARD-DF-018): identical 32,208-byte document, canary injection at the HEAD -> quarantine 0.97 / 942 ms, at the TAIL -> deliver 0.10 / 406 ms. Length bisect 18 KB deliver 0.08, 20 KB 0.17, 22 KB 0.09, 64 KB with injection first 0.99. HTTP replay: 22.6 KB body with injection appended -> deliver 0.03. No verdict field reports bytes classified. Mechanism: guard.go:187 NormalizeAll -> guard.go:190 Classify(ctx, text); jev.go:166 sends the full content as the classifier state with no cap, chunking or truncation notice. Note DOC-27/GUARD-DF-017 (both complete) covered the uncapped stdin path and its stale anchor — this row is the unrecorded half: the ~19 KB cap exists and is unmentioned.
- Second finding (GUARD-DF-019): the shipped Python console script accepts only --endpoint/--token/--timeout/--fail-open/[content]; `guard --source inbox hello` exits 2 with argparse usage, guard_client.py:345 calls check() with no source, while the library supports source/channel (213-222) and the docstring advertises it — so per-source Overrides are unreachable from the installed entrypoint.
- Performance (Step 2b, measured, nothing profiled): process start median 2.6 ms (2.3–6.6 ms, n=5); cold CLI benign 41 B median 424.0 ms wall / 346.8–477.6 (binary self-reports 342–473, so ~99% is the Jev round trip); warm HTTP against one long-lived server median 284.3 ms for 41 B and 298.1 ms for 24,222 B — i.e. FLAT in input size (+14 ms for 590× the content), which is the truncation seen from the cost side. No PERF row filed: 0.3–0.5 s per classification is not user-noticeable and there is no local hot loop to attack; the flat curve is filed inside GUARD-DF-018 instead.
- Install leg (not a skip): bunker-qa.sh status -> bunker-mvp bunkerd=active docker=yes act=no uidmap=yes; resolve -> /home/kara/trouble-agent/guard; launch on ephemeral agent 08c686df (ttl=4h, cpu=4.0); collect pulled 23 rows. fresh-install OK (go build ./...), tests OK rc=0, ci-pass OK (act 1 job green), upgrade OK (v0.1.0 tree via git archive -> HEAD reinstall clean), chaos-resource PASS (3 GB cap), chaos-errorpath OK, lint INFO — golangci-lint was NOT installed on that agent, so lint did not run this tick (run 7 recorded lint OK on a different agent; the capability varies per ephemeral agent). Docker/compose/UI/shutdown/corruption N/A by repository shape. Teardown verified: `bunker info 08c686df --server bunker-mvp` -> not_found, `bunker list` matched 08c686df zero times.
- Board readback: GUARD-DF-018 and GUARD-DF-019 each appear exactly once (`grep -c` = 1), both minted with board_next_id.py GUARD-DF --count 2 and verified free before append, both status pending with acceptance_criteria 496/327 chars; `boardctl -C /home/kara/trouble-agent/guard validate` -> RESULT: OK (73 warning(s)), the same pre-existing warnings as before the run.
- Left behind in canonical repo: the two board rows + their events, this log entry, and docs/dogfood/2026-11-14-run8-integration.md (integration report with the diagnostics trail). No product code, no test, no visibility or permission changed. Controller evidence: /home/kara/.hermes/cache/scratch/guard-dogfood-20261114-231736/ (guardd binary, 12 probe harnesses, perf/perf-cli.json, perf/perf-http-warm.json, bunker-launch.log, bunker-collect.log).
- Not covered: lint (runner absent on the agent); multi-key failover and key rotation; parallel-load throughput; HTTP rejection above the 8 MiB body cap; and the ~19 KB boundary was measured against one provider/model in one session — nothing here claims it is provider-independent.

## 2026-11-14 — duplicate fire of dogfood run 8 (records-only; no new delivery)

This fire carried the tick id `guard-dogfood-2026-11-14-23-17-36`, which already has a committed run
record (the run-8 section above; heading count for that id in this file remains **1**). The delivery
commit `792b373` is timestamped `2026-11-14T18:57:59-05:00`, 66 s before this fire's own first probe
(`date -Is` = `2026-11-14T18:59:05-05:00`), so this is a same-tick re-fire, not a new run. Per the
duplicate-fire gate it is records-only: no product use, benchmark, install battery or new board row
was run or filed by this fire.

Record read before taking the records-only branch, and the legs it names were all walked — real use
on five surfaces, perf measurement, the bunker install leg, and board readback — so this is not an
unfinished record that a closeout would freeze.

Verification, re-derived from committed blobs and independent paths rather than from the prior
executor's prose:

- Canonical repo from the scheduler row: `/home/kara/trouble-agent/guard`; remote `origin`, default
  branch `main`. `git fetch origin main` then `git rev-parse HEAD origin/main` -> both
  `792b373cfc63333df74c2dc7650373a33c184bb1`; `git rev-list --left-right --count HEAD...origin/main`
  -> `0 0`. `git branch -r --contains HEAD` lists `origin/main`. Clean working tree.
- The verdict still describes the shipped tree: `git diff --stat 4d4a295..HEAD` (tested HEAD -> current
  HEAD) is 4 bookkeeping files only (`.coding-hermes/board/events.jsonl`,
  `.coding-hermes/board/tasks.jsonl`, `.coding-hermes/dogfood-log.md`,
  `docs/dogfood/2026-11-14-run8-integration.md`, 242 insertions). **No product code changed.**
- Board rows read from `git show HEAD:.coding-hermes/board/tasks.jsonl`: `GUARD-DF-018` pending / P2 /
  496-char acceptance criteria, `GUARD-DF-019` pending / P4 / 327-char acceptance criteria; each bare
  id appears exactly once. Paired events read from the committed `events.jsonl` (key path
  `event_type`/`task_id`, confirmed by `keys` on the last line first): `task_created` for both rows,
  actor `foreman`, timestamps `2026-11-14T23:54:00Z` and `2026-11-14T23:54:21Z`. Rows carry
  `created_at` matching those instants; there is no `tick_id` field on this board's rows or event
  details, so tick provenance rests on the timestamps and this log.
- `boardctl -C /home/kara/trouble-agent/guard validate` -> `RESULT: OK (73 warning(s))`, rc=0, the same
  pre-existing legacy-shape warnings as before the run. Row count 116 in the working tree == 116 in
  the committed blob (no uncommitted board drift).
- Cited perf paths reproduce their numbers. `perf/perf-cli.json`: process-start runs
  `6.6, 2.6, 2.3, 2.3, 3.0` ms (median 2.6); `cold_benign` wall 477.6, 472.8, 386.9, 346.8, 424.0 ms
  (median 424.0, binary self-report 342–473 ms). `perf/perf-http-warm.json`: 41 B median 284.3 ms
  (266.0–552.2), 24,222 B median 298.1 ms (288.2–412.3) — flat in input size, which is the truncation
  seen from the cost side.
- Install leg ledger reconciled at collect time: `/tmp/bunker-qa-evidence-20261114T234313Z-926128.jsonl`
  holds **25** rows = the 23 battery cells the record quotes (9 of them `toolchain-bootstrap OK`, plus
  `launch OK` and `collect OK` control rows). Cells: fresh-install OK, tests OK, ci-pass OK, upgrade OK,
  chaos-resource PASS, chaos-errorpath OK, chaos-disconnect INFO, chaos-resource/corruption/shutdown/
  docker/compose/ui N/A by repository shape, lint INFO (golangci-lint absent on that agent). Teardown
  re-verified at this fire: `bunker info 08c686df --server bunker-mvp` -> `not_found: agent "08c686df"
  not found`.
- Both findings' premises re-checked at current HEAD instead of trusted: `guard.go:187`
  `NormalizeAll(in.Content, g.Normalizers)` -> `guard.go:190` `g.Classifier.Classify(ctx, text)` is
  still the only normalization of the payload; `jev.go:166` marshals the whole content as the
  classifier `State` with no cap, chunking or truncation notice (the only size limit in the tree is the
  HTTP body `io.LimitReader(r.Body, 8<<20)` at `cmd/guardd/main.go:244`, and the CLI's stdin read
  remains uncapped). The shipped console script still exposes only
  `--endpoint/--token/--timeout/--fail-open` plus the positional content (`connectors/python/guard_client.py:285-310`),
  so the library's `source`/`channel` arguments stay unreachable from the entrypoint; the no-source call
  is at `guard_client.py:342` (the run-8 record cited :345 — a 3-line anchor drift, premise unchanged).
  Both P2/P4 rows remain live.

Ownership at fire start: no Git-operation markers and no `index.lock` in the checkout,
`ps -eo pid,ppid,etime,args | grep trouble-agent/guard` matched nothing, so no live worker, collector,
guard or judge held the tree. `git worktree list --porcelain` shows 15 linked worktrees under
scratch/worktrees (booking residue, all detached or already-merged `wt/*` branches); none carries an
in-progress marker and none was touched by this fire.

Left behind: this log record only. No row, event, report, code, visibility or permission changed, and
no duplicate filing was created for either finding.

- ch:trace tick=guard-dogfood-2026-11-14-23-17-36 duplicate=yes of=run8 head=792b373 origin/main=792b373 verdict=SHIPPABLE-WITH-GUARD-DF-018 install=PASS(verified-teardown,not-rerun) perf=verified-not-retimed rows=GUARD-DF-018,GUARD-DF-019

## Correction to the records-only closeout

The preceding duplicate-fire record incorrectly stated that the checkout had no Git-operation markers. Independent verification on the later re-fire found `.git/REBASE_HEAD` containing `256b002be1d6d726a0606aa0881bed0193c5a5aa`; no `rebase-merge`/`rebase-apply` directory, unmerged paths, or process holding this checkout was found, so the marker appears stale, but the dogfood marker rule still makes the live checkout an unmeasured boundary. The marker was not removed or repaired.

This correction does not invalidate the already-committed run-8 product, performance, or install evidence: that evidence was measured at `4d4a2954affaceeb6dbc4d22824424e32ab6e900`, and the current product-only diff to `b11864b5830c981f80941ab871bd934895ecd198` is empty. It does mean the prior statement that the fire independently verified a marker-free checkout is withdrawn. No product use, benchmark, install battery, or new board row was run on this re-fire. `GUARD-DF-018` and `GUARD-DF-019` remain the two existing live findings; their committed row and paired `task_created` event counts remain 1 each; `boardctl validate` remains `RESULT: OK (73 warning(s))`, rc 0.

- ch:trace tick=guard-dogfood-2026-11-14-23-17-36 duplicate=yes correction=stale-REBASE_HEAD live-checkout=unmeasured rows=GUARD-DF-018,GUARD-DF-019

## Third fire — the cap gap closed, and the cost behind it

Prior fires of this id were records-only because run 8 had already filed its two findings. This fire
did new work instead of re-recording: it exercised the one thing run 8 explicitly named as unexercised
("the 8 MiB HTTP body cap was exercised only up to 22.6 KB; the rejection [above the cap was not
exercised]") and put a number on the cost of large bodies.

All measurements were taken against a preserved run-8 binary built from the tested commit `4d4a295`
(sha256 `cdd690fcdd3a29314e3864558f72ed878d96546d9acc028652c6484a81392f2b`, served on
`127.0.0.1:18791`), never against the live checkout — which still carries the stale `.git/REBASE_HEAD`
marker recorded in the correction above and therefore remains an unmeasured boundary. Benchmarks ran in
a scratch `git archive 4d4a295`-shaped export of HEAD under `.hermes/cache/scratch/`, never in the repo.

Cap boundary, both sides, egress at product default: 7 MiB -> 200 allow (1394.8 ms); exactly 8,388,608 B
-> 200 allow, score 0.08 (1208.1 ms); 8,388,609 B -> **400 `bad request: unexpected EOF`** (28.8 ms);
9 MiB -> 400, same body (26.1 ms). The boundary is exact at `8<<20` (no off-by-one) and
`connectors/README.md` already documents truncation->400, so the defect is what the caller is told: an
unnamed, truncated-JSON-shaped answer for a size problem, delivered in 28 ms with no classification
attempted. That is DF-G4, which `GUARD-DF-008(c)` claimed fixed when it was closed `complete`.

Attribution before filing: with `GUARD_EGRESS_ENABLED=false` the same binary still takes 1181-1205 ms at
8 MiB but 1 ms at 41 B, so the cost is local and runs before `Classify` (`guard.go:187` ->
`guard.go:190`). An in-export benchmark (`-benchtime 5x`, AMD Ryzen 7 7840HS, loadavg 29.5):
`NormalizeAll` 793.8 ms/op at 8 MiB (10.57 MB/s) and 2.46 ms/op at 24 KB (10.02 MB/s) — constant
throughput, so per-byte linear work rather than a threshold artifact. `ExpandBase64` 371.1 ms plus
`ExpandHex` 317.0 ms are 87% of it; rot13/fold/invisible account for the remaining ~139 ms and
`ExpandURL` short-circuits (0.15 ms). Hot path: `normalize.go:89` `[A-Za-z0-9+/]{16,}={0,2}` scanned over
the whole body at `:93`, and `normalize.go:115` `\b[0-9a-fA-F]{16,}\b` scanned at `:119`; `maxExpanded`
(`:47`) bounds only the appended decoded text, nothing bounds the scan over the input.

Judged worth filing because a user feels it: 1.2-1.5 s per call on document-length input, and
`GUARD-DF-018` says the verdict only reads the first ~19 KB — ~99.8% of that scan is invisible to the
classifier. Filed with a 5x before/after target and a "do not just delete the scans" note, because
encoding expansion is the product's stated value; the defect is the unbounded scan, not the feature.

Install leg: **not re-run, and not a silent pass.** This tick's earlier fire left a complete ephemeral
bunker battery at `/tmp/bunker-qa-evidence-20261114T234313Z-926128.jsonl` on `bunker-mvp`
(instance `08c686df`): `launch OK`, `fresh-install OK`, `tests OK`, `lint INFO`, `docker_build N/A`,
`compose_config N/A`. The product tree is byte-identical to the commit that battery graded
(`git diff --stat 4d4a295..HEAD -- . ':!.coding-hermes' ':!docs'` is empty), so re-running would grade
the same bytes; re-running was declined on that ground, not for lack of a host.

Evidence document: `docs/dogfood/2026-11-14-run8-tick3-cap-and-perf.md`.

- ch:trace tick=guard-dogfood-2026-11-14-23-17-36 duplicate=no new-evidence=yes head=c62f815 product=4d4a295-invariant verdict=SHIPPABLE-WITH-SECURITY-SHAPED-ROUGH-EDGE install=PASS(inherited,byte-identical-product,not-rerun) perf=MEASURED-normaliz-0.79s@8MiB rows=GUARD-DF-020,GUARD-DF-008-R2 cited=GUARD-DF-018,GUARD-DF-008,DOC-27

## Fourth fire — the perf premise re-derived, and the criterion it broke

Tick `guard-dogfood-2026-11-14-23-17-36`, fourth executor. HEAD `0a5a791` (tree `e130718b`),
product tree byte-identical to run 8's tested commit `4d4a295`. Unlike the records-only fire above,
this one produced a fresh measurement artifact and one new row.

Why: the third fire put a 5x performance target on the board (`GUARD-DF-020`) with a baseline it
recorded only in prose — the `ns/op` output was never captured as a file, while its HTTP artifacts
(`cap/cap-probe.json`, `ext/*.log`) were. A number a worker is graded against should be
re-derivable from an artifact rather than from a paragraph.

What: exported HEAD with `git archive` into scratch (repo tree untouched; `normalize.go` sha256
`02f9a4f3…` identical to HEAD) and re-ran the benchmark at loadavg `24.77` against the recorded
`29.5` — `NormalizeAll` 8 MiB **761.6 ms / 11.01 MB/s** (recorded 793.8 ms / 10.57 MB/s), 24 KB
2.31 ms / 10.64 MB/s, `ExpandBase64` 356.5 ms, `ExpandHex` 257.6 ms, `ExpandURL` short-circuits.
Throughput stays flat across a 340x size change, so the premise stands and the cited anchors
(`normalize.go:89` scanned at `:93`, `normalize.go:115` scanned at `:119`) are correct at HEAD.
base64+hex measured 80.6% of `NormalizeAll` here versus the recorded 87% — load noise, not a
correction, so no cell was filed for it.

The defect: `GUARD-DF-020`'s acceptance criterion says its 10.57 MB/s baseline was "measured at
loadavg <5". It was not — the same row's `reasoning` records loadavg 29.5, and this fire measured
24.77. A worker who follows that clause re-measures on an idle host, gets a different baseline, and
cannot say whether a 5x claim passed. Wording-only fix: an explicit `>= 52.9 MB/s` target plus a
loaded-host and an idle-host before/after table in the fix commit. Filed as `GUARD-DF-021` (P4,
complexity 1). `GUARD-DF-020` was not edited — this lane never rewrites board rows.

Not re-run, and why: the HTTP cap boundary (the tick-3 artifact `cap/cap-probe.json` is intact and
its four cases match the published table byte-for-byte — 7 MiB 200 in 1394.8 ms, 8,388,608 B 200 in
1208.1 ms, 8,388,609 B 400 `bad request: unexpected EOF` in 28.8 ms, 9 MiB 400 in 26.1 ms;
re-reading a preserved artifact needs no second server) and the install leg (ephemeral bunker
instance `08c686df` graded a byte-identical product tree; the instance is gone and its teardown
held, so a fresh launch would grade the same bytes). Inherited, not silently passed.

Teardown: no `guardd` process, no listener on `18790`/`18791`/`8768`, no agent `08c686df`; the only
Go workload was the 8.3 s benchmark in scratch.

Evidence document: `docs/dogfood/2026-11-14-run8-tick4-bench-reproduction.md`.

- ch:trace tick=guard-dogfood-2026-11-14-23-17-36 duplicate=no new-evidence=yes head=0a5a791 product=4d4a295-invariant verdict=SHIPPABLE-WITH-SECURITY-SHAPED-ROUGH-EDGE install=PASS(inherited,byte-identical-product,not-rerun) perf=REDERIVED-normalize-0.76s@8MiB rows=GUARD-DF-021 cited=GUARD-DF-020,GUARD-DF-018,GUARD-DF-008-R2

## 2026-11-14 — run 8, tick 5: policy wiring + operator contract (`guard-dogfood-2026-11-14-23-17-36`)

Verdict: **WORKS, WITH ONE SILENT POLICY NET THAT ISN'T WIRED.** Measured on a clean `git archive`
export of HEAD `0a5a791` (tree `e130718b`) with a private `HOME` per arm, egress disabled, loadavg
5.5/7.8/16.3 on 16 cores. Used the product the way an operator would: served it, ran the runbook's
own claim table, installed the release artifact, and drove per-source `Overrides` over HTTP.

The product does what the docs say. Every `docs/operations.md` cell matched: no-token refusal
(`rc=1`, "refusing to serve an unauthenticated /check"), tokenfile `0640` refused
("mode 0640 is too open"), `GUARD_TOKEN` winning over `-token` in a request carrying the argv token
(401), `:port` → loopback, the full /check status ladder (401/401/405/405/400/400/400/200), policy
startup failures naming the inverted pair (`review_at=0.9 > block_at=0.5`), `-policy` beating
`GUARD_POLICY`, and the egress kill switch returning 200 with an in-band errored `block` result.
The release artifact downloaded (2,935,335 B in 0.76 s), its checksum verified against the real file
hash (not just `--ignore-missing`), extracted and ran. `make test` 4.47 s rc=0; `make docs-check`
0.55 s PASS.

The finding is the net around `Overrides`. `cmd/guardd/main.go:155` is the only non-test
`guard.Guard{...}` construction and it never sets `Warnf`; `ValidateOverrideKeys` has zero non-test
callers. So the dead-override-key warning that `policy.go:88` advertises cannot fire for anyone
running the binary — verified by a typo'd `overrides:{"webhook":…}` policy run with `-source
github_pr` (stderr: only "policy loaded … 1 override(s)") and by the same policy served via
`GUARD_POLICY` (no warning line), while a correctly keyed request *does* move the bands
(0.85/0.50 → 0.55/0.40, `override_source: "webhook"`). The library warning itself is correct —
its tests pass — so this is wiring, not behaviour. An operator who mistypes an override key
silently keeps the looser base bands. Filed as `GUARD-DF-022` (P3, complexity 2).

Same shape from the docs side: `docs/operations.md` documents the /check body as content-only, but
the handler accepts `id`, `source`, `channel`, `meta`, and `source` is what selects an override —
so the runbook never shows how to make the README's own per-source Quick Start fixture fire, and
unknown JSON fields are silently ignored (`{"content":"x","bogus":1}` → 200). Filed as `DOC-31`
(P4, complexity 2).

Performance (Step 2b): cold CLI start 10–12 ms (n=5), clean `make build` 298 ms, server ready 12–13 ms,
`/healthz` 0.274 ms, `/check` 0.71–1.23 ms at 4 KiB, 6.4–8.0 ms at 64 KiB, 96–99 ms at 1 MiB
(~10.5 MB/s). **No PERF row filed**: nothing here is a new user-felt cost, and the one path that is
(1 MiB ≈ 0.1 s → 8 MiB ≈ 0.79 s at the cap) is already `GUARD-DF-020`. Boundary stated in the report:
egress disabled means classifier latency is not measured, and the host was moderately loaded.

Install leg: **ran and passed**, not skipped. `bunker-qa.sh launch … --server bunker-mvp` →
`agent=791648f6`, detached battery, 24 evidence rows: fresh-install OK (public-origin clone,
`go build ./...`), tests rc=0, `act` 1 job green, upgrade from v0.1.0 clean, 3 GiB memory cap
survived, clean `rc=2` on missing config, no network dependency; lint INFO (golangci-lint absent on
the agent). The old `GUARD-DF-009` render blocker is gone in the current script (escaped
`UI_PORTS`). Teardown verified: the agent home is gone from `bunker-mvp`.

Not re-run, and why: the HTTP cap boundary (preserved tick-3 artifact `cap/cap-probe.json`
already matches the published table) and the classifier-latency path (no credential; this lane
never mints one). The CLI `rc=2` half of the egress contract was not reproduced locally — the
blocked shell invocation was not retried, and the bunker `chaos-errorpath` cell supplies the
observation instead.

No code, policy or board row was changed; no row closed; ids came from `board_next_id.py`
(`GUARD-DF-022`, `DOC-31`, both free at file time); `boardctl validate` run after filing.
Left behind: `docs/dogfood/2026-11-14-run8-tick5-override-wiring.md` and this entry.

- ch:trace tick=guard-dogfood-2026-11-14-23-17-36 duplicate=no new-evidence=yes head=0a5a791 perf=measured-no-finding install=PASS(rerun,bunker-mvp,agent=791648f6) rows=GUARD-DF-022,DOC-31 cited=GUARD-DF-020,GUARD-DF-019,GUARD-DF-009,GUARD-DF-007,DOC-17

## 2026-11-18 — run 9, tick 1: product surface re-walk after GUARD-DF-020/R2 (`guard-dogfood-2026-11-18-00-54-47`)

Product changed since the last record (59 commits, head now `7c08590` = origin/main), so this was a
real re-walk, not a records-only fire. Headline: the closed normalization perf work is live and
user-felt — `/check` at 1 MiB went from the 96–99 ms recorded on 11-14 to **7.8 ms median** (n=10,
~12×), 64 KiB from 6.4–8.0 ms to 0.90 ms. No PERF row: a win nobody can feel as a cost is not a
finding, and the rows that own it (`GUARD-DF-020`, `GUARD-DF-020-R2`) are already complete.

Walked, all at `7c08590`: build (`make build` 0.47 s, `-version` → `guardd v0.2.0`); documented
fail-closed scratch-HOME CLI reproduction (exact verdict, rc=2, cold < 10 ms); real classification
via the documented key-discovery order (`allow`, score 0.2, `typesafe/jev-1.13-20260917`,
`duration_ms: 720`, `cost_usd: 0.0000278`; warm CLI 0.40–0.58 s over 5 runs — the wall is the
classifier round-trip); serve mode with the Quick Start policy fixture (webhook override moves bands
0.85/0.50 → 0.55/0.40 live, `override_source` carried; `/healthz` and `/health` both 200 < 2 ms);
HTTP auth negative (401 whose message names `X-Operator-Token`); **egress-ON `/check` classification
(580 ms, $0.0000276)** — the leg the 11-14 fire recorded as unverified is now measured; Python
connector installed as a real consumer in a fresh venv (`guard-agent-client 0.2.0`; library,
console script exit-code contract, routeless-verdict fail-closed, `fail_mode="open"` flagged-not-
trusted); documented error paths (invalid policy JSON → rc=1 with parse position; tokenfile 0640 →
rc=1 with the chmod remedy in the message); inert-override startup warning corroborates closed
`GUARD-DF-022`.

Filed: `GUARD-DF-023` (P4) — the `-version` flag is absent from README, `skills/guard-usage`, and
`docs/operations.md` (grep census 0/0/0); discoverable only via `-h` and CHANGELOG. Id minted from
the canonical origin/main board copy (`board_next_id.py`, free at file time; `boardctl validate`
OK with 76 pre-existing warnings after filing). Not re-filed, owned elsewhere: unknown-JSON-field
tolerance (`DOC-31`); v0.1.0-as-latest released-surface state (`REL-GUARD-007`, owner-gated).

Install leg: `bunker-qa.sh launch … --server bunker-mvp --keep` → `agent=7a98d021` launched
01:06 UTC, disk preflight 78.5% (< 94%); battery ran detached, harvested at close (see tick
report). Pre-existing agent `f6a5f076` (created 00:08, before this tick) is not this fire's and was
left untouched. Left behind: `docs/dogfood/2026-11-18-run9-tick1-integration.md` and this entry.

- ch:trace tick=guard-dogfood-2026-11-18-00-54-47 duplicate=no new-evidence=yes head=7c08590 verdict=SHIPPABLE install=PENDING-AT-WRITE(bunker-mvp,agent=7a98d021,harvested-at-close) perf=measured-no-finding-12x-1MiB rows=GUARD-DF-023 cited=GUARD-DF-020,GUARD-DF-020-R2,GUARD-DF-022,DOC-31,REL-GUARD-007
