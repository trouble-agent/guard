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
