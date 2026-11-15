# GUARD-DF-019 — Python connector CLI carries `source` / `channel` (2026-11-15)

Row: `GUARD-DF-019` (filed by dogfood run 8,
`docs/dogfood/2026-11-14-run8-integration.md:92`, tabled at `:205`).
Tree: worktree `/home/kara/worktrees/guard-GUARD-DF-019`, branch `wt/GUARD-DF-019`,
base `bf4aa5d90377a7ee37179de01f87622322a52bc0`.

## Defect (as filed, re-verified here)

`guard --help` (the shipped console script, `[project.scripts] guard =
"guard_client:main"`) exposed only `--endpoint`, `--token`, `--timeout`,
`--fail-open`, `[content]`; `main()` called `guard.check(content)`. The library
underneath already accepted `source`/`channel` and sent both in the `/check`
body, so the CLI was the only surface that could not select a per-source
`Policy.Overrides` row — a deployment relying on source-scoped bands silently
enforced the base thresholds for every shell caller.

Pre-fix help surface, verbatim (`python3 connectors/python/guard_client.py --help`,
base commit):

```
usage: guard [-h] [--endpoint URL] [--token SECRET] [--timeout SECONDS]
             [--fail-open]
             [content]
```

Pre-fix `--source` was an argparse usage error (exit 2), i.e. indistinguishable
from a quarantine verdict's documented exit code.

## Change

- `connectors/python/guard_client.py` — `_parser()` gains `--source LABEL` and
  `--channel LABEL` (both default `""`), `main()` calls
  `guard.check(content, source=args.source, channel=args.channel)`, and the
  module/`check()` docstrings state what each label does.
- `connectors/python/test_guard_client.py` — `_SourceScopedStub`, a stub that
  behaves like `policy.go`'s `selectFor`: one fixed classifier score, the
  request's `source` choosing the bands (base 0.85/0.50, tightened `webhook`
  row 0.55/0.40, `>=` comparisons), echoing `override_source` /
  `effective_*`. Five new tests (see `test=` below).
- `connectors/README.md` — `source` / `channel` documented beside endpoint and
  token (heading now "Operational knobs (endpoint, token, timeout,
  source/channel)"), plus one sentence at "The contract".

Existing behaviour pinned, not changed: endpoint/token/timeout resolution,
`--fail-open` semantics (`CONSTRAINTS_FAIL_OPEN`), fail-closed `GuardError` →
exit 3, stdin content path, exit-code map 0/1/2/3, `guard =
"guard_client:main"` entrypoint. Omitting the flags sends `""`, exactly the
old wire body.

## Evidence

**RED (pre-fix `guard_client.py`, HEAD content, new tests staged):**

```
$ python3 connectors/python/guard_client.py --help | grep -c -- '--source'
0
$ cd connectors/python && python3 -m pytest -q test_guard_client.py
FAILED test_guard_client.py::test_cli_source_selects_per_source_policy_result
FAILED test_guard_client.py::test_cli_channel_passes_through_without_changing_the_verdict
FAILED test_guard_client.py::test_cli_help_advertises_source_and_channel
3 failed, 28 passed in 3.71s
```

The two tests that pass pre-fix are the deliberate preserved-behaviour pins
(`test_cli_without_source_sends_empty_source`,
`test_check_sends_source_and_channel_in_request_body`) — they must not fail
pre-fix, because nothing about them changed.

**GREEN (fixed tree):**

```
$ cd connectors/python && python3 -m pytest -q test_guard_client.py
31 passed in 4.21s
```

**Live CLI (fixed tree), flags parse and an outage still exits 3, not 2:**

```
$ python3 connectors/python/guard_client.py --help | sed -n '1,3p'
usage: guard [-h] [--endpoint URL] [--token SECRET] [--timeout SECONDS]
             [--source LABEL] [--channel LABEL] [--fail-open]
             [content]

$ python3 connectors/python/guard_client.py --endpoint http://127.0.0.1:1 \
    --timeout 1 --source webhook --channel crier 'hello'
guard: guard check failed: <urlopen error [Errno 111] Connection refused>
rc=3
```

**Go side (untouched, re-run anyway):**

```
$ go build ./... && go vet ./... && go test ./... -count=1
ok  github.com/trouble-agent/guard          3.581s
ok  github.com/trouble-agent/guard/client   0.005s
ok  github.com/trouble-agent/guard/cmd/guardd  0.017s
ok  github.com/trouble-agent/guard/data/corpus 3.321s
$ make lint
0 issues.
$ make docs-check
... docs-check: PASS   # includes the main.go:131-132 anchor this change cites
```

Note: `make docs-check` validates the ROOT `README.md` and the
`connectors/README.md` → `cmd/guardd/main.go` line anchors. It does not replay
the Python CLI, so the `--source`/`--channel` surface is pinned by the pytest
suite above, not by the Makefile target.

## Not done (honest scope)

- No push (worker branch; the foreman merges and publishes).
- No board or `.gitreins` edit, per row scope.
- `skills/guard-usage/SKILL.md` documents the Go CLI and the Python library, not
  the console script's flags; it is accurate as written and was left alone.
- The harness has no installed `guard` console script on this host, so the
  entry-point wiring was proven by `main()` (the exact symbol
  `[project.scripts]` points at), not by an installed shim. `pip install` was
  not exercised.
