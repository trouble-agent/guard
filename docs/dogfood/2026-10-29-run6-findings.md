# Guard dogfood run 6 — 2026-10-29

Trace: guard-dogfood-2026-10-29-10-20-47
Target: trouble-agent/guard
Repository HEAD: 6cfd37fe5b2b7fb7f172d5e8a6f1553bbc1c5112
Build tree: detached worktree `/tmp/guard-dogfood-20261029-102047-Bz0g/src` at the same HEAD; the canonical checkout was clean and unchanged.
The repo's knowledge artifacts below were written in the canonical checkout; board files were not modified.

## Promise

Guard claims to provide one message-security filter through a Go CLI, an authenticated HTTP service, and a Python connector, with a shared Result contract and fail-closed behavior when classification cannot run. A user should be able to build it, classify a message, operate the HTTP service safely, install the Python connector, and understand the result route and capability constraints.

## Real-use evidence

The detached HEAD build completed in 0.61s (`go build -o /tmp/guard-dogfood-20261029-102047-Bz0g/guardd ./cmd/guardd`, rc 0). `-h` printed the CLI entry points. `go test ./... -count=1` passed all four packages; `go vet ./...` passed; `make docs-check` passed, including the README build, fail-closed JSON, empty-stdin rc=2, and documentation anchors.

A live HEAD server ran at `127.0.0.1:18947` with `GUARD_EGRESS_ENABLED=false` and a scratch token. The listener was independently verified with `ss`, and the log reported the bind only after it succeeded. `/healthz` and the new `/health` alias both returned HTTP 200 with `{"status":"ok","egress_enabled":false}` in 37ms. `/check` returned 401 without a token, 401 with a wrong token, 400 for an empty content object, 400 for malformed JSON, 405 for GET, and HTTP 200 with the documented fail-closed quarantine Result when egress was disabled. The disabled-egress response had `errored:true`, `route:"quarantine"`, and all three capability constraints false. These are the documented contracts, not new findings.

The Go CLI was also exercised with the live key already available in the operator environment: `guardd -content 'hello, world'` returned rc=0 and a deliver Result. The first cold process took 0.86s; later cold processes took 0.30s and 0.29s. Five additional user-visible CLI runs took 319ms, 328ms, 327ms, 478ms, and 493ms wall time, with the service-reported classifier durations 281–457ms. The result reported `provider:"jev"`, `route:"deliver"`, and a small classifier cost. This is dominated by the external Jev round trip; it is not slow enough to justify a PERF row in this run, so no profile was run.

The Python connector was installed from `connectors/python` into a fresh venv with `pip install --no-deps ...` (rc 0). The venv contained the `guard` console script. The installed console command reached the live service and returned rc=2 with the expected disabled-egress quarantine JSON. The initial `shutil.which` check returned no path only because the probe process did not put the venv's `bin` directory on PATH; the corrected check found `/tmp/guard-dogfood-20261029-102047-Bz0g/pyconsumer/venv/bin/guard`. This was a probe correction, not a product defect.

## Install leg

The required fresh-machine leg ran through `bunker-qa.sh` on run-owned agent `1d6f29c2` at `bunker-mvp`, tested HEAD `6cfd37fe5b2b7fb7f172d5e8a6f1553bbc1c5112`, and was collected/destroyed. The launch was recorded at `/tmp/guard-dogfood-20261029-102047-Bz0g/bunker-launch.out`; the final evidence is `/tmp/guard-dogfood-20261029-102047-Bz0g/bunker-evidence.jsonl` (23 lines including launch and collect). Agent-side cells independently showed: `fresh-install OK` with artifact `/home/bunker-1d6f29c2/.local/bin/guardd` (11,086,664 bytes), `ci-pass OK` (one act job green), `upgrade OK`, `chaos-disconnect OK` with `DISCONNECT_SUITE_SUMMARY rc=0 outcomes=13`, `chaos-resource PASS`, and `chaos-errorpath OK`. Docker/UI/shutdown/corruption cells were N/A or INFO by project shape. Teardown was verified after collection: `bunker info 1d6f29c2` returned `not_found` and a fresh `bunker list --server bunker-mvp --status all` contained zero matches. This is install PASS evidence; the host's pre-launch load (53.4% CPU, 49 containers, 2 existing agents, 70% disk, 20 orphan keys) was recorded as capacity context, not misattributed to the product.

## Value judgment

Verdict: SHIPPABLE. The promised source-build, CLI, authenticated HTTP, Python-connector, and fresh-agent install workflows completed at current HEAD. Error behavior was explicit, `/check` authenticated correctly, the Python package exposed its console entry point, and the fresh-agent battery independently exercised install, upgrade, CI, disconnect, resource, and error paths.

No new product finding or PERF row was filed by this read-only lane. Existing pending release-readiness rows remain outside this dogfood pass.

Evidence directory: `/tmp/guard-dogfood-20261029-102047-Bz0g/` (live probe, timing captures, build/test output, and Python consumer).
