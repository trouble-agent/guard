# Guard dogfood run 8, tick 5 — policy wiring + operator contract

Tick: `guard-dogfood-2026-11-14-23-17-36` (5th fire of this tick id)
Lane workdir: `/home/kara/.hermes/stand-in/dogfood/guard` (stand-in, no `.git`)
Canonical repo: `/home/kara/trouble-agent/guard`
Measured tree: clean `git archive HEAD` export of `0a5a791` (tree `e130718b`) at
`/home/kara/.hermes/cache/scratch/guard-df5-1794705011/src`, private `HOME` per arm,
`GUARD_EGRESS_ENABLED=false`, throwaway operator token.
Host: 16 cores, loadavg 5.53 / 7.80 / 16.26 at measurement time.

## Verdict

**WORKS, WITH ONE SILENT POLICY NET THAT ISN'T WIRED.** Everything a user is told to
do this run did what the docs said: the runbook's refusal ladder, token precedence,
bind normalization, /check status ladder, policy startup failures, policy precedence
and the egress kill switch all matched `docs/operations.md` cell for cell; the release
artifact downloads, verifies and runs; `make test` and `make docs-check` are green;
per-source `Overrides` genuinely change the served verdict.

What is not true is the safety net around those overrides. `cmd/guardd/main.go:155` is
the only `Guard` construction in non-test code and it never sets `Warnf`, so the
dead-override-key warning that `policy.go:88` describes cannot fire for anyone running
the product — a typo'd override key silently leaves the *looser* base bands in force and
nothing in the logs says so. Filed as `GUARD-DF-022` (P3).

Second, smaller, same shape from the docs side: `docs/operations.md` documents the
`/check` body as `{"content": ...}` only. The handler accepts `id`, `source`,
`channel`, `meta`, and `source` is the field `Overrides` is keyed on — so the runbook
never shows an operator how to make the README Quick Start's per-source fixture fire,
and unknown JSON fields are silently ignored (verified: `{"content":"x","bogus":1}` →
200), so a guessed field name falls back to base bands silently. Filed as `DOC-31` (P4).

## What a user actually did (and what happened)

All cells below were run against the built binary from the clean export.

| Runbook claim (docs/operations.md) | Cell | Observed |
|---|---|---|
| `-serve` with no token must refuse to start | A | `rc=1`, `guardd: no token provided: refusing to serve an unauthenticated /check; set GUARD_TOKEN, or pass -tokenfile <path> (0600)` |
| tokenfile `0640` refused, `0600` accepted | B | B1 `rc=1`, `tokenfile …: mode 0640 is too open (group/world bits set); chmod 600 it first`. The `0600`-accept half was not re-exercised this fire (every serving cell authenticated with `GUARD_TOKEN`); `docs-check` still anchors `main.go:349-351`, the perm gate, and the suite covers the tokenfile reader. |
| precedence `GUARD_TOKEN` > `-tokenfile` > `-token` | C | env token → 200; request carrying the *argv* token → `401 unauthorized: missing or invalid X-Operator-Token` |
| `:port` normalizes to `127.0.0.1:port` | D | `/healthz` 200 `{"status":"ok","egress_enabled":false}`, `/health` 200; server bound loopback only |
| /check status ladder | E | no token 401; bad token 401; GET 405 `POST only`; PUT 405; malformed JSON 400; empty content 400; omitted content 400; valid content 200 |
| policy startup failures | F | missing file `rc=1`; invalid JSON `rc=1`; inverted pair `rc=1` naming `review_at=0.9 > block_at=0.5` |
| `-policy` beats `GUARD_POLICY` | G | 200 with the `-policy` id applied |
| egress kill switch → 200 + in-band errored result | H | `/healthz` `egress_enabled=false`; `/check` 200 `decision=block ... guard_error (fail-closed): hosted classifier egress is disabled`. The CLI-side `rc=2` half of this contract (README:121) was not re-run this fire — it is documented, not observed here. |

Release install leg, followed verbatim from README:

- `curl -L .../releases/latest/download/guard-linux-amd64.tar.gz` → `http=200 bytes=2935335 time=0.76s`
- `checksums.txt` → 182 bytes, names both tarballs; `sha256sum --ignore-missing -c` → `rc=0`, `guard-linux-amd64.tar.gz: OK`
- Independent cross-check of the actual file hash against the listed line → identical (`b75cb31a…`), so the pass is not a `--ignore-missing` false positive.
- `tar xzf` → `rc=0`; ships `guardd` (6.9 MB), LICENSE, README.md dated 2026-10-02 (the v0.1.0-era README, already `DOC-17`).
- The shipped v0.1.0 binary predates `-version`, as expected; `README.md:121` already warns the released binary predates the current contracts.

Override behaviour, served over HTTP (the point of this tick):

```
policy: {"id":"quick-start","block_at":0.85,"review_at":0.50,"fail_closed":true,
         "overrides":{"webhook":{"block_at":0.55,"review_at":0.40}}}

{"content":"please review this patch"}                        -> block_at 0.85 review_at 0.50 override_source null
{"content":"please review this patch","source":"webhook"}     -> block_at 0.55 review_at 0.40 override_source "webhook"
{"content":"please review this patch","source":"github_pr"}   -> block_at 0.85 review_at 0.50 override_source null
{"content":"x","bogus":1}                                     -> 200
```

## The finding, precisely

`policy.go:88` says *"ValidateOverrideKeys and the Warnf watch are what catch keys no
source ever selects"*. In the shipped binary neither exists:

- `grep -rn 'guard.Guard{\|&Guard{' --include=*.go . | grep -v _test.go` → one hit, `cmd/guardd/main.go:155`: `g := &guard.Guard{Classifier: classifier, Policy: pol}` (no `Warnf`).
- `grep -rn ValidateOverrideKeys --include=*.go . | grep -v _test.go` → comments and the definition only, **zero callers**.
- CLI with `overrides:{"webhook":…}` and `-source github_pr`: stderr is exactly one line, `guardd: policy loaded from -policy (id "typo-net", 1 override(s))`. No dead-key warning.
- Served with `GUARD_POLICY`: startup log records the policy load, the token provenance, the egress warning and the listen address — no dead-key warning.
- Positive control: `go test -run 'WarnUnknownOverrideKeys|NilWarnf|Override' -v .` passes in the export, so the warning logic is correct; only the wiring is missing.

Impact: an operator who mistypes an override key (`"webook"`, `"Webhook"`, or a source
their app never sends) keeps the base bands for traffic they meant to tighten, and the
runbook has no log line to read because none is emitted. Not a breach — fail-closed
defaults still apply — hence P3, not P2.

## Performance (Step 2b)

Headline operations, measured on the clean export:

| operation | measurement |
|---|---|
| cold CLI start, documented stdin path, private HOME | 10–12 ms (5 runs: 11.2, 10.2, 11.7, 10.8, 12.1 ms) |
| `make build` (rm bin, module cache warm) | 298 ms, rc=0 |
| `make test` (full suite) | 4.47 s, rc=0 |
| `make docs-check` (replays README build + commands + main.go line anchors) | 0.55 s, `PASS` |
| server ready after launch (port answering `/healthz`) | 12–13 ms |
| `/healthz`, warm | median 0.274 ms (n=5) |
| `/check`, 4 KiB body, warm | 0.71–1.23 ms |
| `/check`, 64 KiB body, warm | 6.4–8.0 ms |
| `/check`, 1 MiB body, warm | 96–99 ms (≈10.5 MB/s) |

Nothing here is a new user-felt cost: single-digit milliseconds for realistic payloads,
and the 1 MiB figure is the same ~10.5 MB/s normalization throughput already carried by
`GUARD-DF-020` (P3, 8 MiB at the request cap ≈ 0.79 s CPU). **No `PERF-<n>` row filed this
tick** — a win nobody can feel devalues the real ones, and the one path that is slow
enough to notice is already filed. Boundary on this claim: with egress disabled the
classifier call is not exercised (no credential, and this lane must never mint one), and
loadavg was moderate (5.5–16) rather than idle, so the numbers above are local-path costs
only — they are not a classifier-latency claim.

## Install leg (ephemeral bunker)

`bunker-qa.sh launch /home/kara/trouble-agent/guard --server bunker-mvp --ttl 4h` →
`agent=791648f6` on `bunker-mvp`, battery detached, `evidence=/tmp/bunker-qa-evidence-20261115T012021Z-2310724.jsonl`.
The render bug that made `GUARD-DF-009` a SKIPPED leg (unescaped `UI_PORTS` under
`set -u`) is fixed in the current script: `bunker-qa.sh` now ships a 138 KB script and the
launch reports `battery started detached`. This leg **ran and passed** — it is not a skip:

| cell | result |
|---|---|
| `toolchain-bootstrap` | OK — go 1.27.4 discovered against a 1.26.11 floor (ambient 1.22.2), gcc/g++, make, jq, docker + buildx + compose plugin, node 22.15.0 |
| `fresh-install` | OK — public-origin clone into `~/qa-guard`, `go build ./...` compiles on a clean agent |
| `tests` | OK — `rc=0` |
| `lint` | INFO — `golangci-lint` absent on the agent, so lint did not run (reported, not passed) |
| `ci-pass` | OK — `act` ran the one triggerable workflow, 1 job green, `rc=0` |
| `upgrade` | OK — previous release v0.1.0 tree → HEAD reinstall clean |
| `chaos-resource` | PASS — suite survives a 3 GiB memory cap (`cap=3145728KB`, child + subshell saw the cap) |
| `chaos-errorpath` | OK — clean error on missing config, `rc=2` (this is also the observed `rc=2` fail-closed evidence the local run did not reproduce) |
| `chaos-disconnect` | INFO — passes with the network cut (no network dependency) |
| `docker-deploy`, `ui-probe`, `chaos-shutdown`, `chaos-corruption` | N/A — no compose file, no web UI, no db/state files |

Teardown verified: the agent home is gone from `bunker-mvp` after the battery completed, so no
ephemeral instance was left running.

## What I did not do

- No code, policy or workflow change; no board row closed; no id reused (both ids minted
  through `board_next_id.py` → `GUARD-DF-022`, `DOC-31`, both free at file time).
- No repo visibility or permission change; no credential minted to make any leg work.
- Did not re-file the released-tarball README staleness (`DOC-17`, complete) or the
  Python connector `--source` gap (`GUARD-DF-019`, open) — both are live rows already.
