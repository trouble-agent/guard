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
