# guard — run 8, third fire of the tick: the 8 MiB body cap, and what it costs

Tick `guard-dogfood-2026-11-14-23-17-36` (third fire of this id; `b11864b` and `c62f815` are records-only prior fires).
HEAD `c62f815` — product tree byte-identical to the tested commit `4d4a295`
(`git diff --stat 4d4a295..HEAD -- . ':!.coding-hermes' ':!docs'` prints nothing), so the
bunker install leg collected for this tick earlier still describes this exact product.

Preserved run-8 binary used for every probe:
sha256 `cdd690fcdd3a29314e3864558f72ed878d96546d9acc028652c6484a81392f2b`, started on
`127.0.0.1:18791`. Probes ran against that binary, not the live checkout, so no measurement
below is credited to a tree that was not tested.

## What run 8 named as unexercised

Run 8's own gap line: *"The 8 MiB HTTP body cap was exercised only up to 22.6 KB; the
rejection [above the cap was not exercised]."* This fire closed that gap from both sides.

## The cap, exercised at the boundary

POST `/check`, `Content-Type: application/json`, `X-Operator-Token` set, body
`{"content":"A…"}` of exactly N bytes, egress left at the product default:

| body bytes | status | response | wall |
|---|---|---|---|
| 7,340,032 (7 MiB) | 200 | `decision=allow route=deliver score=0.07` | 1394.8 ms |
| 8,388,608 (exactly 8 MiB) | 200 | `decision=allow route=deliver score=0.08` | 1208.1 ms |
| 8,388,609 (8 MiB + 1) | 400 | `bad request: unexpected EOF` | 28.8 ms |
| 9,437,184 (9 MiB) | 400 | `bad request: unexpected EOF` | 26.1 ms |

The boundary itself is exact at `8<<20` — no off-by-one, and `connectors/README.md`
(`Limits`) already documents the truncation→400 behaviour honestly. What the caller is told
is the defect: `bad request: unexpected EOF` reads as a malformed upload, not "you exceeded
8 MiB", and it arrived after 28 ms, i.e. no classification was attempted. Filed as
`GUARD-DF-008-R2` (recurrence — `GUARD-DF-008(c)` had claimed this error text fixed and was
closed `complete` without it).

## Where the time goes

Same binary, `GUARD_EGRESS_ENABLED=false` (nothing leaves the process, so no network is in
the number), 3 runs each, host 16 cores at loadavg 25.1:

| body | egress on | egress off |
|---|---|---|
| 41 B | 536 / 322 / 290 ms | 1 ms |
| 24 KB | 372 / 335 / 341 ms | 5 ms |
| 8 MiB | 1410 / 1444 / 1491 ms | 1195 / 1192 / 1181 ms |

So the ~1.2 s at 8 MiB is local work paid *before* `Classify` (`guard.go:187 NormalizeAll` →
`guard.go:190 Classify`), not classifier latency. With 41 B the fixed ~300 ms is the hosted
classifier and the local part rounds to zero.

Go benchmark on a scratch `git archive HEAD` export (never the repo), AMD Ryzen 7 7840HS at
loadavg 29.5, `go test -run XXX -bench . -benchtime 5x`:

```
BenchmarkNormalizeAll_8MiB          793845975 ns/op   10.57 MB/s
BenchmarkNormalizeAll_24KB            2455412 ns/op   10.02 MB/s
BenchmarkExpandBase64_8MiB          371075072 ns/op   22.61 MB/s
BenchmarkExpandHex_8MiB             316984602 ns/op   26.46 MB/s
BenchmarkExpandRot13_8MiB            50781443 ns/op
BenchmarkExpandFoldHomoglyphs_8MiB   49470664 ns/op
BenchmarkStripInvisible_8MiB         38215482 ns/op
BenchmarkExpandURL_8MiB                147942 ns/op   (short-circuits)
```

Throughput is constant across a 340× size change — per-byte linear work, not a threshold
artifact. The five real passes sum to 827 ms against `NormalizeAll`'s 794 ms, so base64
(`normalize.go:89`, `[A-Za-z0-9+/]{16,}={0,2}`, scanned at `:93`) plus hex
(`normalize.go:115`, `\b[0-9a-fA-F]{16,}\b`, scanned at `:119`) are ~87% of it.
`maxExpanded` (`normalize.go:47`) bounds only the appended decoded text; nothing bounds the
scan over the input. Filed as `GUARD-DF-020`, which also cites `GUARD-DF-018`: the verdict
only reads the first ~19 KB, so ~99.8% of that scan produces text the classifier never sees.

Is this a number a user feels? Yes — an integrator sending document-length messages pays
1.2–1.5 s per call, and a guard that costs that much gets moved off the hot path. It is
filed with a before/after target rather than as an observation.

## Install leg

Covered for this tick, not skipped: the earlier fire's ephemeral bunker run on `bunker-mvp`
(instance `08c686df`) is intact at `/tmp/bunker-qa-evidence-20261114T234313Z-926128.jsonl`
— `launch OK`, `fresh-install OK`, `tests OK`, `lint INFO`, `docker_build N/A`,
`compose_config N/A`. It was **not** re-run this fire because the product tree is
byte-identical to the commit it graded (`4d4a295`); the row below records that reasoning
rather than claiming a silent pass.

## Left behind

- `GUARD-DF-020` — P3, the normalization cost with command, warm/cold numbers, bench output and `normalize.go:89/93/115/119`.
- `GUARD-DF-008-R2` — P3, the unnamed-cap recurrence, citing `GUARD-DF-008` and `DOC-27`.
- this document, and the run record in `.coding-hermes/dogfood-log.md`.

No code, no existing board row, no repo visibility or permission was touched.
