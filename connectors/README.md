# Connectors

The guard is the fleet's **input-sanitization layer** — the equivalent of a
parameterised query for agent/message inputs. The core does the judgement; a
*connector* is the thin adapter that makes it a one-liner in a given language
or framework. Same `Result` shape everywhere, so a verdict is portable.

## The contract (all connectors speak this)

Request — `POST /check` (or the library call):

```json
{ "id": "…", "source": "webhook", "channel": "crier", "content": "<untrusted>", "meta": {} }
```

Response — the `Result`:

```json
{
  "decision": "block", "risk_level": "high",
  "route": "quarantine",
  "attack_class": "instruction_injection", "score": 0.98,
  "constraints": { "allow_tools": false, "allow_network": false, "allow_secrets": false },
  "normalizations": ["rot13"], "load_bearing_normalizations": ["rot13"],
  "provider": "jev", "model": "…", "cost_usd": 0.000027
}
```

**Bind to `route` and `constraints`, not to `decision`.** `route` is the
destination (`deliver | review | quarantine`); `constraints` is the capability
set the handling agent may hold. A wrong verdict is then survivable: a false
negative gets a tool-less reader, a false positive gets a slightly dumber one.

One exception: when `errored` is set (an outage verdict — see Fail mode
below), branch on `errored` first. `constraints` on an outage verdict is
connector-specific and cannot tell an outage from a policy.

## HTTP contract

Exact behaviour of `guardd -serve`'s `/check` endpoint (from `cmd/guardd/main.go`,
`newHandler`). The result body is the same `Result` JSON as above.

**Order of checks** — auth first, then method, then body. A rejected request
reveals nothing about the later stages:

| # | Trigger | Status | Body | Code |
|---|---------|--------|------|------|
| 1 | Missing or wrong `X-Operator-Token` header | `401` | `unauthorized: missing or invalid X-Operator-Token` | main.go:183 |
| 2 | Method other than `POST` | `405` | `POST only` | main.go:187 |
| 3 | Malformed / truncated JSON body | `400` | `bad request: <parse error>` | main.go:192 |
| 4 | JSON decodes but `content` is empty | `400` | `content is required` | main.go:196 |

**Limits:**

- **Body size**: capped at **8 MiB** via `io.LimitReader(r.Body, 8<<20)`
  (main.go:191). A larger body is silently truncated mid-read; the truncation
  breaks the JSON, so an oversized request surfaces as case 3 (`400 bad
  request: ...`), not a dedicated 413.
- **Classification timeout**: **120s** per request via
  `context.WithTimeout(r.Context(), 120*time.Second)` (main.go:199). The CLI
  form uses the same 120s budget (main.go:128), so CLI and HTTP verdicts agree
  on timing too.

**`GET /healthz`** — no auth, always `200` (main.go:172-178) with a JSON body
`{"status":"ok","egress_enabled":<bool>}`, so liveness probes work without the
shared secret. It never classifies anything. The `egress_enabled` field mirrors
the `GUARD_EGRESS_ENABLED` startup flag.

**Note on truncation ordering**: auth (case 1) is checked before the body is
read, so an unauthenticated request cannot spend server resources reading a
huge body.

## Available connectors

| Consumer | Language | Connector | Transport |
|---|---|---|---|
| crier | Go | `github.com/trouble-agent/guard` (in-process) or `.../guard/client` | in-process or HTTP |
| hermes-dagger | Go | `.../guard/client` | HTTP |
| task-router | Python | `connectors/python/guard_client.py` | HTTP |
| anything / shell | any | `guardd` CLI (`printf … \| guardd`) | stdin/stdout |

Go consumers that run in-process import the core and get zero network hops;
Go consumers that share a `guardd` instance use `client`. Both return the same
`guard.Result`, so switching between them is a one-line change.

## Fail mode — choose per surface, never by default

A guard that cannot produce a verdict must do something explicit:

- **`closed`** — the check fails loudly; nothing is delivered. Use where a
  missed injection is worse than a stall (control paths, code-action ingress).
- **`open`** — the check degrades to a flagged **REVIEW** verdict: `route`
  review, `deliver` false, `errored` set. An outage becomes "a human looks" —
  never "no protection", never "work stops", and never a silent allow. What
  the outage verdict does with `constraints` is a per-connector choice
  (below); whatever it says, a consumer running `open` must branch on
  `errored` first and never auto-deliver an outage verdict.

Both are implemented in every connector (`fail_mode=` in Python, `FailMode` in
Go, `-fail-closed` in the CLI) and both are recorded on the result. On outage
capabilities the connectors deliberately diverge — this is documented, not
hidden:

| Connector | `open` outage verdict | `decision` | `constraints` |
|---|---|---|---|
| Python `guard_client.py` (`fail_mode='open'`) | `route='review'`, `errored=True` | `error` | **preserved** — `CONSTRAINTS_FAIL_OPEN` grants `allow_tools: true`, `allow_secrets: true` |
| Go `client` (`FailOpen`) | `RouteReview`, `Errored: true` | `allow` | **withheld** — zero `Constraints{}`, every capability false |
| `guardd` CLI (`-fail-closed=false`) | `route=review`, `errored` | `allow` | **withheld** — constraints stay all-false |

Why the divergence: the Python connector serves task-router, whose doctrine
is "fail-open is sacred; never block the scheduler" (GUARD-DF-001). An outage
verdict that withheld capabilities was indistinguishable from a real
block-all policy and silently stripped tools+secrets from every task for the
whole outage — so the outage verdict PRESERVES capabilities and flags itself
(`errored=True`, `decision='error'`) instead. The Go client (crier,
hermes-dagger) keeps the conservative stance: an outage verdict never grants
anything. Read that as one contract with two capability stances, not as
drift: both return `route` review with `errored` set, and both require the
consumer to gate on `errored`. Do not "reconcile" one side to the other
without re-reading GUARD-DF-001 and both consumers' gating; a new connector
picks one of the two stances and documents it.

## Adding a connector (this is the point)

A connector is intentionally trivial — that is what makes the surface powerful.
To add language or framework N:

1. **HTTP client** — POST `{"content", "source", "channel"}` to `/check`, decode
   the `Result`. That is the whole protocol.
2. **A verdict view** — expose `route` and `constraints` as fields, plus one
   boolean (`deliver` / `IsSafe()`), so callers never parse the raw map.
3. **Fail mode** — implement `closed` (raise/return error) and `open` (return
   a flagged REVIEW verdict: `errored` set, never a silent allow). For the
   outage verdict's `constraints`, pick one of the two existing stances —
   withhold everything (Go client) or preserve-and-flag
   (`CONSTRAINTS_FAIL_OPEN`, Python) — and document which you chose. Do not
   invent a third behaviour.
4. **A framework hook** (optional but encouraged) — a middleware/adapter at the
   consumer's ingress choke point, so guarding is a wrap, not a call site
   scattered through the code.

If a new consumer needs something the contract lacks (a new class, a new
route), extend the **core** — not the connector. Connectors stay thin.

## What NOT to do

- Do not reimplement classification in a connector. One core, many callers.
- Do not ship a connector that returns "allow" when the service is unreachable
  and calls it fail-open. That is silent protection loss — use REVIEW.
- Do not let a connector grant `allow_secrets` on a real verdict. No policy
  path ever should. The deliberate exception is the Python fail-open outage
  verdict, which preserves capabilities and flags the outage with
  `errored=True` (see Fail mode above) — consumers gate on `errored`, not on
  the grant.
