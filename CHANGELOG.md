# Changelog

All notable changes to guard are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions are tagged
in git and cut with goreleaser.

## [Unreleased]

## [0.2.0] - 2026-11-04

First release since v0.1.0 (2026-10-02): 3 features, 19 fixes, no breaking
changes.

### Added

- Policy files: `-policy <file>` / `GUARD_POLICY` load a JSON decision table
  (matching guard.Policy) so per-source overrides are installable without
  writing Go. A missing or invalid file is a startup failure, never a silent
  fall-back to the default bands (GUARD-DF-005).
- Conventional `/health` alias served from the healthz handler, so generic
  orchestrators that probe `/health` pass liveness discovery
  (REVIEW-GUARD-005).
- pip-installable Python client via `connectors/python/pyproject.toml`
  (GUARD-004).
- `guardd -version` prints the stamped build version. Release builds inject it
  via ldflags (Makefile `-X main.version=$(VERSION)`, goreleaser
  `-X main.version={{.Version}}`); a plain `go build` reports `dev`.

### Changed

- Python distribution renamed `guard-client` → `guard-agent-client` (import
  name unchanged): the PyPI name `guard-client` belongs to an unrelated
  project, so the old install instruction fetched the wrong package.
  docs/ADOPTION.md now points at the repo path.
- Python connector fail-open default removed: a truncated body or proxy error
  page no longer parses as route=deliver. A verdict with no valid route now
  routes to review with decision=error.
- guardd CLI exits rc=2 on guard_error (no key configured, egress disabled,
  classifier failure) so exit-status-keying callers can distinguish
  "classified, blocked" from "never classified" (QA-GUARD-002).
- Reported billing now includes the attribution classifier call's cost and
  latency (GUARD-DF-006).
- Building the Go release requires Go >=1.26.11, the minimum toolchain that
  closes the reachable standard-library advisories GO-2026-6859 (`net/http`)
  and GO-2026-6809 (`crypto/x509`) (REL-GUARD-008).
- `make docs-check` requires `jq`; when `jq` is absent, it emits the install
  instruction "Install jq and re-run (see README Prerequisites, GUARD-DF-004)."
- Normalization cost on oversized input was cut by bounding the scan of
  multi-megabyte payloads (8 MiB normalized at 145 MB/s; an 8 MiB
  egress-disabled `/check` at ~57.8 ms per iteration) (GUARD-DF-020).
- Normalizing no longer carries a scan window: the 128 KiB prefix plus 16 KiB
  suffix bound was unreachable — content over the 20,000-byte classification
  cap is rejected before any normalizer runs — so it is removed and the docs
  now describe the shipped behavior: normalization covers all accepted
  content, and anything above the cap is rejected explicitly
  (GUARD-DF-020-R2).

### Fixed

- guardd logs its readiness banner only after the listen succeeds — a failed
  bind no longer looks like a healthy start (GUARD-DF-014).
- Per-source overrides: the applied override is reported on the verdict
  (GUARD-DF-011); override-level fail-closed is honored on the guard-error
  path (GUARD-DF-010); override keys matching no Input.Source are validated
  with a warning (GUARD-DF-007).
- Token handling: shared secret resolved by precedence env > tokenfile > argv
  with constant-time comparison, argv tokens warned loudly (REVIEW-GUARD-001);
  Go and Python clients send X-Operator-Token (REVIEW-GUARD-002).
- Guard-error semantics: capabilities preserved on fail-open outages
  (GUARD-DF-001); applied vs load-bearing normalizations distinguished in
  Result (GUARD-DF-002); classifier egress made explicit via
  GUARD_EGRESS_ENABLED (GUARD-007).
- Calibration replay marks errored runs INVALID instead of printing bare
  metrics; live Jev transport failures are skipped under network cuts
  (QA-GUARD-005-R2-R2).
- Documentation aligned with the published product (2026-11-03): adoption
  paths, install instructions, and released-artifact notices corrected,
  including the v0.1.0 Go client's missing auth header as an upgrade floor.
  Fresh-agent install and live consumer workflows verified SHIPPABLE by the
  dogfood lane (run 6, 2026-10-29).
- golangci-lint is enforced through make lint and CI, with errcheck and
  ineffassign fixes making the gate green (QA-GUARD-006; QA-GUARD-001-R2).
- Long input whose normalized form exceeds the 20,000-byte classification
  budget is no longer scored on a truncated prefix: over-budget normalized
  state now returns an explicit classification-window guard_error naming the
  byte count, and `bytes_seen` reports the complete normalized-state length, so
  expanded content past the old scan window is classified or rejected loudly
  instead of delivered at a low risk score (GUARD-DF-018).

ch:trace row=REL-GUARD-006 spec=CHANGELOG.md#[0.2.0] wave=none:serial-worker test=make-docs-check doc=CHANGELOG.md prompt=none:ephemeral-brief evidence=none:worker-acceptance-output witness=none:ephemeral-acceptance verdict=none:foreman-judge commit=none:foreman-commit memory=none:not-written
ch:trace row=REL-GUARD-011 spec=CHANGELOG.md#0.2.0 wave=.coding-hermes/waves/guard-foreman-2026-11-15-16-29-26.json#REL-GUARD-011 test=make docs-check doc=CHANGELOG.md evidence=none:worker-branch witness=none:worker-branch verdict=none:pending-tier2 commit=none:worker-commit memory=none:not-written

[Unreleased]: https://github.com/trouble-agent/guard/compare/v0.1.0...HEAD
[0.2.0]: https://github.com/trouble-agent/guard/compare/v0.1.0...v0.2.0
