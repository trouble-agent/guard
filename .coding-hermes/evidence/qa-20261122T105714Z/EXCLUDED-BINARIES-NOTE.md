# Evidence exclusion note — QA-GUARD-009 witness battery

Two files from logs/qa-logs-3172927/ were deliberately NOT committed to the repo:
they are compiled Go binaries of the app under test (identical builds, chaos-cell
artifacts), not evidence, and would add 22MB to the repo (committed evidence
convention: text logs only, largest prior file 25KB).

- logs/qa-logs-3172927/corr-torn/app
  sha256 baedd1a7b327eca5730203393fd59c050962fe1c6d1964ef5aa4196e2ca58c77
- logs/qa-logs-3172927/corr-mal/app
  sha256 baedd1a7b327eca5730203393fd59c050962fe1c6d1964ef5aa4196e2ca58c77
  (identical binary, two chaos-cell copies)

Both are `go build` outputs of the tested tree and reproducible from the cell
logs' build steps. Removed from disk after hashing so the checkout stays clean
for the next tick. The complete untrimmed battery evidence set (including
per-cell logs) lives at /home/kara/.hermes/qa/guard-QA-GUARD-009-20261122/ per
PROVENANCE.md there.

— foreman tick guard-foreman-2026-11-22-10-22-47
