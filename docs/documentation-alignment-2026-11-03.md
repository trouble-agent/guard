# Documentation alignment — 2026-11-03

**Purpose:** reconcile what this repo's docs say with what `origin/main` actually
does, after a month of doc surfaces frozen at the `v0.1.0` cut while the code
moved on.

**Verified HEAD:** `ce2a86c` (`origin/main`, 2026-11-03)
**Last code change:** 2026-10-28
**Released artifact:** `v0.1.0`, published 2026-10-02T19:28:24Z

If you are reading a doc dated **before 2026-11-03**, treat its "current state"
claims as a historical snapshot, not as the product.

---

## 1. The structural cause

Every drift below has one root: **the release has not been cut.** `v0.2.0` was
first proposed on 2026-10-04 (`docs/release-readiness-2026-10-04.md`), re-raised
on 2026-10-17, again on 2026-10-20, and again in the 2026-11-01 sweep. It is
still not cut, so `trouble-agent/guard@latest` — what every consumer resolves —
is a month behind the documented product.

The docs are not wrong about intent; they describe a product that exists in the
repo and has never been published. This document marks that boundary explicitly
rather than rewriting the docs to describe an unpublished state as shipped.

## 2. Corrections to earlier "current" claims

These appeared in dated docs and are **no longer true** as of `ce2a86c`:

| Claim | Where it appeared | Reality at `ce2a86c` |
|---|---|---|
| "`/health` is absent while `/healthz` works" | `REVIEW-GUARD-005` / releng runs 2026-10-26 | **Both are served.** Route table registers `/health` and `/healthz`. See §3. |
| "Go client sends `X-Operator-Token`" | implied by the `fb1675c`… era docs | True in **source**, **false in the published `v0.1.0`** — see §4. |
| "pip install guard-client" installs this connector | `docs/ADOPTION.md` before 2026-11-03 | **False.** That PyPI name belongs to an unrelated project (`elhio/guard-python`). Fixed in this change. |

## 3. Route surface — verified

Routes served by `guardd` at `ce2a86c` (source: route registration in
`guard.go`, cross-checked against the live service during the 2026-11-03 review):

- `/check` — classification; requires `X-Operator-Token`
- `/health` — served
- `/healthz` — served

Earlier reviewer reports that `/health` was missing were accurate for the commit
they tested and are **stale now**. Do not carry that finding forward.

## 4. The published Go client cannot authenticate — the one that matters

`github.com/trouble-agent/guard@v0.1.0` ships a `client` package with **no
`Token` field and no `X-Operator-Token` header**. `guardd` rejects `/check`
without that header, so **every Go consumer on the published version gets 401
and cannot use the product at all**.

Verified by downloading `v0.1.0` from the module proxy:

- `grep -c Operator-Token` in `client@v0.1.0/client.go` → **0**
- the only header it sets is `Content-Type`

The fix (`fe4e96d fix: send X-Operator-Token from Go and Python clients`)
landed **after** the tag, so it exists in source and in no published artifact.

**Consequence for docs:** any Go adoption snippet is correct only against a
version containing `Client.Token`. `docs/ADOPTION.md` now states the `v0.1.1`
floor and says plainly that `v0.1.0` must be upgraded.

## 5. Python connector — two defects fixed in this change

1. **Distribution-name collision.** The `pyproject.toml` named the package
   `guard-client`, which is an existing unrelated PyPI project. Renamed to
   `guard-agent-client` (import name stays `guard_client`).
2. **Fail-open on a malformed verdict.** `Verdict.from_dict` defaulted a missing
   `route` key to `ROUTE_DELIVER`, so a truncated body, a proxy error page, or a
   future server schema was read as permission to deliver untrusted input. It
   now returns a `review` verdict with `decision="error"`. Three regression
   tests pin the behaviour, and they fail against the old default.

## 6. Environment variables — verified

| Variable | Read by | Status |
|---|---|---|
| `GUARD_TOKEN` | `guardd`, Go client, Python client | documented, correct |
| `GUARD_POLICY` | `guardd` | documented, correct |
| `GUARD_EGRESS_ENABLED` | `guardd` | documented, correct |
| `GUARD_ENDPOINT` | **Python client only** | documented; note it is not a `guardd` variable |
| `GUARD_CAL_LIVE` | **tests only** (skips live calibration) | documented correctly as a test switch |

No documented variable is missing from the code. `GUARD_ENDPOINT` and
`GUARD_CAL_LIVE` are not `guardd` configuration — several docs are ambiguous on
this; the table above is the authority.

## 7. What is deliberately NOT changed

- **`docs/release-readiness-2026-10-04.md`** and the dated `docs/dogfood/*` runs
  are **historical records**. They are left intact and are accurate *about the
  date they were written*. Rewriting them would destroy the audit trail that
  shows how long the release has been outstanding.
- **The README's released-artifact notice** stays until `v0.2.0` is actually
  cut. It is currently correct.

## 8. What unblocks the rest

Cutting **`v0.2.0`** resolves the whole class of drift at once:

- `go get …@latest` starts resolving a client that can authenticate
- the README released-artifact notice can be removed
- the tarball's bundled README stops being a month-old snapshot
- every "current state" doc becomes true again

That is an owner/release-lane decision, tracked as `REL-GUARD-003` and
`REL-GUARD-004`. This document is the evidence that the gap is now a decision
rather than an unknown.

## 9. Evidence commands

```bash
# route surface
grep -rn 'HandleFunc\|Handle(' --include='*.go' . | grep -v _test

# what consumers actually resolve
curl -s https://proxy.golang.org/github.com/trouble-agent/guard/@v/list

# prove the v0.1.0 client cannot authenticate
go mod download github.com/trouble-agent/guard@v0.1.0
grep -c 'Operator-Token' "$(go env GOMODCACHE)/github.com/trouble-agent/guard@v0.1.0/client/client.go"   # -> 0

# the PyPI collision
curl -s https://pypi.org/pypi/guard-client/json | jq -r '.info.author'   # -> Elhio (not us)

# connector behaviour
cd connectors/python && python -m pytest test_guard_client.py -q
```
