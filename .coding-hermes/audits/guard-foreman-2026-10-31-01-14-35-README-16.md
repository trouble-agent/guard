# README-16 verification evidence

Merged tree: 2e36d6477a330fc673228582906625cedf95c42d
Published ref: origin/main (verified parity after push)

Exact acceptance evidence:

- `make build` succeeded and produced `bin/guardd`.
- In a clean `/tmp/readme-test` directory, the README policy fixture was written before the documented command.
- `GUARD_TOKEN=test GUARD_POLICY=policy.json ./guardd -serve :18970` started successfully and logged: `guardd: policy loaded from GUARD_POLICY (id "quick-start", 1 override(s))`.
- The prior missing-file behavior remains reproducible without the fixture: `guardd: policy file: open policy.json: no such file or directory`, exit 1.
- `go test ./... -count=1` passed for the root, client, cmd/guardd, and data/corpus packages.
- GitReins verdict: `5b4c7c81`, PASS, bound to the implementation commit and verified after merge.

ch:trace row=README-16 spec=README.md#quick-start wave=.coding-hermes/waves/guard-foreman-2026-10-31-01-14-35.json test=documented-command-probe doc=README.md evidence=.coding-hermes/audits/guard-foreman-2026-10-31-01-14-35-README-16.md witness=live:guardd-policy-startup+origin/main verdict=5b4c7c81 commit=2e36d6477a330fc673228582906625cedf95c42d memory=none:namespace-not-resolved

STATUS: PASS
TIER: T3
EVIDENCE: documented policy command starts with the inline fixture; full Go suite passes; GitReins PASS.
