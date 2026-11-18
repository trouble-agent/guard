  `classified_bytes: 0` and `truncated: true`; it never silently scores a prefix.
- **Class labels are noisy near the threshold** (a benign base64 blob was once
  labelled `masquerade` at 0.24). Trust the `route`, not the `attack_class`, at
  the margin.
- **This is a filter, not a gatekeeper.** It informs a capability split; it does
  not replace one — pair it with an agent-side layer that gates tools and
  secrets on the returned verdict.

## Build

```bash
go build ./...                                   # library + CLI
go build -ldflags="-s -w" -o bin/guardd ./cmd/guardd   # stripped static binary
```

## Further docs

- [Operator monitoring and recovery runbook](docs/operations.md) — safe guardd startup, liveness and request checks, and bounded recovery steps.
- [Adoption guide](docs/ADOPTION.md) — the exact choke point where each
  consumer (crier, task-router, shell) hooks the guard, and how its
  connector is wired.
- [Threshold calibration](docs/threshold-calibration.md) — measured
  precision/recall of the default policy bands on a labelled corpus; the
  numbers behind the accuracy and near-threshold claims above.
- [Release readiness](docs/release-readiness-2026-10-04.md) — the 2026-10-04
  re-verification of the v0.1.0 release: tag/commit linkage, published
  GitHub Release assets, and checksums.
- [Historical guard documentation audit](docs/audits/guard-docs-2026-10-16-19-57-44.md) — dated findings and source anchors; the board is the status record.
- [Traceability doctrine](docs/traceability-doctrine.md) — the repository-local `ch:trace` fields, evidence tiers, witness rules, and close-out shape.
- [Connectors](connectors/README.md) — the language adapters that turn the
  guard into a one-liner in each consumer stack.
