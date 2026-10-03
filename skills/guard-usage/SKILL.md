---
name: guard-usage
description: Use when integrating or running the guard message-security filter (CLI, guardd HTTP service, or Python connector) — entry points, prerequisites, and the pitfalls that cost time.
---

# guard — usage for agents

**What it is:** a lightweight prompt-injection filter for untrusted text.
One Go core, three consumption shapes, one `Result` contract. Route on
`result.route` + `result.constraints`, never on `decision` alone.

## Prerequisites (not in the README — learn from this)

- Go ≥1.24 to build (`go build ./...`); zero external deps.
- An OpenRouter API key for ANY classification (Jev decisions model at
  `openrouter.ai/api/alpha/decisions`). Key load order:
  `$OPENROUTER_API_KEY` → `~/.hermes/.env` → `~/9router-deploy/.env.shared`
  (regex `sk-or-v1-…`, tried in order on 401/402/429). An expired key in the
  first file yields opaque `guard_error (fail-closed): key rejected: http 401`
  block verdicts — check `errored` in the Result when a block looks wrong.

## Entry points

```bash
go build -o bin/guardd ./cmd/guardd
printf '%s' "$msg" | ./bin/guardd                    # CLI, stdin
./bin/guardd -content "$msg" -source github_pr       # CLI, one-shot
./bin/guardd -serve 127.0.0.1:8768 -tokenfile <0600-file>  # HTTP (preferred; or GUARD_TOKEN env)
./bin/guardd -serve 127.0.0.1:8768 -token <secret>         # dev: works but warns (token visible in ps)
```

```python
import sys; sys.path.insert(0, 'connectors/python')
from guard_client import Guard
g = Guard(endpoint='http://127.0.0.1:8768', fail_mode='open')
v = g.check(content, source='task-router', channel='router')
if v.errored: ...        # outage path — check BEFORE honouring constraints (GUARD-DF-001)
if not v.deliver: ...    # quarantine/review
```

## Verdict semantics that matter

- `route`: deliver | review | quarantine — the primary output.
- `constraints`: allow_tools/network/secrets for the handling agent.
- `errored: true` ⇒ every other field describes the ERROR verdict, not a
  classification (attack_class is absent, score is 0).
- `normalizations` includes `rot13` on ~every result: ROT13 is applied
  unconditionally by design; the field does NOT mean "this payload was
  encoded" (GUARD-DF-002).
- `attack_class` is noisy near the threshold — benign text often carries
  masquerade at 0.1–0.4. Trust `route`, not the label, at the margin.

## Run commands

```bash
go build ./... && go vet ./... && go test ./... -count=1   # suite runs in ~5ms, no network needed
```

## Performance (measured 2026-10-02)

~260–330 ms per verdict warm, ≈ all of it the Jev API round trip (user+sys
≈ 20 ms locally). Cold first call 0.3–0.9 s. Cost ≈ $0.000027/verdict.
Don't profile the Go code first — the network is the latency.

## Pitfalls

- HTTP /check without `X-Operator-Token` → 401 by design; /healthz is open.
- Fail-open (`fail_mode='open'`) does NOT deliver on outage: it returns
  route=review, errored=True, constraints={} — a consumer that honours
  constraints unconditionally strips tools/secrets during outages.
- The Python kwarg is `endpoint=`, not `base_url=`; default endpoint from
  env `GUARD_ENDPOINT`, else `http://127.0.0.1:8768`.
- Malformed JSON to /check → HTTP 400 with a Go json parse message (readable).
- go.mod says go 1.24; newer toolchains are fine.
