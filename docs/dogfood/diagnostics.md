# Guard — diagnostic trail (dogfood)

This file records durable real-use diagnostics and the current integration boundaries. Historical detailed runs remain in the dated `docs/dogfood/*findings.md` files and `.coding-hermes/dogfood-log.md`.

## Current run: 2026-11-27, HEAD 337890bef4a9a818e5de9efa8b98741f7deab6d1

This run exercised the source-built Go CLI and authenticated HTTP service as a real consumer. `make build` produced `guardd v0.2.0`; a benign hosted classification delivered, injection text quarantined, an isolated no-key CLI returned the documented fail-closed rc=2 error, and HTTP health/auth/egress-disabled paths matched their contracts. Local `go test ./... -count=1`, `go vet ./...`, and `make docs-check` all passed.

Five hyperfine runs after one warmup measured benign CLI mean 774.1 ms (597–1249 ms) and injection mean 1.181 s (1.024–1.319 s) on 16 CPUs at load about 20.36–21.65. User+system CPU was only tens of milliseconds; the hosted Jev round trip dominates. No PERF row was filed because no local code path was slow enough to justify profiling.

The ephemeral `bunker-qa.sh` battery ran against this exact HEAD on agent 9009d5cc at bunker-las-02 and was collected to `/home/kara/.hermes/qa/evidence/guard/qa-20261127T125917Z/`. Fresh install, tests, CI/act, upgrade, resource, disconnect, corruption, and error-path cells completed according to their evidence. The native lint cell failed before linting because another golangci-lint process was running; this is not a product lint result. The agent and key were absent after collection.

## Durable integration rules

- Bind consumers to `route` plus `constraints`, not `decision` alone.
- Check `errored` before trusting classification-only fields such as `attack_class` or `score`.
- Use `GUARD_TOKEN` or a 0600 token file for HTTP; `/check` requires `X-Operator-Token`.
- `normalizations` describes transforms that ran; `load_bearing_normalizations` describes transforms that changed the verdict.
- `duration_ms` and `cost_usd` include all classifier calls caused by the verdict; caught messages can cost about two calls.
- The Python connector's failure exit code is 3; guardd's CLI fail-closed error exit code is 2.
- The HTTP body limit is 8 MiB. The CLI stdin documentation/source anchor is under reconciliation in GUARD-DF-017; do not infer an oversized-input result from this run.

Evidence for the current run is in `/home/kara/.hermes/cache/scratch/guard-dogfood-20261111-223617/` while the dated integration report is `docs/dogfood/2026-11-11-run7-integration.md`.
