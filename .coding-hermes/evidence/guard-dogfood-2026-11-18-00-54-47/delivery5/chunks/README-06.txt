`connectors/python/` (`guard-client`): stdlib-only, speaks the HTTP shape
above, same verdict contract as the Go client.

The HTTP shape is authenticated. A shared secret that **every** `/check`
request must present in the `X-Operator-Token` header is resolved by
precedence **`GUARD_TOKEN` env > `-tokenfile` (must be 0600; group/world-
readable files are refused) > `-token`**; a missing or wrong token is a 401
and the body is never classified. `guardd` refuses to start serving without a
token rather than run unauthenticated, and `GET /healthz` stays open (no
header) so probes keep working — the conventional `GET /health` alias is
served by the same handler and returns the identical body, so generic
orchestrators that probe `/health` pass liveness discovery too. Prefer the
env or tokenfile forms: `-token`
puts the secret in the process list (`ps`, `/proc/<pid>/cmdline`), so serving
with it prints a warning to stderr. `-serve` takes the bind address: an
omitted host defaults to loopback, so `:8768` binds `127.0.0.1:8768` — pass
`0.0.0.0:8768` (or an explicit host) to bind every interface deliberately. The
`listening on <addr>` line is logged only once the socket is bound, so a failed
bind (a taken port, say) prints just `serve: listen tcp <addr>: bind: address
already in use` and exits 1 — the startup banner is never a false green for a
server that never listened. The CLI shapes (`stdin`, `-content`) are unaffected.

### Policy files (`-policy` / `GUARD_POLICY`)

The default decision table (`guard.DefaultPolicy()`: block ≥ 0.85, review ≥
0.50) is the same for every source. To tighten one source — e.g. webhook
deliveries, where the threshold calibration says a per-source Override is the
right tool — name a policy file:

```bash
GUARD_POLICY=/etc/guard/policy.json guardd -serve :8768  # env
guardd -serve :8768 -policy /etc/guard/policy.json       # flag (wins over the env var)
```

The file is JSON matching the `guard.Policy` struct:

```json
{
  "id": "tight-webhook",
  "block_at": 0.85,
  "review_at": 0.50,
  "quoted_downgrade": 0.80,
  "fail_closed": true,
  "overrides": {
    "webhook":   {"block_at": 0.55, "review_at": 0.40},
    "github_pr": {"block_at": 0.75}
  }
}
```

- `id` — policy name recorded on every verdict (`policy` field); an empty id
  keeps `"default"` so verdicts stay attributable.
- `block_at` / `review_at` — the global score bands (classifier scores in
  (0, 1]).
- `quoted_downgrade` — the Quoted confidence at which a quarantine is
  downgraded to review. Leave it out and the shipped default (0.80) applies:
  a literal `0` is indistinguishable from "omitted" in JSON, and a 0 would
  otherwise downgrade EVERY quarantine (the comparison is `Quoted >= 0`,
  always true). To run without the downgrade, set it to `1.0`.
- `fail_closed` — the error stance; `-fail-closed=false` still overrides it
  on the command line.
- `overrides` — per-source tables keyed by the `source` the caller supplies
  (`-source` on the CLI, `source` in the `/check` body). Each entry may set
  `block_at`, `review_at`, `fail_closed`; unset fields inherit the policy's
  own values. An unknown source simply gets the global bands. The verdict names
  the entry that decided it — `override_source` carries the key and
  `effective_block_at` / `effective_review_at` the bands it used (see "Which
  row of the table decided" above), so a misfiring Override is visible in the
  audit trail instead of only in this file.

Loading is strict: a named file that is missing, is not valid
`guard.Policy` JSON, carries a band outside (0, 1], or pairs
`review_at > block_at` (on the policy or on any override) is a startup
failure — guardd refuses to run with a looser table than the operator meant
to install. The resolved table is announced on stderr
(`guardd: policy loaded from -policy (id "…", N override(s))`). The
`-policy` flag and `GUARD_POLICY` apply to the CLI shape too, so a shell
pipeline can run under the same table the service does.
