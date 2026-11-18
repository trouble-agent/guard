### Classification window and accounting

The Jev/OpenRouter provider documents a normal text path around 32,000 bytes
and a separate long-context mode. Guard does not rely on the provider silently
choosing or truncating that mode. Its existing normalization budget is the
stricter, explicit boundary: an input over **20,000 UTF-8 bytes** is rejected
before the provider is called. Normalization that would exceed the same budget
is rejected too. This is deliberate fail-closed behavior, not a verdict on a
prefix, so a canary at the head and one at the tail receive the same
`errored=true`, `truncated=true` outcome.

Every verdict exposes `bytes_seen` (raw input bytes received, or the complete
normalized-state length when normalization exceeds the budget),
`classified_bytes` (bytes sent to the provider across classifier calls), and
`truncated`. A rejected long input has `bytes_seen` set, `classified_bytes: 0`,
and `truncated: true`; callers must route it as an error and must not deliver it.
The limit applies to the Go core and therefore to CLI, HTTP, Go-client, and
Python-connector responses.

### Error verdicts

On any classifier failure the verdict is a DIFFERENT shape: `errored=true`,
`risk_level=medium`, `decision=block|allow`, `route=quarantine|review` (block +
quarantine when the policy fails closed, allow + review when it fails open), and
`reason="guard_error (fail-closed|fail-open): <err>"`. The `errored` field uses
`json:"errored,omitempty"`: it is serialized only when true, so an absent key
means false. Clients must treat absence as false rather than indexing the key
unconditionally. An error verdict carries
NO `attack_class` and NO `model`; `score` is present but always `0` on an error
verdict and is not a measurement. It DOES carry the resolved row:
`effective_block_at` / `effective_review_at` always, plus `override_source` when
a per-source Override was selected — which is what tells you whether that
source's fail-closed (or fail-open) stance came from the policy or from its
Override entry. Integrators: check `errored`
before trusting any other field of a block verdict — an outage looks like a
block, but it is a different shape, and a consumer that parses the success
shape unconditionally will mis-handle it.

The Go core always produces the shapes above. The Python connector's own
fail-open outage verdict additionally uses `decision="error"` (not `allow`) so
consumers can tell an outage from a real block-all policy, and pairs it with
`route=review` plus fail-open constraints — honour those constraints rather
than treating the verdict as a block.

### Attack classes (stable identifiers, shared with crier)
`instruction_injection` · `jailbreak` · `masquerade` · `structured_object` · `none`

## Three consumption shapes

```bash
# 1. CLI — pipe anything through it
printf '%s' "$message" | guardd
guardd -content "$message" -source github_pr -pretty
guardd -content "$message" -source github_pr -channel crier -pretty   # -channel: pass-through label for caller bookkeeping; ignored by the core (default: empty)

# 2. HTTP service — one instance, every language
GUARD_TOKEN="$GUARD_TOKEN" guardd -serve :8768           # env (recommended: token never in argv)
guardd -serve :8768 -tokenfile /run/secrets/guard.token  # or a 0600 token file
guardd -serve :8768 -token "$GUARD_TOKEN"                # dev: works, but warns (token visible in ps)
GUARD_POLICY=/etc/guard/policy.json guardd -serve :8768  # policy file: per-source overrides, custom bands
guardd -serve :8768 -policy /etc/guard/policy.json       # same, via flag (wins over GUARD_POLICY)
curl -s localhost:8768/check \
  -H "X-Operator-Token: $GUARD_TOKEN" \
  -d '{"source":"inbox","channel":"crier","content":"..."}'

# 3. Go library
g := &guard.Guard{Classifier: guard.NewJev(), Policy: guard.DefaultPolicy()}
res := g.Check(ctx, guard.Input{Source: "github_pr", Content: raw})
if res.Route != guard.RouteDeliver { /* honour res.Constraints */ }
```

For Python callers, a pip-installable connector package lives in
