# Guard dogfood run 4b — 2026-10-19 (tick `guard-dogfood-2026-10-20-01-29-06`, SECOND executor)

Run 4a (`b05b8da`, same tick, first executor) covered the Go consumer, the HTTP surface, the
Python connector, `make docs-check` on a fresh box and the install leg. This run is the tick's
**second executor**: same HEAD (`bcb6002`), deliberately different angle — the **CLI wire shape**
and the **internals of the override door** (`policy.go`) — plus an independent re-proof of the
two rows run 4a filed and a fresh-agent install leg of its own.

Nothing here re-sweeps 4a: it re-proves where the row is a claim, and it takes the surface 4a
did not drive (`guardd -content` / `-source` / `-channel` on in-band payloads, and
`Override.FailClosed`).

## 0. Artifacts

| artifact | path |
|---|---|
| HEAD build (both runs) | `go build -o ~/guard-df-run4/bin/guardd ./cmd/guardd` → `go1.26.9`, mod `v0.1.1-0.20261019232500-bcb60026218a` |
| real consumer (outside the repo) | `~/guard-df-run4/consumer/` (`go.mod` + `replace` → the checkout; documented `Classifier` seam = deterministic stub) |
| payload scans | `~/guard-df-run4/out/scan.txt`, `out/border.txt`, raw JSON under `out/payloads/`, `out/border/` |
| CLI A/B matrix | `~/guard-df-run4/ab-cli.sh` → `out/ab-cli.txt` |
| HTTP battery | `~/guard-df-run4/http-battery.sh` → `out/http-battery.txt` |
| install leg (fresh agent) | `~/guard-df-run4/install-leg.sh` → `out/install-leg.txt` |
| render break re-proof | `~/guard-df-run4/out/gen.sh` (0 bytes), `out/gen.err` |

## 1. GUARD-DF-005 re-proved behaviourally on the CLI shape

The row (filed by 4a) says per-source `Overrides` are unreachable from the CLI/served surfaces
because `cmd/guardd/main.go:108` hardcodes `guard.DefaultPolicy()`. 4a's evidence was structural
(no loader exists). Here is the behavioural half — a **discriminating probe**, not a rerun:

Payloads were scanned live until one landed in the review band **above** the documented
override's `BlockAt 0.55` (B2; a webhook override, were it reachable, must quarantine it):

```
B2 plain             run1  score=0.79 route=review
B2 plain             run2  score=0.81 route=review
B2 plain             run3  score=0.78 route=review
B2 -source webhook   run1  score=0.79 route=review
B2 -source webhook   run2  score=0.77 route=review
B2 -source webhook   run3  score=0.80 route=review
B2 -source inbox     run1  score=0.81 route=review
B2 -source inbox     run2  score=0.76 route=review
B6 plain             score=0.84 route=review
B6 -source webhook   score=0.84 route=review
```

Same build, same message, in-process consumer with the README literal (`README.md:308-317`):

```
policy=readme-override            source="webhook" -> route=quarantine decision=block risk=high
policy=readme-override-control    source="inbox"   -> route=review     decision=allow risk=medium
policy=readme-override-nosource   source=""        -> route=review     decision=allow risk=medium
policy=readme-override-above-both source="webhook" score 0.90 -> route=quarantine (both bands)
```

Control that the flag plumbing itself works: `-fail-closed=false` + `GUARD_EGRESS_ENABLED=false`
→ `route=review reason="guard_error (fail-open)…"` vs `-fail-closed=true` → `quarantine`
(same process, same policy object). So the CLI reads flags into the policy fine — it just has no
way to install an `Overrides` map, and `policy.go` never falls back to a policy-id key.

**Verdict on the row: premise holds, now proven on both shapes.** File: row byte-untouched;
fresh evidence appended to `events.jsonl` (task_id `GUARD-DF-005`).

## 2. `-channel` is inert (independent re-proof, sub-evidence of GUARD-DF-008)

```
B2 -channel crier          score=0.83 route=review
B2 -channel nonsense-label score=0.81 route=review
B2 -channel + -source webhook score=0.79 route=review
```

`grep -rn 'Channel' --include=*.go` (non-test): `guard.go:72` (field), `cmd/guardd/main.go:97`
(flag), `cmd/guardd/main.go:133` (assigned into `Input`) — nothing reads it and no `Result`
field carries it. Re-proved on my own runs; no new row.

## 3. NEW: `Override.FailClosed` is exported, JSON-tagged, and never read

`policy.go:27-33` declares `FailClosed *bool \`json:"fail_closed,omitempty"\`` on `Override`;
`bandsFor` (`policy.go:47-61`) resolves it per source; `apply` (`policy.go:65`) **discards it**
(`blockAt, reviewAt, _ := p.bandsFor(source)`), and the error path (`guard.go:155-168`) keys on
`g.Policy.FailClosed` only. Deterministic consumer evidence (stub classifier returning an error):

| policy | source | expected if the field were honoured | observed |
|---|---|---|---|
| `FailClosed=true` + `Overrides{webhook:{FailClosed:false}}` | `webhook` | fail-open review | `quarantine` / fail-closed |
| same | `inbox` (control) | fail-closed | `quarantine` / fail-closed |
| `FailClosed=false` + `Overrides{webhook:{FailClosed:true}}` | `webhook` | fail-closed block | `review` / fail-open |
| same | `inbox` (control) | fail-open | `review` / fail-open |

Inert in **both** directions, each identical to its control → filed as **GUARD-DF-010 [P3]**
(exported API surface advertising a switch it never honours; sibling of GUARD-DF-007).

## 4. NEW: the verdict cannot report that a per-source override was applied

With an override in effect the JSON is indistinguishable from a base-band verdict: `policy`
stays the policy id (`"default"`, set at `guard.go:148`) and the reason text is the generic band
wording — `readme-override/webhook → "injection signals above block threshold"` reads exactly
like the default-band block at 0.90. An operator asking "why was this 0.60 webhook message
quarantined?" cannot answer from the verdict, and once GUARD-DF-005's loader lands, an override
typo/misfire leaves no trace. Filed as **GUARD-DF-011 [P4]** (doctrine anchor: `guard.go:15-17`
"never a silent default … it records which choice it made").

## 5. HTTP surface re-driven (CLI-side complement to 4a's table)

```
GET  /healthz (no token)                   200 {"status":"ok","egress_enabled":true}
POST /check  (no token)                    401 "unauthorized: missing or invalid X-Operator-Token"
POST /check  (wrong token)                 401 same
POST /check  {content}                     200 review (score 0.79, policy "default")
POST /check  {content,source:webhook}      200 review  <-- override unreachable on the wire too
POST /check  {content,source,channel,meta} 200 review  (channel accepted, ignored)
POST /check  {input:…}                     400 "content is required"      (DF-G1, in GUARD-DF-008)
GET  /check  (token)                       405 "POST only"
POST /check  not-json                      400 "bad request: invalid character 'o' in literal null (…)"
POST /check  9 000 000-byte body           400 "bad request: unexpected EOF"  (DF-G4, in GUARD-DF-008)
POST /nope                                 404 "404 page not found"
```

## 6. Install leg — own fresh agent, `bunker-qa.sh` still cannot ship a battery

`bunker-qa.sh` render break re-proved **without spawning an agent** (the generator hook):

```
$ bash ~/.hermes/scripts/bunker-qa.sh __gen-remote /home/kara/trouble-agent/guard
rc=1   bytes=0
stderr: /home/kara/.hermes/scripts/bunker-qa.sh: line 1061: UI_PORTS: unbound variable
```

Line-level root cause (adds to GUARD-DF-009): the unquoted render heredoc (line 1061,
`( cd / && cat <<EOF`, body lines 1062-2757) is expanded on the **controller** under
`set -uo pipefail` (line 44). Lines **2397-2407** are unescaped —

```
2397:  UI_PORTS="${BUNKER_QA_UI_PORTS:-}"
2398:  if [ -z "$UI_PORTS" ]; then        <-- aborts the render here (nounset)
2399:    UI_PORTS="3000 5173 8000 8080 8081 8766 8767 9000 9090"
2403:          UI_PORTS="$UI_PORTS $p"
2407:    UI_PORTS=$(echo "$UI_PORTS" | tr ' ' '\n' | sort -un | tr '\n' ' ')
```

— a later edit that lost the escape fence the surrounding lines keep (2288/2424/2436/2450/2454
are escaped). Line 2407's unescaped `$(…)` is the same gap's second hazard: had 2398 not aborted
first, a command substitution inside the body would have **executed on the controller** — the
class of the 2026-10-12 incident the fence comment at 1049-1060 warns about. Fix direction: escape
those five lines, or hoist `UI_PORTS` above the heredoc the way `QA_HEAD_SHA`/`QA_HEAD_PULL`
(1047-1048) are hoisted and inject the literal.

Substitute leg (the skill's sanctioned fresh-agent procedure) on **my own** agent
`701e4fac @ bunker-mvp`, TTL 90m — spawned, used, **destroyed** (`bunker destroy --server
bunker-mvp 701e4fac` → home archived, local key removed), list re-checked:

```
fresh box:  bunker-701e4fac, Linux 6.8.0; git/make/python3/curl present, jq MISSING,
            image go1.22.2  <  go.mod floor 1.26.9
clone       https://github.com/trouble-agent/guard.git  -> b05b8da == origin/main (public, no creds)
build       go build ./cmd/guardd           rc=0  45.94s (toolchain directive auto-downloads go1.26.9)
first run   printf 'hello, world' | ./guardd rc=2  byte-exact documented fail-closed JSON
serve       /healthz 200 documented body; POST /check no token 401; with token 200 (no key on box)
docs-check  make docs-check                 rc=2  "docs-check FAIL: could not parse README
                                                 fail-closed JSON block"  (GUARD-DF-004 reproduced)
tests       go test ./... -count=1          rc=0  12.97s, three packages ok
flags       ./guardd -h -> channel, content, fail-closed, pretty, serve, source, token, tokenfile
            (no policy/override loader — GUARD-DF-005 confirmed on a fresh box too)
INSTALL_SECONDS ≈ 46 (build dominates: toolchain download)
```

Guard-DF-004's premise is independently reproduced: `jq` is absent from the fresh box and
`make docs-check` fails with a message that blames the README, not the missing tool.

**Still unproven:** the `bunker-qa.sh` battery cells (ci-pass / upgrade / chaos-*), a real
classification on a fresh box (no key; credentials never minted or copied), and the Python
connector / Go client paths (covered by run 4a, not repeated).

## 7. Performance (Step 2b) — measured, no PERF row

| operation | measurement | note |
|---|---|---|
| CLI classify, deliver verdict (1 classifier call) | **343.9 ms ± 40.1** (hyperfine, 8 runs) | Jev round trip |
| CLI classify, caught verdict (2 calls by design) | **590.8 ms ± 29.7** (5 runs) | attribution re-classification |
| CLI local floor (`GUARD_EGRESS_ENABLED=false`, fail-closed) | **4.9-7.8 ms** (10 runs) | no network |
| process startup floor (`guardd -h`) | **3.7 ms ± 1.1** | 2.6-6.0 ms range |
| in-process policy evaluation (whole 11-case consumer, stub classifier) | **3.4 ms ± 0.5** | policy is free |

Every local path is single-digit milliseconds in a fresh process; the only wait a user feels is
the designed remote classifier round trip, and the 2x on caught messages is already filed as the
accounting gap GUARD-DF-006 — not a slowdown. **No PERF row**: nothing here is slow enough to
profile, and a row for it would devalue the real findings.

## 8. ch:trace

```
ch:trace row=GUARD-DF-010,GUARD-DF-011 spec=guard.go,policy.go,cmd/guardd/main.go
  wave=tick guard-dogfood-2026-10-20-01-29-06 (second executor)
  test=~/guard-df-run4/{ab-cli.sh,http-battery.sh,install-leg.sh}+consumer
  doc=docs/dogfood/2026-10-19-run4b-second-angle.md
  evidence=docs/dogfood/2026-10-19-run4b-second-angle.md + .coding-hermes/board/events.jsonl
  witness=fresh agent 701e4fac @ bunker-mvp (destroyed) verdict=SOUND commit=bcb6002
  memory=none:not-written
```
