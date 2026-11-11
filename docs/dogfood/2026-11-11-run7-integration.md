# Guard dogfood run 7 — integration report

Tick: guard-dogfood-2026-11-11-22-36-17
Target HEAD: 92c3e07178f2dd267a1512318029c06e50922c5e
Canonical checkout: /home/kara/trouble-agent/guard

Verdict: SHIPPABLE core, with two P4 documentation findings.

Promise: provide one message-security filter through the Go CLI, authenticated HTTP service, and Python connector, with fail-closed behavior and one Result contract.

Real-use results:

- Go build, vet, and `go test ./... -count=1` passed. The suite covered four packages and completed in 6.30s real; `make docs-check` passed in 0.70s real.
- `./bin/guardd --version` returned `guardd v0.2.0` with rc=0.
- Live-key benign CLI classification returned rc=0, route=deliver, score=0.19, duration_ms=843, wall 0.84s.
- Live-key injection classification returned rc=0, route=quarantine, score=0.98, duration_ms=811, wall 0.81s.
- Isolated-HOME/no-key CLI classification returned the documented fail-closed block with rc=2.
- Local HTTP server with egress disabled returned health 200, unauthenticated /check 401, empty content 400, and authenticated fail-closed error verdicts with HTTP 200.
- A fresh Python virtual environment installed `connectors/python` successfully and exposed the `guard` console command. An unreachable endpoint returned the documented connector failure with rc=3.

Performance:

- Headline operation: `./bin/guardd -content "The quarterly report is ready for review."`.
- Five warm runs after two warmups, all rc=0: mean 406.0ms ±17.3ms; range 378.8–421.8ms. Exact JSON: `/home/kara/.hermes/cache/scratch/guard-dogfood-20261111-223617/live-benign-hyperfine.json`.
- Direct first-process samples were 0.84s for deliver and 0.81s for injection. These are process-cold observations, not a fully isolated provider/cache-cold arm; full cold timing remains unverified because the live credential was intentionally not copied or minted.
- User+system CPU was about 21ms on the direct deliver run versus 840ms wall, so the wait is provider/network dominated. No local profile was justified and no PERF row was filed.
- GUARD-DF-016 records stale precision in the usage skill's dated performance band.

Install leg:

`bunker-qa.sh launch` and `collect` ran on fresh agent 1867ca04 at bunker-mvp. The terminal evidence file contains 23 collected rows: fresh-install OK, lint OK, tests OK, CI act 1 job green, upgrade v0.1.0→HEAD OK, chaos-disconnect INFO, chaos-resource PASS, chaos-errorpath OK; Docker, compose, UI, shutdown, and corruption cells were N/A by repository shape. Collect destroyed the run-owned agent. Post-collect `bunker info 1867ca04` returned not_found and `bunker list --server bunker-mvp` contained no 1867ca04.

The selected host had shared network namespaces and unrelated running agents, so this is not a tenant-isolation proof. That boundary is recorded, not treated as a product failure. Install evidence: `/home/kara/.hermes/cache/scratch/guard-dogfood-20261111-223617/bunker-evidence.jsonl`.

Findings:

- GUARD-DF-016: current live CLI timings no longer reproduce the 0.24–0.40s usage-skill band precisely; the Jev round trip remains the likely dominant cost, so this is documentation drift rather than a code slowdown claim.
- GUARD-DF-017: the usage skill's CLI stdin-cap source anchor is stale against current `cmd/guardd/main.go`; this tick did not run a multi-gigabyte probe and makes no unmeasured memory claim.

Board proof: each finding id occurs exactly once, each has a paired task_created event, and `boardctl -C /home/kara/trouble-agent/guard validate` returned rc=0 with 73 pre-existing warnings unchanged.

No product code was modified. Existing unrelated untracked files were left untouched: `.coding-hermes/audits/guard-readme-2026-10-30-21-53-18.md` and `.coding-hermes/board/tasks.jsonl.pre-resolve`.
