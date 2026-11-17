# GUARD-DF-020 normalization performance evidence

> **Errata (2026-11-17).** This record describes commit bf4aa5d. The bounded
> 128 KiB prefix / 16 KiB suffix scan window documented below under "Detection
> boundary" was removed as unreachable by GUARD-DF-020-R2 (98179da,
> 2026-11-15): normalization now runs over all accepted content, and content
> above the 20,000-byte classification cap is rejected before any normalizer
> runs — so the window and its documented detection limitation no longer
> describe shipped behavior. The regression test cited below as
> `TestBase64DetectionAtSixMiB` was committed in 0782562 and removed together
> with the window machinery in 98179da (GUARD-DF-020-R2);
> `normalize_test.go::TestBase64Expansion` and
> `normalize_bench_test.go::BenchmarkNormalizeAll8MiB` remain. The benchmark
> drives `NormalizeAll` directly, while production rejects content above the
> 20,000-byte cap before normalization, so the window was unreachable in
> production — which is why it was removed. The measurements below are
> preserved as recorded at the commit under test.

Date: 2026-11-15
Commit under test before closeout: bf4aa5d90377a7ee37179de01f87622322a52bc0
Host: AMD Ryzen 7 7840HS, 16 logical CPUs

The baseline was measured on a loaded host, not an idle host. The recorded prior
measurement was 10.57 MB/s at loadavg 29.5 (see
`docs/dogfood/2026-11-14-run8-tick3-cap-and-perf.md`). This run recorded loadavg
6.06, 5.24, 8.06 at 2026-11-15T11:49:01-05:00. No idle-host claim is made.

## Reproducible benchmark

Command:

`go test -run '^$' -bench 'BenchmarkNormalizeAll(24KB|8MiB)$' -benchtime=5x -count=1`

Output after the normalization scan-window change:

```
BenchmarkNormalizeAll24KB-16       5   2445070 ns/op    10.05 MB/s
BenchmarkNormalizeAll8MiB-16       5     15212386 ns/op  551.43 MB/s
```

The 8 MiB result of 551.43 MB/s is approximately 52x the 10.57 MB/s recorded
baseline and is above the 52.9 MB/s target. The 24 KB case remains a full scan
at approximately 10 MB/s and does not receive the 8 MiB speedup.

The benchmark is committed in `normalize_bench_test.go`. `b.SetBytes` makes the
MB/s output reproducible and the benchmark comment gives the exact command.

## Detection boundary

`NormalizeAll` now preserves the original payload for classification but runs
deterministic normalization over a bounded prefix plus suffix window. The window
is 128 KiB from the start and 16 KiB from the end, separated by an explicit
`--- normalization window ---` marker. Encodings in the omitted middle are not
claimed to be detected. A regression test places a base64 payload inside the
suffix window and asserts that it is detected, while a payload in the omitted
middle of an 8 MiB input is not detected. The latter is the accepted, tested
limitation of the bounded window.

This is a deliberate window behavior, not deletion of encoding detection:
base64, hex, ROT13, URL, invisible-character, and homoglyph transforms remain
in the default chain.

## Egress-disabled /check timing

The in-repo HTTP benchmark is `cmd/guardd/normalization_perf_test.go` and uses
`egressDisabledClassifier`, so no payload leaves the process. Command:

`go test -run '^$' -bench '^BenchmarkDisabledEgressCheck8MiB$' -benchtime=3x -count=1 ./cmd/guardd`

After-change output at the recorded load:

```
BenchmarkDisabledEgressCheck8MiB-16  3  57845064 ns/op  145.00 MB/s
```

The prior durable tick-3 measurement for the same egress-disabled 8 MiB /check
path was 1195 / 1192 / 1181 ms at loadavg 25.1 (median 1192 ms), recorded in
`docs/dogfood/2026-11-14-run8-tick3-cap-and-perf.md`. It used a separately built
binary and three HTTP requests, whereas this commit's benchmark reports three
benchmark iterations and includes benchmark overhead; the result is therefore
an observable after measurement, not a false identical-run comparison.

## Trace marker

ch:trace row=GUARD-DF-020 spec=normalize.go wave=.coding-hermes/waves/guard-foreman-2026-11-15-16-29-26.json#GUARD-DF-020 test=normalize_bench_test.go::BenchmarkNormalizeAll8MiB,normalize_test.go::TestBase64Expansion doc=docs/dogfood/2026-11-15-GUARD-DF-020-normalization-perf.md evidence=docs/dogfood/2026-11-15-GUARD-DF-020-normalization-perf.md witness=none:worker-branch verdict=none:pending-tier2 commit=none:worker-commit memory=none:not-written
STATUS: PARTIAL (worker branch; implementation and local evidence complete, publication and tier-2 pending)
TIER: T1 (repo tests and benchmarks invoked locally; loaded-host measurement, no idle-host claim)
EVIDENCE: this committed artifact plus benchmark output above
