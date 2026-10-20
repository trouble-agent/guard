# Guard dogfood run 4 — 2026-10-19 (tick guard-dogfood-2026-10-20-01-29-06)

**Verdict: ✅ SHIPPABLE (core unchanged) — one P2 fresh-machine tooling gap, one P2
docs/contract gap, one P3 accounting gap.**

**Promise (Step 1):** "a user can classify untrusted text for prompt injection —
CLI, `guardd` HTTP service, Go library, or Python connector — and get one stable
`Result` contract (`route` + `constraints`) everywhere."  **Reality: it holds.** Every
documented status code, body, exit code, resolution order and fail-mode stance was
driven live and matched the docs exactly (matrix below). The gaps found are in
tooling/documentation/accounting, not in the classification path.

**Angle (delta-driven).** Last dogfood entry: run 3b, 2026-10-16, HEAD 0b9db86. Delta
to this run's HEAD `bcb6002`: docs DOC-16/17/18/19 + README-8, and one CODE fix —
`d71b80c` (REVIEW-GUARD-003: `Policy.apply` now selects per-source `Overrides` on
`Input.Source`, previously on `res.Policy`). That fix's own surface was never driven
by any dogfood run (run 2 recorded "webhook source override accepted", which at
b23aad1 was accepted but inert). So this run took the surface no prior run touched:
**the policy layer as an integrator configures it** (per-source Overrides +
`QuotedDowngrade`), plus the connector contract DOC-18/17-R3 wrote up on 2026-10-18,
plus a fresh-machine install leg.

Environment: dev box `go1.26.9`, HEAD `bcb6002` = `origin/main` (public
`https://github.com/trouble-agent/guard.git`); fresh box = ephemeral bunker agent
`ae4ac4b7` on `bunker-mvp` (Ubuntu, image go1.26.5, no `jq`), DESTROYED after the run.
Scratch consumer lives OUTSIDE the repo at `~/guard-integrator-1019/`
(`go.mod` with a `replace` to the checkout, `main.go`, `http-probe.sh`,
`conc-probe.sh`, `py-probe.sh`, `reproof-perf.sh`, `agent-*.sh`).

## 1. Real use: the Go library consumer (the fix's surface)

`go run . offline` — a real consumer's ingress policy, with the documented
`Classifier` expansion point supplying a FIXED score so the policy, not the model, is
what is observed:

```
== per-source Override at score 0.60 (default block_at=0.85 review_at=0.50) ==
source=webhook   route=quarantine decision=block risk=high   score=0.60 reason="injection signals above block threshold"
source=inbox     route=review     decision=allow risk=medium score=0.60 reason="injection signals in the review band"
source=<empty>   route=review     decision=allow risk=medium score=0.60 reason="injection signals in the review band"
== review-band override at score 0.40 ==
source=webhook   route=review     decision=allow ... (override ReviewAt 0.30 applied)
source=inbox     route=deliver    decision=allow risk=low    score=0.40
```

**REVIEW-GUARD-003 verified at HEAD from the consumer side:** a `webhook` override
tightens (0.60 → quarantine/block), sibling sources keep the base bands, an empty
source selects no override. Capabilities follow the route (quarantine/review strip
tools+network, deliver grants them).

`go run . live` — `QuotedDowngrade` (connectors/README.md:38-43) driven with three
independent quotation phrasings, live Jev:

```
direct-injection   route=quarantine score=0.99 class=instruction_injection
benign             route=deliver    score=0.08
quoted-warning     route=review     score=0.97 class=instruction_injection   (a "DO NOT FOLLOW THIS" wrapper)
quoted-codefence   route=review     score=0.97 class=instruction_injection   (a fenced test case)
quoted-testcase    route=review     score=0.95 class=masquerade              (a canary string)
```

**The documented promise holds exactly:** above-block scores on quoted/example text
downgrade quarantine → review, never to deliver, and never grant capabilities. The
promise is not just documented — it is reachable in real use with ordinary phrasing.

`go run . http` — the Go HTTP connector: real verdict against a live `guardd`;
against a dead endpoint `FailClosed` returns `guard unreachable: … connection
refused`, `FailOpen` returns `route=review decision=allow errored=true` with **all
capabilities withheld** — exactly the Go-client stance in the connectors' fail-mode
table.

## 2. Real use: HTTP surface (documented contract, all four cases + limits)

`bash http-probe.sh` on `guardd -serve` (HEAD build):

| probe | result | documented? |
|---|---|---|
| `GET /healthz` (no auth) | `200 {"status":"ok","egress_enabled":true}` | exact ✓ |
| `POST /check` no token | `401 unauthorized: missing or invalid X-Operator-Token` | exact ✓ |
| `POST /check` wrong token | same `401` | exact ✓ |
| `GET /check` with token | `405 POST only` | exact ✓ |
| malformed JSON | `400 bad request: invalid character 'o' in literal null (…)` | shape ✓ (decoder text leaked — DF-G2 class, now documented as `<parse error>`) |
| `{}` | `400 content is required` | exact ✓ |
| `{"input":"hello"}` | `400 content is required` | **DF-G1 still live** (see findings) |
| valid `{"source":"inbox",…}` | `200` + full `Result` (deliver, score 0.09, `duration_ms` 435) | ✓ |
| 9 000 014-byte body | `400 bad request: unexpected EOF` | shape ✓, limit unnamed (DF-G4) |

Concurrency (5 simultaneous injection payloads): all `200`, all `quarantine`,
per-request wall 0.55–0.59 s vs single-request 0.54 s — no queue degradation and no
cross-request bleed at this depth; `guardd` writes no per-request log line, so
attribution has to come from the caller's own correlation (acceptable for a shared
instance, noted only).

## 3. Real use: Python connector (DOC-18 contract, live)

`bash py-probe.sh` — `pip install ./connectors/python` into a venv (clean, no PEP 668
friction when a venv is used), then the documented exit-code map against a live
server:

| leg | rc | matches docs |
|---|---|---|
| deliver | `0` | ✓ |
| injection (quarantine) | `2` | ✓ |
| guardd absent | `3` (+ stderr reason, no stdout) | ✓ |
| wrong token / no token (401) | `3` | ✓ |
| `--fail-open`, guardd absent | `1`, `decision="error"`, `errored=true`, `constraints` PRESERVED (`allow_tools`, `allow_secrets` true) | ✓ (table row "Python") |
| unknown flag | `2` (argparse usage on stderr, **no** verdict JSON on stdout) | ✓ |
| stdin mode (no content arg) | works, verdict on stdout | ✓ |
| `GUARD_ENDPOINT` env vs `--endpoint` | `--endpoint` wins; env honoured when flag absent | ✓ |
| module default snapshots env at IMPORT | late env change ignored, `Guard()` used the import-time endpoint and failed as documented | ✓ (documented trap at connectors/README.md:86-96 — reproduced exactly) |
| `Guard(fail_mode='open')` against a REACHABLE server | real verdict returned (no synthetic outage verdict) | ✓ |

## 4. Install leg (fresh machine) — driver broken, leg run by hand

`bunker-qa.sh launch` **failed before spawning a battery**:

```
→ agent=ae4ac4b7 @ bunker-mvp (tailscale)
/home/kara/.hermes/scripts/bunker-qa.sh: line 1061: UI_PORTS: unbound variable   (x2)
ERROR: battery shipped EMPTY (~/qa-run.sh 0 bytes) on agent=ae4ac4b7 — script generation failed; NOT launching
```

`bunker-qa.sh` runs `set -uo pipefail` and renders the remote battery through
`( cd / && cat <<EOF … )`; an unescaped `$UI_PORTS`/`${BUNKER_QA_UI_PORTS:-}` fragment
in the render body expands on the controller (`UI_PORTS` is set later, inside the
remote script text) → `nounset` aborts the render → 0-byte battery. The driver
correctly refused to claim a launch (good), but **no battery can ship from this
script revision**, fleet-wide (script mtime 2026-10-19 14:42, i.e. the same day).
The generated script's own fence comment (bunker-qa.sh:1050-1058) documents exactly
this hazard class. Filed as the SKIPPED-install-bunker row.

Substitute leg (the skill's sanctioned fresh-agent procedure) on the SAME fresh agent
that was already spawned, clone of the documented PUBLIC origin (no credentials
minted, no visibility touched):

```
fresh box: git/go/make/python3/pip3 present, jq MISSING, go1.26.5 (image) < go 1.26.9 floor
clone https://github.com/trouble-agent/guard.git  -> HEAD bcb6002 == origin/main    (~0s)
go build ./cmd/guardd                             -> rc=0, 47s (toolchain auto-downloads go1.26.9)
printf 'hello, world' | ./guardd                  -> rc=2, byte-exact documented fail-closed JSON
GUARD_TOKEN=<throwaway> ./guardd -serve :10200    -> /healthz 200 documented body; no token 401;
                                                     with token 200 (error verdict: no API key on the box)
make docs-check                                   -> rc=2  "could not parse README fail-closed JSON block"
go test ./... -count=1                            -> rc=0  13s, all three packages ok
```

`INSTALL_SECONDS ≈ 60` (clone + build; build dominated by the toolchain download).
Agent destroyed (`bunker destroy --server bunker-mvp ae4ac4b7`), local key removed.

`make docs-check` on a fresh box is a **new P2 finding, root-caused**: the Makefile
needs `jq` (Makefile:84,86,89,102) but the README's Prerequisites name only Go and an
API key (0 mentions of jq in README.md), and the failure text blames the README
("could not parse README fail-closed JSON block") instead of the missing tool. Proof
of cause: `jq` absent → `jq_rc=127`; install `jq 1.7.1` into `$HOME/.local/bin`
(no sudo, user-space, the way a real user rescues it) → **`make docs-check` rc=0,
1s, all anchors + the fail-closed JSON replay PASS**. CI is green only because
GitHub's Ubuntu runner ships jq preinstalled — a clean-machine-only class, which is
the whole reason the fresh-machine leg exists.

## 5. Findings (board rows filed this tick)

- **GUARD-DF-004 [P2]** fresh-machine `make docs-check` fails on an undocumented `jq`
  prerequisite and misdiagnoses it as a README JSON parse error (evidence above).
  Fix: name `jq` in README Prerequisites + the Makefile-target row, and make the
  failure self-diagnosing (check for the tool, not just its output).
- **GUARD-DF-005 [P2]** per-source `Overrides` are unreachable from the CLI and the
  served surface: `cmd/guardd/main.go:108` hardcodes `guard.DefaultPolicy()` and no
  flag/env/file supplies a `Policy`, while `docs/threshold-calibration.md:111,156`
  prescribes "tighten that source with an Override" for the structured-object family.
  Only in-process Go consumers (crier) can configure one; task-router (Python, over
  HTTP — connectors/README.md:184) cannot. Fix: a `-policy <file.json>` / `GUARD_POLICY`
  loader on both surfaces, or scope the calibration line to in-process consumers.
- **GUARD-DF-006 [P3]** the verdict's own accounting hides the attribution call.
  Measured with `go run . stub` (the shipped `JevClassifier` pointed at a local stub
  through its public `Endpoint` field): a deliver verdict = **1** classifier call,
  `cost_usd` 0.0010 = true 0.0010; a quarantine verdict = **2** classifier calls,
  `cost_usd` 0.0010 vs true 0.0020. Live HTTP: deliver wall 0.25–0.29 s with
  `duration_ms` 248–290; caught wall 0.53–0.61 s with `duration_ms` 263–362 —
  overlapping ranges, so the field cannot distinguish a 1-call from a 2-call verdict
  while the user waits ~2x. Cause: `guard.go:154` sets `DurationMs` before the
  attribution re-classification at `guard.go:186-195`, and `guard.go:172` takes
  `CostUSD` from the first `Signals` only. README:226 documents the extra call but
  not that the fields exclude it.
- **GUARD-DF-007 [P4]** an `Overrides` key that matches no `Input.Source` is silently
  ignored (`policy.go:47-61`): a typo (`"webook"`) disables the tightening with no
  signal anywhere, against a project whose stated doctrine is "never silent".
- **GUARD-DF-008 [P3]** roll-up of still-live run-2/3 findings that were never filed
  as rows, re-proved at HEAD bcb6002 with exact commands: DF-G1 (`cmd/guardd/main.go:7`
  still says `POST /check {input}`; body `{"input":…}` → `400 content is required`),
  DF-G3 (unbounded `io.ReadAll(os.Stdin)` at `cmd/guardd/main.go:123` vs the 8 MiB HTTP
  cap at `:193`), DF-G4 (9 MB body → `400 bad request: unexpected EOF`, no limit named),
  DF-G6 (`-channel` inert: 3-run matrix `''`/`crier`/`task-router` → identical verdict,
  no channel in the Result, yet README:254 calls it a "classifier label").
- **GUARD-DF-009 [P2 infra]** SKIPPED-install-bunker: `bunker-qa.sh` cannot ship a
  battery (`UI_PORTS: unbound variable` → 0-byte `qa-run.sh`); installability was
  proved by the hand-walked fresh-agent leg instead (what remains unproven: the
  battery's own cells — ci-pass/upgrade/chaos — not the install path).

## 6. Performance (Step 2b) — measured, and why there is no PERF row

| operation | warm | notes |
|---|---|---|
| HTTP `/check`, deliver verdict (1 classifier call) | **0.25–0.29 s** (5 runs, client wall; `duration_ms` 248–290) | Jev round trip |
| HTTP `/check`, caught verdict (2 classifier calls) | **0.53–0.61 s** (5 runs; `duration_ms` 263–362) | attribution call |
| CLI/local floor (`GUARD_EGRESS_ENABLED=false`) | **0.00–0.01 s**, rc=2 | no network |
| `make docs-check` (fresh box) | 1–2 s (with jq) | README replay |
| `go test ./... -count=1` (fresh box) | 13 s | three packages |

The only felt cost is the designed classifier round trip — twice on caught messages by
design (README:226). Nothing local is slow enough to profile (user+sys ≈ 20–25 ms;
the wait is the remote model), so **no PERF row is filed**; "nothing here is slow
enough for a user to feel" is the honest result. The 2x is reported as the P3
accounting finding (GUARD-DF-006) because the verdict's own `duration_ms`/`cost_usd`
hide it — not because the wait itself is wrong.

## 7. What was NOT exercised (stated, not passed off)

- The `bunker-qa.sh` battery cells (ci-pass / upgrade / chaos-*) — blocked by the
  driver break above; only the install path was proved.
- A real classification on the fresh box: the agent has no OpenRouter key and
  credentials are never minted or copied into a bunker; the fresh box proves the
  fail-closed path, the dev box proves classification.
- `docs/ADOPTION.md`-style multi-host adoption, MCP surface (none exists).

ch:trace row=GUARD-DF-004..009 spec=docs/dogfood/2026-10-19-run4-findings.md
evidence=this file witness=bunker-mvp agent ae4ac4b7 (destroyed) verdict=SHIPPABLE
commit=bcb6002
