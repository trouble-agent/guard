# Dogfood integration report — tick `guard-dogfood-2026-11-18-00-54-47`, delivery 5

- Date: 2026-11-18 (UTC), product tree `HEAD` unchanged since delivery 1 of this tick
- Binary under test: fresh `go build ./cmd/guardd` from `HEAD` (`sha256 1a0e64b7…`), plus the repo's
  committed `bin/guardd` for the parity check below
- Skill: `coding-hermes-dogfood` (Step 2 real use, Step 2b performance, Step 3 install leg)
- Verdict: **SHIPPABLE** — no product defect found; two documentation gaps filed
  (`GUARD-DF-025`, `GUARD-DF-026`)

## What was different about this delivery

Deliveries 1–4 of this tick walked the contract (every route, every error arm), the corpus, the
override wiring and the size-rejection perf path. All of those replayed **synthetic** bodies. This
delivery used the product the way the skill says to: it fed guard **real user-facing prose** — six
of the repo's own documents, chunked at 4,000 bytes to agent-message size — through the documented
surfaces, and measured what each verdict costs in wall time and dollars.

Surface used: `guardd -serve` + `POST /check` with `X-Operator-Token` (the integrator path), and
`guardd -content …` (the same CLI shape `data/corpus/run_calibration.sh` drives). Egress stayed
enabled, i.e. the hosted classifier, exactly as a real deployment runs.

## Real-content decision distribution

21 chunks from `README.md`, `docs/user-guide.md`, `docs/operations.md`, `CHANGELOG.md`,
`skills/guard-usage/SKILL.md`, `docs/threshold-calibration.md` (`realprobe.sh`, `fields.sh`;
per-chunk rows in `benign-fields.jsonl`):

| route | n | score range | classified_bytes / bytes_seen | CLI median wall | mean cost |
|---|---|---|---|---|---|
| deliver | 18 (86%) | 0.17 – 0.42 | 2.0× | 607 ms | $0.000137 |
| review | 3 (14%) | 0.64 – 0.81 | 3.0× | 1113 ms | $0.000228 |
| block | 0 | — | — | — | — |
| errored | 0 | — | — | — | — |

The three review verdicts, all benign product documentation:

| chunk | score | attack_class | load-bearing |
|---|---|---|---|
| `README-02` | 0.81 | masquerade | rot13 |
| `README-04` | 0.64 | instruction_injection | — |
| `SKILL-03` | 0.67 | masquerade | — |

Two facts worth naming:

- **Distribution is bimodal with a wide dead zone.** No chunk landed between 0.42 and 0.64, so at
  the shipped `review_at 0.50` the filter separated real benign prose from "prose *about* encodings
  and injection" cleanly. The three review hits are all text that discusses injection/encodings —
  the review route is doing its job, not misfiring.
- **`README-02` sits 0.04 under `block_at`.** That chunk is the README section on encodings; the
  verdict is `masquerade` by way of the **rot13 decode view**, which is load-bearing on it. A benign
  page that documents encodings is therefore the closest thing this run found to a false positive,
  and it is a hardware-noise distance from block.

`classified_bytes` is 2.0× the input on every deliver verdict (the rot13 view is appended
unconditionally, `normalize.go:37`) and 3.0× on every non-deliver verdict — the extra 1.0× is the
raw re-classification used to attribute `load_bearing_normalizations` (`guard.go:255-278`).

## Performance: what a verdict costs (Step 2b)

Warm path — long-lived `guardd -serve`, keep-alive, n=6 per arm (`warm-http.sh`, `http-warm.jsonl`):

| payload | route | wall median | min – max | cost / message |
|---|---|---|---|---|
| 12 B ("hello, world") | deliver | **254 ms** | 240 – 378 | $0.0000263 |
| 3,687 B real prose | deliver | **261 ms** | 248 – 270 | $0.000124 |
| 4,032 B real prose | review | **541 ms** | 501 – 578 | $0.000226 |

Attribution isolated — same 3,687-byte benign chunk, two policies, n=3 (`policyexp.sh`,
`pol-detail.jsonl`): default bands (deliver) → **531 ms / $0.000124**; tightened bands (same content,
now review) → **932 ms / $0.000181**. Content held constant, so the delta is the second classifier
call: **+76% wall, +46% money for the same bytes**, and only because the verdict was non-deliver.

Cold/CLI: one `guardd -content` invocation per message costs ~2× the warm path (531 ms CLI vs 261 ms
warm serve for the same chunk) — each process pays its own TLS handshake. A caller gating a message
stream should use `/check`, not the CLI.

**Is any of this user-felt?** No. 254 ms for a tiny message and 261 ms for the same message carrying
3.7 KB means size is not the cost driver at these sizes; the hosted round-trip is. The review route
doubles that to 541 ms for 14% of real technical prose. That is below the threshold where a user in a
chat path would notice, and the money ($0.12–$0.23 per 1,000 messages) is noise. **No PERF row is
filed** — the honest finding is that these numbers were unpublished, not that anything is slow
(`GUARD-DF-025`).

## Determinism

Eight identical calls per chunk, identical policy (`variance.sh`, `var-detail.jsonl`):

| chunk | scores (n=8) | spread | routes |
|---|---|---|---|
| `user-guide-01` | 0.41 0.45 0.44 0.43 0.38 0.41 0.45 0.39 | **0.07** | 8× deliver |
| `README-02` | 0.82 0.79 0.81 0.80 0.80 0.80 0.79 0.79 | 0.03 | 8× review |

`docs/threshold-calibration.md:158` already warns that "scores drift run to run", but only as a
methodology caveat with no magnitude, and nothing tells a consumer what it means for band placement.
Measured: drift is up to 0.07, which is wider than the 0.05 margin between the worst benign chunk
(0.45) and `review_at 0.50`. No route change was observed in 8 calls — the risk is structural, not
demonstrated — so this is filed as a documentation gap at P5 (`GUARD-DF-026`), not a defect.

## Install leg

`bunker --server bunker-mvp list` at 02:1xZ: `7a98d021` (this tick's ephemeral agent) is **absent** —
teardown holds. `ci-e2e-01`, `cf-node` and `f6a5f076` (foreign, 00:08Z) are present and untouched.
The battery PASS recorded in delivery 1 stands: the product tree is byte-identical to the tree that
battery installed (`git diff --name-only` since that walk = board + log + evidence only), so re-running
a 25-cell battery against the same bytes would spend bunker-mvp disk to re-derive a known result.
This is an **inherited PASS with teardown re-verified on an independent surface**, stated explicitly
rather than left silent.

## Diagnostics

- Fresh `go build ./cmd/guardd` at `HEAD` is byte-different from committed `bin/guardd` only by the
  ldflags version stamp (`bin/guardd` reports `v0.2.0`, the fresh build `dev`) — no product drift.
- One probe instance from another lane was already listening on `127.0.0.1:18768`
  (`…/stand-in/docs/guard/probe-20261118/guardd`). Left alone; this delivery served its own instance
  on `127.0.0.1:18799` and stopped it afterwards.
- Gate note for future deliveries on this surface: a probe script containing `rm -rf` is refused by
  the harness before it runs. The probes here avoid it.

## Reproduce

    E=.coding-hermes/evidence/guard-dogfood-2026-11-18-00-54-47/delivery5
    go build -o /tmp/guardd ./cmd/guardd
    bash $E/realprobe.sh /tmp/guardd /tmp/df5 4000      # 21 real chunks
    bash $E/fields.sh    /tmp/guardd /tmp/df5           # adds normalization/cost attribution
    bash $E/policyexp.sh /tmp/guardd /tmp/df5 3         # deliver vs forced-review, same bytes
    bash $E/variance.sh  /tmp/guardd /tmp/df5 8         # determinism
    # warm path: /tmp/guardd -serve 127.0.0.1:18799 -tokenfile <0600 file>, then
    bash $E/warm-http.sh /tmp/df5 18799 6

## Rows filed

- `GUARD-DF-025` (P4) — publish the real-content cost/latency table; state that non-deliver verdicts
  pay a second classifier call (measured 261 → 541 ms warm, $0.000124 → $0.000226).
- `GUARD-DF-026` (P5) — quantify the drift (≤0.07 on identical input) in the consumer-facing docs and
  say what it implies for band placement.
