# Guard — diagnostic trail (dogfood)

This file records durable real-use diagnostics and the current integration boundaries. Historical detailed runs remain in the dated `docs/dogfood/*findings.md` files and `.coding-hermes/dogfood-log.md`.

## Current run: 2026-11-11, HEAD 92c3e07178f2dd267a1512318029c06e50922c5e

The Go CLI, authenticated HTTP service, and Python connector all exercised successfully. The no-key CLI path returned the expected fail-closed block with rc=2. The local HTTP path returned health 200, unauthenticated /check 401, empty content 400, and an authenticated egress-disabled error verdict with HTTP 200. A fresh Python virtual environment installed `connectors/python` and exposed the `guard` console command; an unreachable endpoint returned rc=3.

The live provider path returned a benign deliver verdict in 0.84s and an injection quarantine verdict in 0.81s. Five warm deliver runs measured 406.0ms ±17.3ms, range 378.8–421.8ms. User+system CPU was about 21ms on the direct deliver run, so the Jev network round trip dominates. The documented usage range is tracked by GUARD-DF-016 because its precision no longer reproduces at this HEAD.

The install battery completed on fresh bunker agent 1867ca04 at bunker-mvp: fresh-install, lint, tests, CI, upgrade, resource, and error-path cells were green/OK; cells for Docker, compose, UI, shutdown, and corruption were N/A by repository shape; network disconnect was INFO because the suite has no network dependency. Collect destroyed the agent and post-collect info returned not_found. The host used shared network namespaces and had unrelated agents, so tenant-isolation coverage is not claimed.

## Durable integration rules

- Bind consumers to `route` plus `constraints`, not `decision` alone.
- Check `errored` before trusting classification-only fields such as `attack_class` or `score`.
- Use `GUARD_TOKEN` or a 0600 token file for HTTP; `/check` requires `X-Operator-Token`.
- `normalizations` describes transforms that ran; `load_bearing_normalizations` describes transforms that changed the verdict.
- `duration_ms` and `cost_usd` include all classifier calls caused by the verdict; caught messages can cost about two calls.
- The Python connector's failure exit code is 3; guardd's CLI fail-closed error exit code is 2.
- The HTTP body limit is 8 MiB. The CLI stdin documentation/source anchor is under reconciliation in GUARD-DF-017; do not infer an oversized-input result from this run.

Evidence for the current run is in `/home/kara/.hermes/cache/scratch/guard-dogfood-20261111-223617/` while the dated integration report is `docs/dogfood/2026-11-11-run7-integration.md`.
