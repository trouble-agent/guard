# Guard — diagnostic trail (dogfood 2026-10-02)

How the thing is built, why, what bit during real use, and the right way to
use it. Read this before integrating guard anywhere.

## Architecture as experienced

One Go module, three surfaces, one `Result`:

- `guard.go` — the Guard: normalise → classify → policy → route. The policy
  layer (policy.go) owns thresholds (`BlockAt 0.85 / ReviewAt 0.50`) and the
  quoted-downgrade rule; classification never decides the route alone.
- `normalize.go` — decode base64/hex/URL/ROT13, strip zero-widths. ROT13 is
  applied to EVERY message on purpose (normalize.go:152): a bare ROT13 payload
  once scored 0.32 = delivered. The label noise that cost is FIXED
  (GUARD-DF-002): `normalizations` = what ran (rot13 on ~everything),
  `load_bearing_normalizations` = the decoded variant actually drove the
  verdict — read the second for "why it was caught".
- `jev.go` — the classifier: OpenRouter's `/api/alpha/decisions` (NOT
  /chat/completions — a decisions model 400s there; that's "the usual first
  mistake" per the code comment). Keys load env → ~/.hermes/.env →
  9router-deploy/.env.shared, regex-extracted `sk-or-v1-…`, deduped, tried in
  order on 401/402/429.
- `cmd/guardd` — CLI + HTTP. Since GUARD-001 (c796feb): token required for
  /check, loopback default bind, /healthz open. Since REVIEW-GUARD-001: the
  token resolves GUARD_TOKEN env > `-tokenfile` (0600 enforced) > `-token`
  (dev; warned on stderr — argv leaks via ps).

## Errors hit during real use, and what each actually meant

1. `KeyError: 'attack_class'` parsing fresh-box output → the verdict was a
   fail-closed guard_error block, which omits attack_class. Meaning: the KEY
   was dead, not the JSON shape. Lesson: check `errored` before trusting any
   field of a block verdict.
2. Fresh-box `http 401` on every classification while the dev box worked →
   the key I transferred came from the FIRST file in the load order
   (~/.hermes/.env), which was expired; guardd's own fallback to
   .env.shared is why the dev box worked. Lesson: when replicating guardd's
   environment elsewhere, replicate the FILE SET, not one variable — and the
   "key rejected" message never tells you which key or which file failed.
3. `TypeError: unexpected keyword argument 'base_url'` on the Python
   connector → the parameter is `endpoint=` (guard_client.py:104), and the
   default is the env var GUARD_ENDPOINT or :8768. The kwarg name differs
   from the transport name; read the constructor before wiring.
4. Fail-open probe returned deliver=False → not a bug in the probe: that is
   the connector's actual outage contract today (GUARD-DF-001).

## The right way to use guard (as of this run)

- Bind consumers to `route` + `constraints`, exactly as docs/ADOPTION.md says
  — but ALSO branch on `errored` if you use fail_mode='open', or an outage
  silently strips capabilities (row GUARD-DF-001 tracks the fix).
- Never trust `attack_class` near the threshold; README's own honest-
  limitations section says the same and the dogfood battery agreed (benign
  text labelled masquerade at 0.09–0.37 several times).
- For the HTTP shape, always pass an explicit loopback host and supply the
  token via GUARD_TOKEN env or a 0600 `-tokenfile` (never `-token` in prod —
  ps leaks it; guardd warns).
- Expect ~260–330 ms per verdict, essentially all of it the Jev decisions
  API round trip; budget for it in synchronous paths.

## Environment facts from the run

- go 1.26.5 builds the module (go.mod says 1.24) with zero new deps; vet and
  the full test suite are clean in ~5ms — the suite is fast enough for CI
  (which is exactly what GUARD-002 asks for).
- The live key used by dev-box runs lives in 9router-deploy/.env.shared
  (OPENROUTER_API_KEY_FLEET); ~/.hermes/.env's OPENROUTER_API_KEY was expired
  at run time.
