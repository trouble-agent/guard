# Guard dogfood findings — 2026-10-13 (run 2, tick guard-dogfood-2026-10-13-14-52-46)

Real-use pass over guard at HEAD b23aad1: CLI (stdin, -content, exit codes),
HTTP serve (auth, /healthz, error shapes, 60KB body), Go library consumer
(replace-directive scratch project), Python connector (pip install on fresh
bunker, Guard class, `guard` CLI). Fresh-machine install leg on bunker-mvp
agent `guard-dogfood` (las-bunker-03 + 3 other bunkers unreachable over
tailscale). Findings below are SUGGESTED BOARD ROWS — this lane is read-only;
the foreman files/works them.

- DF-G1 [P3] HTTP /check request-shape doc gap: doc comment in cmd/guardd/main.go:7
  says `POST /check {input}` but the handler decodes guard.Input whose JSON field
  is `content` (README:178 has it right). A user writing `{"input":...}` per the
  main.go header gets 400 "content is required" with no hint of the right key.
  Fix: correct main.go:7 to {content}.

- DF-G2 [P4] CLI stdlib JSON error leaks decoder internals: `printf 'not json' |
  guardd` prints `bad request: invalid character 'o' in literal null (expecting
  'u')` (HTTP body /check path identical). Cosmetic but the only non-obvious
  error a first-time user hits; suggest mapping decode errors to
  "bad request: body is not valid JSON".

- DF-G3 [P3] CLI stdin mode has no size cap: cmd/guardd/main.go:121
  `io.ReadAll(os.Stdin)` is unbounded while the HTTP path caps at 8MiB
  (io.LimitReader, main.go:191). A pipe fed a multi-GB stream will buffer
  unbounded in memory before classification. Suggest the same 8MiB cap + a
  clear error, or document the asymmetry.

- DF-G4 [P4] Malformed >8MiB body returns 400 "bad request: unexpected EOF"
  (verified with an 18MB body): correct refusal, but the message does not name
  the size limit. Suggest including the 8MiB cap in the error text.

- DF-G5 [P5] Corpus doc example is a truncated payload: docs/threshold-calibration.md:21
  shows mq-01's content as "aUdub3Jl..." (the real corpus.jsonl line is a full
  base64 injection). A user replaying the doc example verbatim gets a verdict,
  but not for the payload the doc describes. Cosmetic docs fix.

## Verified-holding (no rows)
- Fail-closed egress refusal (GUARD_EGRESS_ENABLED=false): block+quarantine,
  errored=true, rc=2 — matches DOC-8/ADOPTION.md exactly.
- Base64 + ROT13 chains: direct injection 0.98-0.99 block; rot13-wrapped
  injection 0.91 block WITH load_bearing_normalizations=[rot13] (GUARD-DF-002
  fix re-proven); plain ROT13 benign text delivers with no load-bearing attrib.
- Auth: missing/wrong token 401 before classification; /healthz open; token
  precedence env>tokenfile>argv with ps warning.
- Policy override: webhook source override accepted (BlockAt 0.55 shape).
- Fresh-box install (bunker-mvp agent guard-dogfood): clone→go 1.26.6
  auto-download→build 33s→fail-closed first run (rc=2, correct)→go vet+tests
  green→pip install connectors/python→`guard` CLI works end-to-end (rc=3
  connection-refused to absent server = correct). Install ≈3min cold.

## Perf (Step 2b)
- HTTP /check warm: 394.1ms ± 12.7ms (10 runs) — provider (Jev API) bound;
  duration_ms field agrees (350-450ms typical).
- CLI with real classification warm: 638ms ± 112ms (20 runs) — same provider
  floor, no local hot spot worth a PERF row.
- Local-only path (egress disabled, no classify): ~4-8ms — nothing slow.
- Verdict: no PERF row — everything slow is the remote classifier, which is
  the design, and the cost is visible per-call in cost_usd (~$0.000027).
