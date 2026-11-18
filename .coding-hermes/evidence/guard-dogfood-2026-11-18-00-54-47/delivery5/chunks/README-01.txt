# guard

A lightweight, language-neutral **message-security filter**. One core, many
callers: crier (Go, in-process), task-router (Python, over HTTP), shell scripts
(piped through the CLI). The contract is deliberately the same shape crier's
`internal/guard` already speaks, so adopting it is not a second dialect.

## Quick Start

**Prerequisites**

- **Go ≥ 1.26.11** to build (the repo itself declares Go 1.26.11 in `go.mod`).
- **`jq` is required by `make docs-check`** (the README-drift gate shells out
  to it to compare verdict JSON). Nothing else in the repo needs it — build
  and `go test` run on Go alone. The target probes for jq first and, when it
  is missing, stops with `docs-check FAIL: jq was not found on PATH — this
  target shells out to jq (four times) to compare verdict JSON. Install jq
  and re-run (see README Prerequisites, GUARD-DF-004).` GitHub Actions
  runners ship jq preinstalled, so CI stays green where a fresh box fails.
- **`make release` additionally needs `goreleaser`; no other target does.**
- **An OpenRouter API key is required for ANY classification.** Classification
  runs on the Jev decisions model via OpenRouter; with no key available, every
  verdict comes back as the fail-closed block shown below. A key is picked up
  from, in order, `$OPENROUTER_API_KEY`, then `~/.hermes/.env`, then
  `~/9router-deploy/.env.shared` (key-shaped `sk-or-v1-…` values are scanned
  from each file, deduplicated, and tried in that order — a 401/402/429 moves
  to the next key). There is no key flag — the CLI and HTTP shapes both
  resolve keys through this same order (`loadOpenRouterKeys`, `jev.go`).
  To reproduce the no-key example on a machine whose normal HOME contains a
  key, use an isolated HOME for the command, for example `TD=$(mktemp -d) &&
  env -u OPENROUTER_API_KEY HOME=$TD bash -c 'printf "hello world" | ./guardd'`.

**Build and run**

```bash
go build -o guardd ./cmd/guardd          # produces ./guardd
printf '%s' "hello, world" | ./guardd          # CLI: classify via stdin
GUARD_TOKEN=s3cr3t ./guardd -serve :8768       # or serve over HTTP
cat > policy.json <<'JSON'                       # create a per-source policy fixture
{
  "id": "quick-start",
  "block_at": 0.85,
  "review_at": 0.50,
  "fail_closed": true,
  "overrides": {
    "webhook": {"block_at": 0.55, "review_at": 0.40}
  }
}
JSON
GUARD_TOKEN=s3cr3t GUARD_POLICY=policy.json ./guardd -serve :8768 # serve with per-source overrides
```

**First run with no key — this is the expected, correct (fail-closed) result**

(On a box with a live key you will instead see a successful classification; to
reproduce the verdict below, use the scratch-HOME note that follows this block.)

```json
{
  "decision": "block",
  "risk_level": "medium",
  "route": "quarantine",
  "reason": "guard_error (fail-closed): no openrouter api key available",
  "policy": "default",
  "effective_block_at": 0.85,
  "effective_review_at": 0.5,
  "provider": "jev",
  "score": 0,
  "errored": true,
  "bytes_seen": 12,
  "classified_bytes": 29,
  "truncated": false,
  "constraints": {
    "allow_tools": false,
    "allow_network": false,
    "allow_secrets": false
  },
  "normalizations": [
    "rot13"
  ]
}
```

> **To reproduce this on a box with a live key:** the example's JSON block is the
> verdict you would see, so the reproduction runs the same one-shot stdin call on
> a scratch HOME, stripping `$OPENROUTER_API_KEY` from the environment so the key
> discovery order (env → `~/.hermes/.env` → `~/9router-deploy/.env.shared`) finds
> nothing:
> ```bash
> TD=$(mktemp -d) && env -u OPENROUTER_API_KEY HOME=$TD bash -c 'printf "hello world" | ./guardd'
> ```

This is fail-closed working as designed: a guard that cannot judge must not
wave content through, so the message is blocked and quarantined and the
`reason` names the error. The CLI (guardd run without `-serve`) exits with
code 2 in this state and 0 on a successfully classified run, so shell and CI
callers keying on rc can distinguish "classified" from "config broken"
