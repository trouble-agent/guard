# Release readiness re-verification — 2026-10-04

Repository: `trouble-agent/guard`
Verified HEAD: `812f092580931db80aec2837496426bf094d940a`

## v0.1.0 release status

- Local tag `v0.1.0` exists. Its commit is `11b37cfa66fd160c219356b13c7b3aba0de14f7d`.
- The same tag is present on `origin`.
- GitHub Release `v0.1.0` is published (2026-10-02 19:28:24 UTC): https://github.com/trouble-agent/guard/releases/tag/v0.1.0
- Published assets: `guard-linux-amd64.tar.gz`, `guard-linux-arm64.tar.gz`, and `checksums.txt`.

## Delta since v0.1.0

`git log v0.1.0..HEAD --oneline` contains **37 commits** in total, including merge, board, QA-cron, and documentation commits.

Applying the requested material-change rule (commit subjects beginning `feat:` or `fix:`; exclude docs, board, merge, and other commits) yields **6 material commits**:

- `fb1675c` feat: pip-installable Python client via pyproject.toml (GUARD-004)
- `fe4e96d` fix: send X-Operator-Token from Go and Python clients (REVIEW-GUARD-002)
- `b0d5492` fix: resolve guardd token from env/tokenfile instead of argv (REVIEW-GUARD-001)
- `2251470` fix: distinguish applied vs load-bearing normalizations in Result (GUARD-DF-002)
- `342afb8` fix: preserve capabilities on fail-open guard outages (GUARD-DF-001)
- `f801086` fix: guardd CLI exits rc=2 on guard_error errorpath (QA-GUARD-002)

## Proposed next version

**Propose `v0.2.0`.** The delta contains a `feat` (the pip-installable Python client), so under SemVer's backward-compatible feature rule this warrants a MINOR increment from `v0.1.0`; fixes alone would not raise the version beyond that. This is a recommendation, not authorization to cut a release.

## CI on HEAD

**Green.** The newest listed run is `ci` run [37178936396](https://github.com/trouble-agent/guard/actions/runs/37178936396), triggered by push and completed with conclusion `success`. Its `headSha` exactly matches verified HEAD `812f092580931db80aec2837496426bf094d940a`; its `ci` job also concluded `success`.

## Evidence commands

- `git tag -l 'v0.1.0'`
- `git ls-remote --tags origin 'refs/tags/v0.1.0' 'refs/tags/v0.1.0^{}'`
- `gh release list --limit 10`
- `gh release view v0.1.0 --json tagName,name,publishedAt,url,assets`
- `git log v0.1.0..HEAD --oneline`
- `git rev-list --count v0.1.0..HEAD`
- `gh run list --limit 3 --json databaseId,headSha,status,conclusion,name,workflowName,event,createdAt,url`
- `gh run view 37178936396 --json headSha,status,conclusion,jobs`
