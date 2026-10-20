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
