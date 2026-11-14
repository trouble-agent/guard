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

ch:trace row=REL-GUARD-006 spec=CHANGELOG.md#[0.2.0] wave=none:serial-worker test=make-docs-check doc=CHANGELOG.md prompt=none:ephemeral-brief evidence=none:worker-acceptance-output witness=none:ephemeral-acceptance verdict=none:foreman-judge commit=none:foreman-commit memory=none:not-written

[Unreleased]: https://github.com/trouble-agent/guard/compare/v0.1.0...HEAD
[0.2.0]: https://github.com/trouble-agent/guard/compare/v0.1.0...v0.2.0
