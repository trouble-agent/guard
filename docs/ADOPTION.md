# Adoption — where each consumer hooks the guard

The guard is only useful if it sits at a **choke point** where untrusted content
crosses into a context that will act on it. One hook per consumer; the
classification itself is the shared core. This file names the exact seam in each
of the first three consumers.

The pattern is identical everywhere, which is why one connector shape suffices:

```
untrusted input ──▶ guard.check() ──▶ route == deliver ?
                                        ├─ yes ─▶ proceed as normal
                                        └─ no  ─▶ honour route + constraints
                                                  (quarantine = do not deliver;
                                                   review = handle with no tools/secrets)
```

---

## 1. crier (Go — agent-to-agent message bus)

**Choke point:** `internal/registry/handler.go` → `Handler.HandleDeliver`
(`POST /agents/{id}/inbox`). crier already has a guard here
(`internal/guard`, spec `specs/LLM-MESSAGE-GUARD.md`), so this is **not** a new
call site — it is a provider swap.

crier's guard classifies by asking a **chat LLM** to *emit JSON*, then parsing it
(`ParseVerdict`). The shared core classifies with **Jev**, which returns typed
answers, so the parse step and its failure mode disappear.

**Two integration levels:**

- *Minimal* — add Jev as a provider class behind the existing `ProviderSpec`
  seam (`internal/guard/router.go`, presets in spec §5.2). The verdict contract
  is already compatible (`allow|block` — `sanitize` is reserved for contract
  parity with crier but no path in this core produces it yet; risk is
  `low|medium|high`), so nothing downstream changes.
- *Full* — have crier's guard delegate classification to the shared core
  (in-process import, or the `client` package against a shared `guardd`) and keep
  crier's delivery integration (per-channel policy, sanitize/rewrite, quarantine
  payload) on top. This is the option that makes "one core" true.

**Fail mode:** crier's spec default is **fail-open** (availability first) with a
deterministic pre-scan that still blocks high-confidence hits. Match that:
`FailMode: FailOpen` on the connector, and keep the pre-scan blocking.

---

## 2. task-router (Python — deterministic model router)

**Choke point:** the runtime entry `scripts/router_spawn.py` and the router
server (`:9092`). Every task carries a description that ends up in a model's
context; that description is the untrusted input.

**Connector:** `connectors/python/guard_client.py`
(pip installable: `pip install guard-client` from this repo's `connectors/python/`).

```python
import os

from guard_client import Guard

guard = Guard(fail_mode="open", token=os.environ["GUARD_TOKEN"])   # a guard outage must not stall the scheduler

def route_task(task):
    v = guard.check(task.description, source="task-router", channel="router")
    if v.errored:
        # guard outage under fail-open: capabilities are PRESERVED (flagged),
        # route to REVIEW. Never auto-deliver an errored verdict.
        task.context = {"guard": {"route": v.route, "error": v.reason}}
        return task
    if not v.deliver:
        # do not let an injected description reach a model with tools/secrets
        task.context = {"guard": {"route": v.route, "class": v.attack_class}}
        task.tools = [] if not v.allows_tools else task.tools
        task.secrets = [] if not v.allows_secrets else task.secrets
    return task
```

**Fail mode:** **open** — task-router's own doctrine is "fail-open is sacred;
never block the scheduler." A guard outage must therefore degrade to REVIEW, not
to a stall.

### Constraints branching

`constraints` is a **capability map, not a verdict**. Branch on `v.errored`
first, in this order:

- **`v.errored` is True** — the guard was unreachable and `fail_mode='open'`
  applied. The verdict *grants* the usual capabilities
  (`Guard.CONSTRAINTS_FAIL_OPEN` = `{"allow_tools": True, "allow_secrets":
  True}`) precisely so an outage does not silently cripple every task, and sets
  `route='review'` / `decision='error'`. **Keep `task.tools` and
  `task.secrets` intact and take the review path — never auto-deliver an
  errored verdict, regardless of what `constraints` says.** The strip pattern
  below is for real policy verdicts only.
- **`v.errored` is False** — a real verdict. Honour `route` and `constraints`:
  route `deliver` means proceed; anything else means review/quarantine, and
  clear (`[]`) each capability the verdict does not grant.

Why: before this rule an outage returned `constraints={}` — indistinguishable
from a real block-all policy — so the documented strip pattern emptied
`task.tools` and `task.secrets` on every task for the whole outage, with no
signal a consumer was likely to check (`deliver` was the documented binding
key, and it only says *not to deliver*, not *why*). An errored verdict now
carries its own `decision='error'` and always routes to review. Fail-open means
"delivery continues flagged"; it must not mean "fail-crippled".

---

## 3. hermes-dagger (Go — sandbox-first DAG engine)

**Choke points** (content crossing into a context that acts):

- **Webhook ingress** — `cmd/dagger` webhook endpoints. A webhook body is
  attacker-controlled and becomes a DAG input.
- **`fetch()` bridge** — `src/bridge/` (sandbox `fetch()` → Hermes chat API).
  Data pulled from the network flows into LLM context.
- **`$()` selectors** — HTML/JSON/Markdown/CSV/TOML/DuckDB parsed from
  untrusted documents; the selected content can carry instructions.

**Connector:** `github.com/trouble-agent/guard/client` (HTTP to a shared
`guardd`), or the core package in-process.

```go
c := client.New(guardEndpoint)
c.Token = os.Getenv("GUARD_TOKEN") // /check requires X-Operator-Token
res, _ := c.Check(ctx, guard.Input{Source: "webhook", Channel: "dagger", Content: body})
if res.Route != guard.RouteDeliver {
    return daggerErr(res.Reason)   // do not build a DAG from this input
}
```

**Note:** dagger is already a sandbox-first engine (QJS + bwrap), which contains
*execution* risk. The guard covers the complementary risk — **instruction**
content entering an LLM's context — which a sandbox does not address. They stack;
neither replaces the other.

**Fail mode:** **closed** for webhook/control ingress (a missed injection could
drive DAG construction), **open** for read-only `$()` enrichment.

---

## Adding the next consumer

1. Find the **choke point**: where untrusted bytes cross into a context that
   acts (an LLM prompt, a code path, a tool invocation).
2. Call `guard.check(content, source=..., channel=...)` there.
3. Honour `route` and `constraints` — do not "look and proceed".
4. Pick the **fail mode** deliberately and record it.

If the consumer needs a lane the contract lacks, extend the core — the
connectors stay one-liners, which is what keeps the surface powerful.

For agent-facing integration details the consumer docs don't repeat
(prerequisites, key load order, `GUARD_ENDPOINT`, the `errored` verdict
contract), see [skills/guard-usage/SKILL.md](../skills/guard-usage/SKILL.md).
