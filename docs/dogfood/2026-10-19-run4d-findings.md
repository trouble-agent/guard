# Guard dogfood run 4d — 2026-10-19 (tick `guard-dogfood-2026-10-20-01-29-06`, FOURTH executor)

This executor was **delta-driven**: while it ran, the tick's own rows were fixed and pushed —
`2a2fc30` (GUARD-DF-010, per-source `Override.FailClosed`), `e858041` (GUARD-DF-006, attribution
cost/latency accounting) and `6836cc8` (a trace-carrier commit for the first). So the real-use pass
became the **independent re-proof of both fixes**, plus the operator/parity surfaces no earlier
run had touched.

Artifact discipline: every leg ran against a **clean `git archive HEAD` copy** at
`/home/kara/guard-df-run4/head-4d2` (binary `bin/guardd-4d2`, suite green in 4.0 s), never the
working tree — a sibling lane had uncommitted edits to `cmd/guardd/main.go`, `guard.go` and their
tests at the start of this run and had withdrawn them by the end, so they are neither measured nor
judged here, and nothing of theirs was staged.

## 1. GUARD-DF-010 (per-source `Override.FailClosed`) — VERIFIED from the consumer side

`bin/dfconsumer-4d` (documented `Classifier` seam, deterministic error-returning stub):

| case | policy | source | observed |
|---|---|---|---|
| A1 | FailClosed=true, `webhook:{FailClosed:false}` | webhook | **review / allow**, `guard_error (fail-open)` |
| A2 | same | inbox (control) | quarantine / block, fail-closed |
| A3 | same | "" (no source) | quarantine / block, fail-closed |
| B1 | FailClosed=false, `webhook:{FailClosed:true}` | webhook | **quarantine / block**, fail-closed |
| B2 | same | inbox (control) | review / allow, fail-open |
| C1 | `webhook:{BlockAt:0.55}`, score 0.60 | webhook | quarantine / block (score path untouched) |
| C2 | same | inbox | review / allow |
| D1 | `webhook:{BlockAt:0.55, FailClosed:false}`, classifier error | webhook | review / allow |
| D2 | same, score 0.60 | webhook | quarantine / block (both fields coexist) |
| D3 | same, classifier error | inbox (control) | quarantine / block |
| E1 | `DefaultPolicy()`, classifier error | task (no override) | quarantine / block (policy stance kept) |

Pre-fix (run 4b, HEAD `bcb6002`) A1 and B1 were byte-identical to their controls — the override was
ignored. All ten cases now match the fix's documented contract. Full JSON:
`~/guard-df-run4/out/consumer-4d.json`.

## 2. GUARD-DF-006 (accounting) — RE-PROVED with a before/after pair

`bin/stubprobe-4d`: the shipped `JevClassifier` pointed at a local `httptest` stub through its
public `Endpoint` field — cost **0.0010 per call**, plus an artificial **120 ms** lag on the
attribution call, so the verdict's own fields can be compared with the truth:

```
payload                  route      calls    reported_cost  true_cost      reported_ms  client_wall
benign (deliver)         deliver    1        0.0010         0.0010         0            1ms
injection (quarantine)   quarantine 2        0.0020         0.0020         240          241ms
quotation (review)       quarantine 2        0.0020         0.0020         240          242ms
```

Pre-fix baseline (run 4b, same instrument): deliver 0.0010 = 0.0010; **caught: reported 0.0010 vs
true 0.0020**, and `duration_ms` stopped at the first call. The bill now covers both calls and the
attribution lag. (The third row's stub always answers `quoted=0.1`, so no downgrade fires — it is a
two-call case like the second.)

**One stale consumer remains:** `skills/guard-usage/SKILL.md` still documents the *old* semantics —
filed as **GUARD-DF-013**. The fix commit itself says so ("docs fix left out of scope").

## 3. NEW — GUARD-DF-014: `listening on <addr>` is logged before the bind

```
$ guardd -serve 0.0.0.0:18806          # a second server, port already bound
guardd: token loaded from GUARD_TOKEN
2026/10/19 21:13:51 WARNING: classifier egress is enabled; ...
2026/10/19 21:13:51 guardd listening on 0.0.0.0:18806 (POST /check [X-Operator-Token required], GET /healthz)
2026/10/19 21:13:51 serve: listen tcp 0.0.0.0:18806: bind: address already in use
$ echo $? -> 1
```

The success banner comes first (`serveHTTP` logs at `cmd/guardd/main.go:219`, then serves at `:220`),
so a launcher, unit file or log-scraper keying on that line reads a failed start as a healthy one.
`/healthz` remains the correct readiness probe and the exit code is honest, so the impact is bounded
— but it is the same false-green class the project's own doctrine rejects. Acceptance: a failed bind
prints no success banner (bind via `net.Listen` first, log after — or reword to "attempting to
listen on").

## 4. Operator startup ladder — every documented leg holds

| leg | result |
|---|---|
| `-serve` with no token | rc=1, `no token provided: refusing to serve an unauthenticated /check; set GUARD_TOKEN, or pass -tokenfile <path> (0600)` |
| `-tokenfile` mode 0644 | rc=1, `mode 0644 is too open (group/world bits set); chmod 600 it first` |
| `-tokenfile` mode 0600 | starts, `token loaded from tokenfile`, `/healthz` ok, `/check` 200 with the file token |
| `GUARD_TOKEN` + staged tokenfile | env wins: env token 200, file token **401** on the same server |
| bare `:port` | binds `127.0.0.1` only |
| explicit `0.0.0.0:port` | binds `*` |
| port already in use | rc=1 + the bind error (but see GUARD-DF-014) |
| `-token` on argv | the documented loud `ps`-visibility warning |
| `GUARD_EGRESS_ENABLED=false` | documented WARNING; `/healthz` → `egress_enabled:false`; `/check` → HTTP **200** in-band error verdict (`guard_error (fail-closed): hosted classifier egress is disabled …`); the process keeps serving |
| `/healthz` methods | GET/POST/PUT/DELETE/OPTIONS/PATCH/HEAD all 200 (documented: no method restriction) |
| path shapes | `/check` 401-before-405 (auth first); `/nope` 404; `/check/` 404; `/check?x=1` 200; empty body 400 `bad request: EOF`; `{"content":null}` 400 `content is required` |

## 5. In-process vs HTTP parity — the documented "one-line change" holds

`bin/parity-4d`: the same input through the in-process core and through `client.New(...).Check`
against a live `guardd`:

```
benign     route(ip=deliver http=deliver) score(ip=0.08 http=0.07) class(ip=masquerade http=masquerade)
           constraints agree  policy(ip=default http=default)  wall(ip=329ms http=240ms) hop_delta=-88ms  fields_agree=true
injection  route(ip=quarantine http=quarantine) score(ip=0.98 http=0.98) class(ip=instruction_injection http=instruction_injection)
           constraints agree  wall(ip=555ms http=546ms) hop_delta=-9ms  fields_agree=true
quoted     route(ip=review http=review) score(ip=0.97 http=0.97) class agree
           constraints agree  wall(ip=545ms http=551ms) hop_delta=+5ms  fields_agree=true
```

Every compared field (route, decision, risk, errored, policy, provider, attack_class, constraints)
agrees; the HTTP hop (±88/−9/+5 ms against a 240–555 ms remote classifier) is inside the noise, so
the documented zero-hop in-process advantage is unmeasurable at this latency. **No PERF row.**

## 6. Install leg — full battery at the fix-bearing HEAD

`bunker-qa.sh` renders (rc=0, 119,363 bytes) and the battery ran on fresh agent `3ea7d415` @
bunker-mvp at HEAD `6836cc8` (evidence `/tmp/bunker-qa-evidence-20261020T022122Z-763314.jsonl`):
repo-head OK · toolchain-bootstrap OK · fresh-install OK · ci-pass OK (act 1 job green) · upgrade OK
(v0.1.0 tree → HEAD reinstall clean) · chaos-resource PASS (3G cap) · chaos-errorpath OK (rc=2) ·
docker-deploy / ui-probe / chaos-shutdown / chaos-corruption **N/A** by repo shape ·
**chaos-disconnect UNVERIFIED** (third consecutive observation → event on QA-GUARD-005) · collect OK,
agent destroyed.

## 7. Performance (Step 2b) — measured, no PERF row

| operation | measurement |
|---|---|
| startup → first 200 `/healthz` (5 fresh processes) | **13–15 ms** (the operator's "can I send traffic") |
| CLI local floor (egress disabled, no network) | 3.8 ms ± 0.4 (hyperfine, 10 runs) |
| clean-copy suite (`go test ./... -count=1`) | 4.0 s |
| full battery wall | ~4 min |
| HTTP hop vs in-process | −88 / −9 / +5 ms vs 240–555 ms remote → noise |

Nothing a user would feel; the only real wait is the remote classifier.

## 8. Not exercised (stated, not passed off)

- docker / UI / state chaos cells — N/A by repo shape (no compose file, no web UI, no db/state files).
- A real classification on a bunker agent — no key there, and credentials are never minted or copied.
- The sibling lane's in-flight edits — uncommitted and then withdrawn; explicitly NOT measured.

## 9. ch:trace

```
ch:trace row=GUARD-DF-013,GUARD-DF-014 (+events on GUARD-DF-010, GUARD-DF-006, QA-GUARD-005)
  spec=skills/guard-usage/SKILL.md,cmd/guardd/main.go,guard.go,policy.go
  wave=tick guard-dogfood-2026-10-20-01-29-06 (fourth executor)
  test=~/guard-df-run4/{ops-ladder.sh,parity.sh,startup-4d.sh,stubprobe-4d,consumer-4d}
  doc=docs/dogfood/2026-10-19-run4d-findings.md
  evidence=docs/dogfood/2026-10-19-run4d-findings.md + /tmp/bunker-qa-evidence-20261020T022122Z-763314.jsonl
  witness=clean git-archive build of HEAD 6836cc8 + fresh agent 3ea7d415 (destroyed) verdict=SOUND
  memory=none:not-written
```
