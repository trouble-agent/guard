# Install battery — collected to terminal (dogfood delivery 8)

Tick `guard-dogfood-2026-11-27-12-27-32`, delivery 8. Tested source:
`64167c23fb4a45ef8e6eda7adca229ba804174d3`, which equals `origin/main` and `HEAD` at collection.

## Provenance

| Field | Value |
|---|---|
| Agent | `4aae569b` on `bunker-mvp`, TTL 4h, launched `2026-11-27T14:45:31Z` |
| Launching delivery | a prior delivery of this same tick (third delivery; see `docs/dogfood/2026-11-27-122732-third-delivery.md`) |
| Collect command | `bash ~/.hermes/scripts/bunker-qa.sh collect --evidence /home/kara/.hermes/qa/evidence/guard/guard-dogfood-2026-11-27-12-27-32-third/evidence.jsonl` |
| Collect result | `COLLECTED agent=4aae569b rows=23`, ledger `27` rows, sha256 `ed2aca2f…def979b5` |
| Source path (scratch) | `/home/kara/.hermes/qa/evidence/guard/guard-dogfood-2026-11-27-12-27-32-third/` |
| Copies here | `qa-evidence.jsonl` (= `evidence.jsonl`), `executable-probe.log`, `HANDOFF.md`, `logs/` (= `qa-20261127T144926Z/logs/`) |

Transformations: **none**. Byte-identical copies, including absolute host paths. This repo has no
publish/elision gate (no `.git/hooks/pre-commit`, no publish-hook script) and the previously
committed battery evidence already carries the same path shapes, so no elision was applied.

## Cell census (27 rows)

19 `OK`, 1 `PASS`, 2 `SKIPPED`, 3 `N/A`, 1 `INFO`, 1 `WARN`. Nothing `FAIL`, nothing `UNVERIFIED`.

- `target-binding` OK — repo identity from `origin` remote.
- `launch` OK — agent `4aae569b` (bunker-mvp), ttl 4h, cpu 4.0.
- `toolchain-bootstrap` OK ×10 — Go 1.27.4, gcc/g++, GNU Make 4.3, jq 1.8.2, compose plugin, buildx v0.39.1, node v22.15.0 + npm 10.9.2, `HARNESS-COMPOSE=ok`, `HARNESS_NODE=ok`.
- `fresh-install` OK — `go build ./...` succeeded. **Compile-only by design**: the cell reports
  `no artifact expected … a compile/dependency-install leg ships code, not binaries`, so it does
  not by itself prove an executable was installed. See "executable" below.
- `lint` OK — `golangci-lint run (rc=0)`. This is the cell that read `FAIL` in two earlier
  deliveries of this tick; on this agent the toolchain matched and it passes.
- `tests` OK — `go test ./... -count=1` rc=0.
- `ci-pass` OK — `act`: 1 job green, rc=0.
- `upgrade` OK — previous release v0.1.0 (tree via `git archive`) → HEAD reinstall clean.
- `chaos-corruption` OK, `chaos-resource` PASS (suite survives a 3 GiB cap), `chaos-errorpath` OK
  (clean rc=2 on missing config), `chaos-disconnect` INFO (passes with the network cut).
- `docker-deploy`, `chaos-shutdown` SKIPPED — battery safety gate (controller is the live Hermes
  host). `docker_build`, `compose_config`, `ui-probe` N/A — no trigger in this repo.
- `collect` OK — `pulled=23`.
- `evidence-outside-repo` WARN — the raw ledger lived outside the checkout; **this committed copy
  is the repair**, so a closeout no longer cites a deletable scratch path.

## Executable on the fresh host (separate manual probe, not a cell)

`executable-probe.log`: on the fresh host a separate `go build -o "$HOME/guardd-dogfood" ./cmd/guardd`
built an executable and `guardd-dogfood` ran with an isolated HOME and no API key:

```
{"decision":"block","route":"quarantine","reason":"guard_error (fail-closed): no openrouter api key available", …,"errored":true}
classification_exit=2
```

`guardd dev` — the version string is `dev` because the binary was built from the tree, not
stamped by the release build. So: an executable **built from this source runs and fails closed on
a fresh machine**. What remains unproven is the *documented install path for a stranger* (the
published release tarball), which is stale behind owner-gated `REL-GUARD-007`, not this battery's
subject.

## Teardown

`bunker info 4aae569b --server bunker-mvp` → `not_found`; `bunker list --server bunker-mvp
--status all` no longer lists it. `bunker-las-02` (agents `9009d5cc`, `699c50b0` from earlier
deliveries) reports `No agents found.` One unnamed agent on `bunker-las-03` (`18beb8fa`, created
`2026-11-27T14:46:06Z`, expires `18:46:06Z`) is not named by any record of this tick and was left
running.
