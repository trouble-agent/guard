# Dogfood run 8, tick `guard-dogfood-2026-11-14-23-17-36`, fourth fire — benchmark re-derivation

Lane `/home/kara/.hermes/stand-in/dogfood/guard`, board written in `/home/kara/trouble-agent/guard`.
HEAD `0a5a791ecb8cdc777a015fc746504a42a1bc94da`, tree `e130718b8b1a80b5e6c854629f629a828b724688`.
Product tree is byte-identical to run 8's tested commit `4d4a295`
(`git diff --stat 4d4a295..HEAD -- . ':!.coding-hermes' ':!docs'` prints nothing), and
`normalize.go` at HEAD hashes `02f9a4f3f6d91f07335f3c4278951d6189b66ca07fa723f7a175d04bca45aaaf`
— the same bytes the benchmark below was run against.

## Why this fire exists

The first three fires of this tick walked `arm -> launch -> fresh-install -> tests -> use` and
then exercised the 8 MiB body cap. Their closeout is intact. This fire does not re-run product
use for new findings; it re-derives the numeric premises the tick put on the board, because the
prior fire kept the Go benchmark **only in prose** — no captured benchmark output was written
next to the other artifacts (`cap/cap-probe.json`, `ext/*.log` exist; nothing containing `ns/op`
did). A baseline that a worker is judged against should be reproducible from an artifact, not
from a paragraph.

## Re-derived measurement

Scratch `git archive` export of HEAD (never the repo tree), AMD Ryzen 7 7840HS, 16 logical CPUs,
loadavg `24.77 29.35 28.24` (the prior fire ran at `29.5` — comparable, both far from idle):

```
go test -run XXX -bench 'BenchmarkNormalizeAll|BenchmarkExpandBase64|BenchmarkExpandHex|BenchmarkExpandURL' \
        -benchtime 5x -count 1
BenchmarkNormalizeAll_8MiB-16       5   761580355 ns/op   11.01 MB/s
BenchmarkNormalizeAll_24KB-16       5     2312204 ns/op   10.64 MB/s
BenchmarkExpandBase64_8MiB-16       5   356455789 ns/op   23.53 MB/s
BenchmarkExpandHex_8MiB-16          5   257643866 ns/op   32.56 MB/s
BenchmarkExpandURL_8MiB-16          5      130973 ns/op  64048.57 MB/s   (short-circuits)
PASS   ok  github.com/trouble-agent/guard  8.328s
```

Artifact: `~/.hermes/cache/scratch/guard-dogfood-20261114-231736/ext/head-c62f815/bench-tick4.log`
(produced this fire; the harness `zz_bench_dogfood_test.go` sits beside it).

| measurement | tick-3 record | this fire | delta |
|---|---|---|---|
| `NormalizeAll` 8 MiB | 793.8 ms / 10.57 MB/s | 761.6 ms / 11.01 MB/s | −4.1 % time |
| `NormalizeAll` 24 KB | 2.46 ms / 10.02 MB/s | 2.31 ms / 10.64 MB/s | −6.1 % time |
| `ExpandBase64` 8 MiB | 371.1 ms / 22.61 MB/s | 356.5 ms / 23.53 MB/s | −4.0 % time |
| `ExpandHex` 8 MiB | 317.0 ms / 26.46 MB/s | 257.6 ms / 32.56 MB/s | −18.7 % time |
| base64+hex share of `NormalizeAll` | 87 % | 80.6 % | scalar noise |

Conclusions that survive re-derivation:

- `GUARD-DF-020`'s premise holds. Throughput is flat across a 340× size change
  (11.01 MB/s at 8 MiB vs 10.64 MB/s at 24 KB) — per-byte linear work, not a threshold artifact.
- base64 (`normalize.go:89`, scanned at `:93`) and hex (`normalize.go:115`, scanned at `:119`)
  are still the dominant passes at ~81 % of `NormalizeAll` combined, at the same two `file:line`
  anchors the row cites. The 87 %/81 % spread is measurement noise, not a correction.
- The user-visible consequence is unchanged: an integrator sending an 8 MiB document pays
  ~0.76–1.5 s of local CPU per call before the classifier is reached.

## Defect found in the board row (filed)

`GUARD-DF-020`'s acceptance criterion says the 10.57 MB/s baseline was "measured at loadavg <5".
It was not: that row's own `reasoning` records loadavg `29.5`, and this fire re-measured at
`24.77`. The number is reproducible on a loaded host (11.01 MB/s, +4 %), but a worker told to
reproduce it at loadavg <5 will measure a different baseline and cannot tell whether a 5× claim
passes. Filed as `GUARD-DF-021` (P4) to re-anchor the criterion to an explicit MB/s target with
both a loaded and an idle before/after table. `GUARD-DF-020` itself was not edited — this lane
never rewrites board rows.

Note for readers comparing digests: tick-3's `detail.tree_fp` (`d3d9ef5a…`) does not match any
digest derivable from this tree (`sha256` over `git ls-tree -r HEAD` = `1a68c9a6…`); its method
was never stated. It is not used as evidence in this document.

Digest bookkeeping for `GUARD-DF-021` itself: `boardctl create` stores the `--evidence-fingerprint`
argument in its evidence block's `tree_fp` slot, so that slot carries this finding's signature
digest, not a tree digest —
`sha256("GUARD-DF-021|GUARD-DF-020-ac-baseline-loadavg|1a68c9a6337ac43cb56970f190a2d06b933cd2577e863df00c9fb5fd5ceb8175")`
= `10e6b012fe65e72f43a105d2e5ae4e0a6a990ab63062344cacfbd95e48db4bf4`. The tree digest that input
embeds is `1a68c9a6…`. `detail.fingerprint` (`9c83a19d…`) is boardctl's own row hash.

## What this fire did not re-run, and why

- HTTP `/check` cap boundary: not re-run. The tick-3 artifact `cap/cap-probe.json` is intact and
  its four cases match the published table byte-for-byte — 7 MiB → 200 in 1394.8 ms;
  8388608 B → 200 in 1208.1 ms; 8388609 B → 400 `bad request: unexpected EOF` in 28.8 ms;
  9437184 B → 400 in 26.1 ms. Re-deriving a stored artifact does not require a second server.
- Install leg: not re-run. The ephemeral bunker install on `bunker-mvp` (instance `08c686df`)
  graded a byte-identical product tree; the instance is gone (teardown held) and a fresh launch
  would grade the same bytes. This is recorded as inherited, not as a silent pass.

## Teardown

No `guardd` process, no listener on `18790`/`18791`/`8768`, no bunker instance `08c686df`.
The only Go workload this fire ran was the 8.3 s benchmark in the scratch export.
