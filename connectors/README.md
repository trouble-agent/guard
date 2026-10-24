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

Only `source` and `content` affect the verdict. `id`, `channel`, and `meta` are
accepted-but-ignored pass-through fields: they are not echoed, do not affect
classification, and produce no error.

Response — the `Result`:

```json
{
  "decision": "block", "risk_level": "high",
  "route": "quarantine",
  "attack_class": "instruction_injection", "score": 0.98,
  "policy": "tight-webhook", "override_source": "webhook",
  "effective_block_at": 0.55, "effective_review_at": 0.40,
  "constraints": { "allow_tools": false, "allow_network": false, "allow_secrets": false },
  "normalizations": ["rot13"], "load_bearing_normalizations": ["rot13"],
  "provider": "jev", "model": "…", "cost_usd": 0.000027
}
```

`policy` names the decision table; `override_source` names the row INSIDE it that
decided this verdict (absent when the policy's own bands did) and
`effective_block_at` / `effective_review_at` are the bands actually compared
(GUARD-DF-011). See README "Which row of the table decided".

**Bind to `route` and `constraints`, not to `decision`.** `route` is the
destination (`deliver | review | quarantine`); `constraints` is the capability
set the handling agent may hold. A wrong verdict is then survivable: a false
negative gets a tool-less reader, a false positive gets a slightly dumber one.

One exception: when `errored` is set (an outage verdict — see Fail mode
below), branch on `errored` first. `constraints` on an outage verdict is
connector-specific and cannot tell an outage from a policy.

**Quoted content is a downgrade, not an exemption.** When the score reaches
the block band but the classifier is confident the content is a quotation or
worked example (`sig.Quoted` >= `Policy.QuotedDowngrade`, default `0.80`),
the quarantine is downgraded to `review` — never to `deliver`
(`policy.go:243-249`). The score is not reduced and capabilities are not
restored; the consumer still routes a human look.

## HTTP contract

Exact behaviour of `guardd -serve`'s `/check` endpoint (from `cmd/guardd/main.go`,
`newHandler`). The result body is the same `Result` JSON as above.

**Order of checks** — auth first, then method, then body. A rejected request
reveals nothing about the later stages:

| # | Trigger | Status | Body | Code |
|---|---------|--------|------|------|
| 1 | Missing or wrong `X-Operator-Token` header | `401` | `unauthorized: missing or invalid X-Operator-Token` | main.go:220 |
| 2 | Method other than `POST` | `405` | `POST only` | main.go:224 |
| 3 | Malformed / truncated JSON body | `400` | `bad request: <parse error>` | main.go:228-229 |
| 4 | JSON decodes but `content` is empty | `400` | `content is required` | main.go:233 |

**Limits:**

- **Body size**: capped at **8 MiB** via `io.LimitReader(r.Body, 8<<20)`
  (main.go:228). A larger body is silently truncated mid-read; the truncation
  breaks the JSON, so an oversized request surfaces as case 3 (`400 bad
  request: ...`), not a dedicated 413.
- **Classification timeout**: **120s** per request via
  `context.WithTimeout(r.Context(), 120*time.Second)` (main.go:236). The CLI
  form uses the same 120s budget (main.go:165), so CLI and HTTP verdicts agree
  on timing too.

**`/healthz`** — no auth, always `200` (GET and other methods are accepted; the
handler does not restrict the method; `cmd/guardd/main.go:209-215`) with a JSON
body `{"status":"ok","egress_enabled":<bool>}`, so liveness probes work without
the shared secret. It never classifies anything. The `egress_enabled` field
mirrors the `GUARD_EGRESS_ENABLED` startup flag.

**Note on truncation ordering**: auth (case 1) is checked before the body is
read, so an unauthenticated request cannot spend server resources reading a
huge body.

## Operational knobs (endpoint, token, timeout)

The knobs a new integrator actually sets. Names are exact; defaults are what
the code does today.

**`GUARD_ENDPOINT`** — base URL of the guardd HTTP service. Default
`http://127.0.0.1:8768` (loopback, port 8768). The Python connector resolves
it as CLI `--endpoint` flag → `GUARD_ENDPOINT` environment variable → the
built-in default ([`connectors/python/guard_client.py:264`](python/guard_client.py),
[`connectors/python/guard_client.py:308-310`](python/guard_client.py)). The
library form is `Guard(endpoint=...)`, falling back to the module default
([`connectors/python/guard_client.py:38`](python/guard_client.py)), which
snapshots `GUARD_ENDPOINT` at import time — set the variable before importing,
or pass `endpoint=` explicitly, if you resolve late. The Go client has no
environment default: `client.New(endpoint)` takes the URL explicitly
([`client/client.go:63-65`](../client/client.go)).

**`GUARD_TOKEN`** — the shared secret every `/check` must present as the
`X-Operator-Token` header; guardd answers `401 unauthorized: missing or
invalid X-Operator-Token` without it (see HTTP contract above). Clients
resolve it explicit-argument first (Python `token=` constructor argument /
`--token` flag, Go `Client.Token` field), else from the `GUARD_TOKEN`
environment variable ([`connectors/python/guard_client.py:180`](python/guard_client.py),
[`client/client.go:67-75`](../client/client.go)). There is no default value.

**guardd's own token precedence (serving side)** — when `-serve` starts, it
resolves the secret once at startup, env > tokenfile > argv
([`cmd/guardd/main.go:148`](../cmd/guardd/main.go),
[`cmd/guardd/main.go:305`](../cmd/guardd/main.go)):

1. `GUARD_TOKEN` environment variable — preferred; it wins even when a
   tokenfile is staged, so an operator can override ad hoc without editing
   the file ([`cmd/guardd/main.go:307-308`](../cmd/guardd/main.go)).
2. `-tokenfile <path>` — file contents, surrounding whitespace trimmed. The
   file must be owner-only (`0400` or `0600`); group- or world-accessible
   modes are refused at startup
   ([`cmd/guardd/main.go:333-335`](../cmd/guardd/main.go)).
3. `-token <secret>` — works, but guardd warns loudly on stderr: an argv
   secret is world-readable via `ps` and `/proc/<pid>/cmdline`.

With none of the three, guardd refuses to serve at all — it never runs an
unauthenticated `/check`.

**Timeout** — Python bounds each check with the `timeout` constructor
argument (default `120.0` seconds) or the CLI `--timeout` flag (default 120)
([`connectors/python/guard_client.py:172-175`](python/guard_client.py),
[`connectors/python/guard_client.py:272-278`](python/guard_client.py)); the
Go client's `New()` sets the same 120s ([`client/client.go:63-65`](../client/client.go)).
guardd's own classification deadline is fixed at 120s (HTTP contract above).
A timeout is a connector-side failure and follows the fail-mode rules below.

**Fail mode** — Python: `fail_mode` constructor argument (`"closed"` is the
default; `"open"` is the alternative) or the CLI `--fail-open` flag. In
`open` mode any check failure (transport, timeout, HTTP error status, bad
JSON — [`connectors/python/guard_client.py:212`](python/guard_client.py))
becomes a flagged REVIEW verdict that preserves capabilities and sets
`errored=True`; in `closed` mode the same failure raises `GuardError` and
nothing is delivered. Full semantics and the Go/CLI comparison are in
"Fail-open semantics" and "Fail mode" below.

## Exit codes (two schemes — do not mix them)

The Python CLI and the guardd CLI answer different questions with their exit
statuses. Shell integrators must pick the map that matches the binary they
invoked.

**Python CLI** (`guard` console script from `connectors/python/`, or
`python connectors/python/guard_client.py`) — the exit code branches on the
verdict's `route`, so shell callers never parse JSON
([`connectors/python/guard_client.py:290-293`](python/guard_client.py)):

| Exit | Meaning |
|---|---|
| `0` | route `deliver` — safe to deliver unchanged |
| `1` | route `review` — a human looks (includes the fail-open outage REVIEW verdict) |
| `2` | route `quarantine` — blocked |
| `3` | no verdict — `GuardError` was raised (transport failure, timeout, HTTP error status such as 401, malformed response JSON; any failure in `closed` fail mode). Nothing is delivered. |

Usage errors (no content, unknown flag) exit with argparse's own code `2`
plus a usage message on stderr, and print no verdict JSON on stdout —
distinguishable from a quarantine verdict, which always prints the verdict
JSON first.

**guardd CLI** (one-shot `guardd`, content on stdin) — the exit code
distinguishes *classified* from *never classified*, not the verdict's
polarity ([`cmd/guardd/main.go:141`](../cmd/guardd/main.go)):

| Exit | Meaning |
|---|---|
| `0` | any verdict the guard served — **including a normal `block`** (blocking is the guard working, not failing) |
| `2` | `guard_error` — the guard could not judge at all (no classifier API key configured, egress disabled, classifier failure). The error verdict is still emitted as JSON with `errored: true`, and the exit is `2` whichever way `-fail-closed` resolved the decision. |

So guardd's `2` does **not** mean quarantine, and guardd's CLI has no
route-based four-way map — read `route` from the JSON when you need it.
Serving mode (`/check`) has no exit code at all: it answers HTTP, and error
verdicts arrive in-band with HTTP 200 (see Fail-open semantics).

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

## Fail-open semantics

There are two different failure boundaries: the `guardd` HTTP server talking to
its classifier, and a connector talking to `guardd`. Do not confuse a `guardd`
error verdict with a connector transport failure.

**Classifier timeout/error inside guardd:** `POST /check` runs classification
under a **120s request-context deadline**. This is hard-coded in the HTTP
handler (and the one-shot CLI also uses 120s); the HTTP handler has no timeout
flag or environment override. The Jev HTTP client has its own **90s default**
when no custom client is supplied. If classification returns an error (including
a deadline/transport failure), `/check` still returns **HTTP 200** with a JSON
`Result` whose `errored` is true. Its decision/route follows the server policy:
default `-fail-closed=true` yields `decision="block"`,
`route="quarantine"`; `-fail-closed=false` yields `decision="allow"`,
`route="review"`. Both include a `reason` beginning `guard_error (...)`; this
is an in-band verdict, not an HTTP error response. (Sources:
[`cmd/guardd/main.go:128-144`](../cmd/guardd/main.go),
[`cmd/guardd/main.go:236-239`](../cmd/guardd/main.go),
[`guard.go:192-216`](../guard.go),
[`jev.go:161-164`](../jev.go).)

**Connector timeout or unreachable guardd:** when no HTTP response can be
obtained, there is no server status or body to report. The Go and Python clients
apply their own `FailMode`/`fail_mode`: in `closed` mode Go returns an error and
Python raises `GuardError`; in `open` mode each synthesizes a local REVIEW
verdict with `errored=true` and a `guard_error (connector fail-open): ...`
reason. These synthesized verdicts are not an HTTP response. Go withholds all
capabilities; Python preserves its configured outage capabilities
(`allow_tools=true`, `allow_secrets=true`). An `errored` result must never be
auto-delivered. (Sources:
[`client/client.go:80-137`](../client/client.go),
[`connectors/python/guard_client.py:170-182`](python/guard_client.py),
[`connectors/python/guard_client.py:200-230`](python/guard_client.py).)

**HTTP errors from a reachable guardd:** for example, missing/invalid
`X-Operator-Token` produces HTTP **401** with a plain-text
`unauthorized: missing or invalid X-Operator-Token` body. Both connectors treat
non-200 responses as check failures; fail-open mode then returns the same
connector-local REVIEW verdict described above, while fail-closed returns/raises
an error. This does not change guardd's response into a 200. (Sources:
[`cmd/guardd/main.go:184-198`](../cmd/guardd/main.go),
[`client/client.go:109-137`](../client/client.go),
[`connectors/python/guard_client.py:200-230`](python/guard_client.py).)

**Configuration:** guardd's HTTP classification deadline is fixed at 120s.
The Go connector's timeout is configurable through `Client.Timeout` (zero uses
120s); its failure mode is `Client.FailMode`. Python's timeout is configurable
through the `timeout` constructor argument / CLI `--timeout` (default 120s),
and its mode through `fail_mode` / CLI `--fail-open`. The server policy is
configurable with `-fail-closed` (default true), affecting only the in-band
classifier-error verdict, not the HTTP status. (Sources:
[`cmd/guardd/main.go:128-144`](../cmd/guardd/main.go),
[`cmd/guardd/main.go:236-239`](../cmd/guardd/main.go),
[`client/client.go:53-64`](../client/client.go),
[`connectors/python/guard_client.py:170-182`](python/guard_client.py),
[`connectors/python/guard_client.py:264-287`](python/guard_client.py).)

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
   the `Result`. Only `source` and `content` affect the verdict; `channel` is an
   accepted-but-ignored pass-through field (as are `id` and `meta` when supplied),
   so none of those fields is echoed or produces an error. That is the whole
   protocol.
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
