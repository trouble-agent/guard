# Guard dogfood findings — 2026-10-16 (run 3, tick guard-dogfood-2026-10-16-15-45-07)

Real-use pass over guard at HEAD aeb2f80 (delta vs run 2's b23aad1: docs
DOC-12/13/14, README-3/4/5, doc-comment/docs-parity fixes, SEC-GO-1269 go
1.26.6→1.26.9 bump; **no runtime code changed**). Angle this run = the
**`-channel` flag surface** (documented 10-14 by DOC-13, never dogfooded) +
released-artifact check (v0.1.0, 2026-10-02) + fresh-box install re-proof.
Findings below are SUGGESTED BOARD ROWS — this lane is read-only; the foreman
files/works them. Continues the DF-G namespace (DF-G1..G5 run 2, c1fa6eb).

## Suggested rows

- DF-G6 [P2] `-channel`/`-source` are accepted and silently discarded —
  README:212 (DOC-13, 3d552dd) documents
  `guardd -content "$message" -source github_pr -channel crier -pretty` with
  "# -channel: classifier label", and `guard.Input` carries Source/Channel
  JSON fields (guard.go:71-72) — but `Guard.Check` (guard.go:146-185)
  classifies `in.Content` ONLY: `Classifier.Classify(ctx, content)`
  (jev.go:148) never receives the labels, and `Result` has no
  channel/source fields (probe: neither string appears in the CLI or HTTP
  result JSON). Proven with a 5-variant × 5-content matrix (bare /
  -channel crier / -channel task-router / both / -source github_pr):
  decisions identical across variants for every content; score deltas
  0.01-0.04 = model noise, not label effect. An integrator passing
  `-channel crier` believes the classifier is channel-aware (README calls it
  a "classifier label"); it is a no-op. Fix direction: feed channel/source
  to the classifier as context (accuracy upside on crier's known channels)
  OR reword README:212 + SKILL.md:30 to "client-side metadata, ignored by
  the classifier incurring no added accuracy today".

- DF-G7 [P3] verdict jitter straddles the review threshold on borderline
  payloads: one secret-bearing payload scored 0.46-0.53 over 9 identical
  replayed calls (4 bare: 0.50/0.50/0.52/0.51; 5 labeled variants
  0.46-0.53), ROUTING deliver (0.46-0.49) vs review (0.50-0.53) 4-vs-5
  across runs — Jev model noise at the 0.5 review threshold, not
  channel-correlated (jitter appears bare too, isolated in
  borderline_bare_x4). A real consumer retrying or fanning out the same
  borderline message sees route flips between quarantine-boundary
  destinations. Fix direction: threshold band (e.g. review from 0.55) or
  documented jitter envelope in docs/threshold-calibration.md.

- DF-G8 [P1] the released artifact (v0.1.0, 2026-10-02) ships its own
  README and none of HEAD's user-facing docs: exit-code contract
  QA-GUARD-002 (rc=2 on guard_error, 8ef7ffe), GUARD_EGRESS_ENABLED
  kill-switch (DOC-8, 4f6958f), `-channel` flag examples (DOC-13,
  3d552dd), Makefile targets + docs-check gate (DOC-12, 6dcdf69),
  corrected deploy/install FAQ (README-3/4/5), plus the SEC-GO-1269 go
  1.26.9 security bump (a1b8ff3, GO-2026-6617 HTTP/2 HPACK race fix) are
  all post-tag (`git tag --contains` empty for each; merge-base vs v0.1.0
  = not ancestor). A user following the released tarball's README does not
  learn the egress kill-switch exists, the fail-closed exit code, or that
  they are running a binary with a known net/http vuln fixed 12 days
  later. Fix direction: cut v0.1.1 (GoReleaser flow already proven at
  v0.1.0) — the stale README is the artifact, not a doc task.

- DF-G9 [P2] `pip install connectors/python` fails verbatim on a current
  fresh box (PEP 668): documented connector install on the fresh agent
  (Ubuntu-derived, Python 3.12.3) exits rc=1 with
  `externally-managed-environment`; rescue with
  `pip install --user --break-system-packages` succeeds (rc=0) and the
  `guard` CLI works. Run 2's agent tolerated the bare form (run-2 design
  notes claim fresh-install green); today's vanilla agent does not — the
  docs should name the PEP 668 footnote next to the connector install
  line (connectors/README.md), or ship a venv-based quickstart. No code
  fix implied; pure docs.

- DF-G4 (run 2) re-proven at HEAD on the fresh box: >8MiB body → 400
  "bad request: unexpected EOF" and no-key first run → rc=2 + exact
  fail-closed verdict shape (`guard_error (fail-closed): no openrouter
  api key available`, constraints all-false). Premise unchanged; no new
  row. Guard's 401/world-visible shape also re-proven on the live HTTP
  leg (401 wrong token, /healthz 200 open, `{input}` 400 "content is
  required" — DF-G1's main.go:7 doc-comment still says `{input}`).

## Verified-holding this run (no rows)
- Classification parity where it matters: plain benign delivers,
  direct instruction injection blocks 0.97-0.98, ROT13-wrapped injection
  blocks WITH `load_bearing_normalizations:["rot13"]` (GUARD-DF-002 fix
  holds at aeb2f80), determinism on clear-cut payloads (inject ×3 =
  block/0.98 every time).
- Exit-code ladder at HEAD: 0 classified / 0 block-but-classified / 2
  fail-closed (egress off) / serving always 0 — QA-GUARD-002 contract intact.
- Fresh-box leg (bunker-mvp agent guard-df-1016, DESTROYED after):
  clone from public origin →commit aeb2f80 (main tip matches) →
  `go build ./cmd/guardd` 41s cold (go1.26.9 auto-downloaded onto the
  agent's go1.22.2 per go.mod toolchain directive) → fail-closed smoke rc=2
  with the exact documented verdict → a deliberately-injected sk-or-shaped
  probe key (not a real secret) rc=2
  "key rejected: http 401". INSTALL_SECONDS=44 (wall incl. clone).
  Residue recheck post-destroy: no orphan units/homes (bunker list clean
  95s after destroy).
- DF-G1 unresolved premise holds (main.go:7 still `{input}`); DF-G2/G3/G5
  untouched by the delta, not re-swept (out of this run's scope).

## Perf (Step 2b, coding-hermes-perf law; HEAD aeb2f80, bin=/tmp build of HEAD)
- Headline CLI classification warm: 391.0ms ± 39.0ms (hyperfine 20 runs,
  range 350.8-496.0) — provider-bound (Jev API); agrees with run 2's
  394ms ± 12.7 (same envelope; runtime code unchanged so no drift expected).
- Cold single run (~fresh process, first classification): ~0.85s wall,
  user 0.01s — all wait is the remote classifier.
- Binary startup only (-help): 2.8ms ± 0.4ms (20 runs).
- Local-only floor (GUARD_EGRESS_ENABLED=false, no classify): 3.0ms ± 0.5ms.
- Fresh-machine build from clone: 41s cold incl. go1.26.9 toolchain
  auto-download.
- **No PERF row**: the only visible cost is the designed remote
  classifier round-trip (visible per-call in cost_usd ≈ $27e-6); local
  floor is 3ms — nothing locally slow enough to feel. The go 1.26.9 bump
  did not move the number (no regression from SEC-GO-1269 verified warm).

## Board note
Run 2's caveat ("tasks.jsonl tail shows DOC-2..DOC-10 'done' rows
duplicating earlier 'pending' rows — status-by-id ambiguous") is now
RESOLVED by the foreman (commit 588a5dd, chore(board): reconcile duplicate
DOC-9/DOC-11 rows, 2026-10-14): last-occurrence-per-id census over the
current board = 45 unique ids, 0 rows in an open state. Nothing to file.

## Verdict
SOUND (✅ at the run-2 level of assurance; value real, usability good,
honesty ladder intact). The sharp edges this run found are docs-vs-release
(DF-G8) and the silently-discarded label args (DF-G6) — cheap to fix, no
runtime defect found in the delta.
