# DOC-11 — Guard Documentation Parity Audit (re-run)

- **Auditor:** DOC-11 worker (docs re-audit lane)
- **Date:** 2026-10-14
- **HEAD:** 160afea (branch main; == origin/main at audit start, verified `git status -sb`)
- **Workdir:** /home/kara/trouble-agent/guard
- **Workflow:** documentation-auditing skill (derive-surface-first, execute docs, re-triage, re-verify prior rows by execution)
- **Scope note:** audit artifact excluded from its own scan inputs. .gitreins/history/ summaries are machine-written verdict homes, not reader docs. dagger.db* are sqlite sidecar files, not repo content.

## 1. Shipped surface derived from SOURCE (not docs)

Commands:

```
$ cat Makefile | grep -E '^[a-zA-Z_-]+:'
build:
test:
vet:
lint:
release:
docs-check: build
print-go-directive:

$ cat go.mod
module github.com/trouble-agent/guard
go 1.26.6

$ ls cmd/guardd connectors/python skills/guard-usage
cmd/guardd: main.go, main_test.go
connectors/python: guard_client.py, pyproject.toml (project.scripts: guard = "guard_client:main"), test_guard_client.py
skills/: guard-usage/SKILL.md only (no other agent-facing skill)
```

CLI flags (source `cmd/guardd/main.go:97-105` flags block; verified against live `-h` output in §3):

| Flag | Source anchor |
|---|---|
| -serve | main.go:98 |
| -token | main.go:99 |
| -tokenfile | main.go:100 |
| -content, -source, -channel, -fail-closed, -pretty | main.go:100-105 |

HTTP surface (source `newHandler`, main.go:172-204): `GET|* /healthz` (main.go:174-179, no auth, `{"status":"ok","egress_enabled":<bool>}`), `POST /check` (main.go:181-204: auth→method→body order, 401/405/400/400, 8 MiB LimitReader main.go:190, 120s ctx main.go:201).

Library surface: `guard.Guard/Check/Input/Result` (guard.go:69-78+), `client.Client{Endpoint,Token,Timeout,FailMode}` (client/client.go:33-51), `New(endpoint)` defaults 120s + FailClosed (client/client.go:63-65). Python connector `connectors/python/guard_client.py` with console script `guard` (pyproject.toml:16-17).

Makefile targets: build, test, vet, lint, release, docs-check (replays README fences), print-go-directive. CI: .github/workflows/ci.yml (build/vet/test/gitleaks); release.yml (tag push → goreleaser). .goreleaser.yaml: binary `guardd`, archives `guard-{{.Os}}-{{.Arch}}.tar.gz`, `checksums.txt`.

## 2. Reader-doc corpus read in full

README.md (14,650 B, 317 L), docs/ADOPTION.md (9,744 B), docs/threshold-calibration.md (8,717 B), docs/release-readiness-2026-10-04.md (2,471 B), docs/dogfood/* (3 files, dated findings), connectors/README.md (11,408 B, 213 L), skills/guard-usage/SKILL.md.

## 3. Executable parity checks (every documented command spot-run)

### 3.1 `make docs-check` — README command replay (T1/T2)

```
$ make docs-check
docs-check: go directive = 1.26.6
docs-check: checking README at README.md
docs-check: full go directive 1.26.6 found in README
docs-check: replaying README build: go build ./cmd/guardd          # produces ./guardd
docs-check replaying README command: printf '%s' "hello, world" | ./guardd          # CLI: classify via stdin
docs-check: fail-closed JSON matches README
docs-check: empty-stdin fail-closed rc=2 OK
docs-check: PASS
(exit 0)
```

(The third line of the replay echo above dropped the colon in transcription; target output was `docs-check: replaying README command: ...`. No content change.)

### 3.2 CLI probes (env -i, no key → deterministic fail-closed)

```
$ env -i HOME=/tmp/doc11-probe PATH=$PATH ./bin/guardd -h
Usage of ./bin/guardd:
  -channel string
  -content string
  -fail-closed   (default true)
  -pretty
  -serve string
  -source string
  -token string
  -tokenfile string
(exit 0; 8 flags = flags block main.go:97-105)

$ printf 'hello' | env -i HOME=/tmp/doc11-probe PATH=$PATH GUARD_EGRESS_ENABLED=true ./bin/guardd
{"decision":"block","risk_level":"medium","route":"quarantine","reason":"guard_error (fail-closed): no openrouter api key available","policy":"default","provider":"jev","score":0,"errored":true,"constraints":{"allow_tools":false,"allow_network":false,"allow_secrets":false},"normalizations":["rot13"]}
rc=2   ← exactly the README's fail-closed JSON + documented exit code
```

### 3.3 Live HTTP probes (server: GUARD_TOKEN=doc11tok, GUARD_EGRESS_ENABLED=false, -serve 127.0.0.1:8791)

```
GET /healthz (no header)          → 200 {"status":"ok","egress_enabled":false}
POST /check, no token             → 401 unauthorized: missing or invalid X-Operator-Token
POST /check + X-Operator-Token    → 200 in-band error verdict:
  {"decision":"block",...,"reason":"guard_error (fail-closed): hosted classifier egress is disabled (GUARD_EGRESS_ENABLED=false); no local classifier is configured","provider":"disabled","errored":true,...}
POST /check, content:""           → 400 content is required
GET /check                        → 405 POST only
```

All five match connectors/README.md's status table (401→main.go:183, 405→main.go:187, 400→main.go:192/196) and the documented 8 MiB/120s budgets.

### 3.4 Remaining README fences

```
$ go build ./... ; go vet ./...            → OK (audit start)
$ go build -ldflags="-s -w" -o /tmp/doc11-probe/guardd-stripped ./cmd/guardd
→ STRIPPED_BUILD_OK, ELF 64-bit x86-64
$ go test ./... -count=1
ok  github.com/trouble-agent/guard      4.029s
ok  github.com/trouble-agent/guard/client   0.005s
ok  github.com/trouble-agent/guard/cmd/guardd   0.009s
$ curl -sI https://github.com/trouble-agent/guard/releases/latest/download/checksums.txt
→ HTTP/2 302 → .../releases/download/v0.1.0/checksums.txt  (release exists; v0.1.0 live)
```

## 4. DOC-N re-triage (re-verified by execution, not by grep alone)

| Row | Fix | Verified present at HEAD | Evidence |
|---|---|---|---|
| DOC-2 | /healthz contract + main.go anchors | connectors/README.md:64-68 `{status, egress_enabled}` + anchors 172-178 | anchor check: `newHandler` main.go:172, healthz handler 174-179 — matches; live /healthz probe §3.3 returns documented body |
| DOC-6 | CLI exit codes | README.md:55-62 (rc 2 on guard_error, 0 classified, serve always in-band); connectors/README.md:100-101 fail-closed=true→block/quarantine, =false→allow/review | §3.2 rc=2 live; §3.3 in-band 200 live; source contract main.go:41-46, guard.go:153-167 |
| DOC-7 | Go 1.26.6 + Python fail-open | skills/guard-usage/SKILL.md:14,82 Go ≥1.26.6; ADOPTION.md:86-107 `decision='error'` + CONSTRAINTS_FAIL_OPEN; connectors/README.md:163-180 divergence table | `make print-go-directive` = 1.26.6 == go.mod == README:12 |
| DOC-8 | GUARD_EGRESS_ENABLED | README.md:203-220 (values false/0/no/off, default enabled, egress-disabled warning text verbatim); ADOPTION.md:161-205 (full operator section); connectors/README.md:67-68 | §3.3 egress-disabled probe returns the documented reason string + `provider:"disabled"` verbatim |
| DOC-10 | fail-open capability semantics | connectors/README.md:87-180 (three-boundary semantics + divergence table Python preserved / Go+CLI withheld); ADOPTION.md §2 constraints branching | table matches client.go:51-61 (FailOpen withheld) and guard_client.py:170-182/200-230 anchors |

**All five merged fixes verified present and accurate at HEAD. No regression found.**

## 5. New findings (gap N — new, gap O — discoverability)

### DOC-12 (P4) — Makefile targets (incl. `make docs-check`) absent from all reader docs

- Exact discovery: `grep -rn 'make\b\|docs-check\|Makefile' README.md docs/ADOPTION.md docs/threshold-calibration.md connectors/README.md skills/guard-usage/SKILL.md` → **zero hits** (rc=1).
- Source: Makefile targets build/test/vet/lint/release/docs-check/print-go-directive; `make docs-check` runs green at HEAD (§3.1) and is the repo's own README-drift gate (landed via README-6).
- Why it matters: guardd ships seven make targets — the drift gate itself is undocumented. A contributor reading only the README builds with `go build` and never discovers `make docs-check`; CI runs raw go commands, not make.
- Not on board: `grep -i 'makefile\|make docs\|make build\|docs-check' tasks.jsonl` matches only README-6 (the row that CREATED the target; its fix shipped, closed done) — no row documents the targets for readers.
- Acceptance: at least one reader doc (README Build/Testing section) lists the make targets with one line each (build, test, vet, lint, release, docs-check), and `make docs-check` is named as the README-drift gate with its go-directive behavior.

### DOC-13 (P5) — `-channel` CLI flag absent from CLI examples/help docs

- Exact discovery: `grep -rn 'channel' README.md skills/guard-usage/SKILL.md connectors/README.md` → prose "channel" only (README:78 "delivery-channel problem"; ADOPTION JSON body `"channel": "router"`); no `guardd ... -channel` example anywhere.
- Source: `-channel` is one of the 8 CLI flags (main.go:102), echoed onto the result's Input; `guard.Input.Channel` JSON tag `channel` (guard.go:72); HTTP body examples DO show channel, CLI examples never do.
- Why it matters: CLI integrators keying on docs never learn the flag exists — asymmetric with -source (documented at README:170, SKILL:29).
- Not on board: no prior row names the `-channel` flag (board `"channel"` hits are the HTTP body in DOC-2 only).
- Acceptance: one CLI example per reader doc (README "Three consumption shapes", skills/guard-usage/SKILL.md) shows `-source github_pr -channel crier` style usage, or the flag is documented as optional/ignored by policy.

### DOC-14 (P5) — connectors/README.md discoverable from nothing

- Exact discovery: `grep -rn 'connectors/README\|Connectors' README.md docs/ADOPTION.md docs/threshold-calibration.md skills/guard-usage/SKILL.md` → zero hits (rc=1).
- Why it matters: the 213-line connectors reference (the integrator-facing contract) is linked from no README/docs/skill page; readers reach it only by browsing the tree.
- Not on board: no open row names connectors/README discoverability (README-4 is docs/ links only — verified closed with README linking docs/).
- Acceptance: README "Further docs" section links connectors/README.md.

## 6. Audience coverage

| Audience | Home | Status |
|---|---|---|
| Integrator (Go in-process) | README "The surface" + §Three shapes | documented |
| Integrator (HTTP, any language) | connectors/README.md HTTP contract + status table | documented; **discoverability gap (DOC-14)** |
| Operator | README auth/egress sections + ADOPTION.md §Operator configuration | documented |
| Agent (guard-usage skill) | skills/guard-usage/SKILL.md | documented |
| Contributor (build/test/release) | README Build/Testing + Install | partially documented — **Makefile targets missing (DOC-12)** |
| CLI integrator | README §Three shapes + SKILL.md | partially documented — **-channel flag missing (DOC-13)** |

## 7. Unverified scope

- Real Jev classification (live OpenRouter key run): not exercised — requires a live key; no-key fail-closed and egress-disabled paths probed instead. The agent-facing skill and README both document key pickup order ($OPENROUTER_API_KEY → ~/.hermes/.env → ~/9router-deploy/.env.shared) — present in docs, not re-executed with a real key.
- Python connector E2E: `guard` console script + guard_client.py anchors spot-checked (source read, pyproject entry point verified) but no live server round-trip driven from Python in this audit.
- Release artifacts themselves: latest/download resolves to v0.1.0 assets (HTTP 302 verified); tarball contents not downloaded/extracted.
- goreleaser `make release` (snapshot) not run — requires network + goreleaser binary; config vs README asset names verified textually.
- Threshold-calibration numbers: reproduced the documented policy values from source (policy.go BlockAt 0.85/ReviewAt 0.50/QuotedDowngrade 0.80 match the doc); corpus precision/recall figures not re-measured.

## 8. Verdict

- 5/5 merged DOC fixes (DOC-2/6/7/8/10) verified present and accurate at HEAD 160afea — no reopen.
- 3 new actionable gaps: DOC-12 (P4), DOC-13 (P5), DOC-14 (P5). Filed to .coding-hermes/board/tasks.jsonl this commit.
- DOC-11 closed done; DOC-9 superseded by this audit.
