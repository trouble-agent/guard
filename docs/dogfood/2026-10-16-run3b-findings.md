# Guard dogfood findings — 2026-10-16 (run 3b, tick guard-dogfood-2026-10-16-15-45-07)

SECOND run under the same tick (sibling run 3 = 7a0aca3, agent guard-df-1016, DESTROYED, evidence
docs/dogfood/2026-10-16-run3-findings.md). Rule
applied: same-tick sibling => continue the row namespace (DF-G6..G9 taken),
change the ANGLE, do not re-sweep a covered surface. Run 3 covered: the
-channel/-source no-op surface (DF-G6), threshold jitter (DF-G7), released-
artifact staleness (DF-G8), PEP 668 connector friction (DF-G9), and a hand-
walked fresh-box install (44s). Run 3b drove the surfaces run 3 did NOT
touch, all at HEAD aeb2f80, tree clean, no runtime code changed this delta.

## Suggested row

- DF-G10 [P2] `make docs-check` — the repo's own README-replay drift gate on the
  docs side — verifies a STALE hardcoded Go version. Makefile:106-107:
  `print-go-directive:\n\t@echo 1.26.6` is a literal, but go.mod declares
  `go 1.26.9` since the SEC-GO-1269 bump (a1b8ff3, 2026-10-14). At HEAD the gate
  prints "docs-check: go directive = 1.26.6" and PASSes (run verbatim, docs-check
  seen PASS with that line) — so the gate built to catch a stale "Go 1.26.6 in
  README" claim (would have caught README-2, per its own Makefile comment) now
  rubber-stamps the stale claim it exists to catch. The drift is live in the tree:
  README.md:12 "the repo itself declares Go 1.26.6 in go.mod" (false),
  skills/guard-usage/SKILL.md:14 and :84 "go.mod declares go 1.26.6" (false).
  The go 1.26.9 security bump's docs fallout is exactly the drift the gate was
  written to fail on. Fix direction (small): make print-go-directive read go.mod
  (`grep '^go ' go.mod | awk '{print $2}'` or `go mod edit -json -godebug`-free
  parse) instead of echoing a constant, then correct README.md:12 +
  SKILL.md:14,84 to 1.26.9 — one source of truth, four touched files.
  Evidence: docs-check output ("directive = 1.26.6") vs `go mod -n` line in
  go.mod:3; Makefile:106-107.

## Verified-holding at HEAD (re-proofs + discrimination, no new rows)

- DF-G1 (run 2) re-proven: HTTP body {"input":...} per the main.go:7 doc comment
  still yields 400 "content is required" (main.go:7 unchanged: still `{input}`);
  correct {"content":...} accepted, 200.
- DF-G4 (run 2) re-proven: 9MiB body -> 400 "bad request: unexpected EOF"
  (message still does not name the 8MiB cap).
- DF-G5 (run 2) re-proven unchanged: docs/threshold-calibration.md:20-21 still
  shows the truncated "aUdub3Jl..." mq-01 example; the real corpus.jsonl row
  carries the full base64 payload (SUdNT0RF... SUdOT1JFIEFMTCBQUkVWSU9VUyBJTlNUUlVDVElPTlM=).
- DF-G2 (run 2) NARROWED, premise corrected: the raw decoder-internals error
  (`invalid character 'o' in literal null (expecting 'u')`) is an
  HTTP-BODY-ONLY surface — `/check` body "not json" -> 400 with that text;
  the CLI maps "not json" on STDIN to plain text and classifies it rc=0
  (allow, score 0.10). Run 2's row said CLI stdin path identical — it is not;
  fix scope is the HTTP decode arm only (main.go's http handle path).
- Classification quality (real use, same axes as runs 1-2): direct instruction
  injection block 0.98-0.99 (quarantine), ROT13-wrapped injection block with
  rot13 normalization, benign quarterly-report text allow -> review
  (0.51, matching the documented 11-of-33 benign-review calibration band),
  base64 mq-01 corpus payload block 0.99 (instruction_injection).
- Exit codes this run: 0 on classified deliver; 2 on fail-closed (env -i,
  no key -> `guard_error (fail-closed): no openrouter api key available`,
  constraints all false) — QA-GUARD-002 contract intact at aeb2f80; serving
  mode exits only on kill.

## `make docs-check` leg (never driven by runs 1-2, passed but see DF-G10)

docs-check replays the README's build + stdin example live (this tick's run:
build OK, "printf 'hello, world' | ./guardd" replay OK, fail-closed JSON
matches README, empty-stdin rc=2 OK — PASS). The gate works for the surfaces
it checks; DF-G10 is its stale version CONSTANT, not the replay mechanism.

## Standard bunker-qa.sh battery leg (bunker-mvp agent 102dffc5, TTL 4h, destroyed on collect)

sibling run 3 hand-walked a fresh machine; this leg ran the repo-agnostic
standard battery against the same HEAD (aeb2f80, ff-only):
- fresh-install OK on the ephemeral agent: agent's distro Go was 1.26.5
  (< the new 1.26.9 floor); go.mod's toolchain directive pulled
  "go1.26.9 (linux/amd64)" itself — the security bump's minimum works on a
  fresh box without hand-praying (none of the PREREQ reading run 1 needed).
- ci-pass OK: act, 1 jobs green, rc=0.
- upgrade OK: release tree v0.1.0 -> HEAD reinstall clean.
- chaos-resource PASS: suite survives 3G memory cap.
- chaos-errorpath OK: clean error on missing config (rc=2).
- chaos-disconnect UNVERIFIED: suite produced no output under network cut
  (no-test-path marker) — battery cell gap, not a product verdict.
- chaos-* docker/state cells N/A (no compose file, no state files).
- 19 evidence rows; agent destroyed in collect (bunker list clean; the two
  running agents on mvp predate this tick and belong to other sessions).
INSTALL_SECONDS (battery, wall 15:58Z spawn -> 16:10Z collect) ≈ 9 min for the
full battery; the plain build inside it was seconds (toolchain resolve:
go1.26.9 download).

## Perf (Step 2b, coding-hermes-perf law; HEAD binary, same workload as run 3)

- Headline CLI classification (warm, with live Jev key, 10 hyperfine runs):
  405.2 ms ± 68.7 ms (range 344.1-559.3; user 15.1 ms, system 9.0 ms) — the
  wait is the remote decision API round-trip, not the binary.
- Fail-closed (no key) warm: 5.1 ms ± noise (3 runs); first-run cold in a
  stripped env: 193.8 ms total incl. first process+fs-cache (the only thing
  "cold" does is resolve a missing key fast — the binary does not classify).
- No PERF row: same verdict as runs 1-3 — the only felt cost is the designed
  classifier HTTP round-trip; the local floor is ~3-5 ms. Nothing locally slow
  enough for a user to feel.

## Board note

Board state at this run's read: 45 unique ids, 0 open rows (census both run 3
and this run agree; run 2's duplicate-id caveat closed by 588a5dd). DF-G1..G10
are suggested rows in this doc namespace; none are filed on the board — the
lane contract is read-only; the foreman files/works them.

## Verdict

SOUND (consistent with runs 1-3). This run's new finding is DF-G10 (the drift
gate verifying a stale version constant — small, cheap fix, and the only
defect found in the docs-check surface). Everything else re-proofed green;
installability re-proven two ways in one tick (sibling hand-walk + standard
battery), no SKIPPED-install-bunker row needed.
