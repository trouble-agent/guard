(serving mode always exits 0 — the block verdict is the legitimate answer to
a request). Set `OPENROUTER_API_KEY` (or populate one of the two
`.env` files above) and real classification starts. To invert this on purpose,
run with `-fail-closed=false` (an error then routes to `review` flagged instead
of blocking).

```
message in ──▶ normalise ──▶ classify ──▶ policy ──▶ route out
                (decode)      (Jev)       (code)      (deliver|review|quarantine
                                                       + capability constraints)
```

## Install (released binaries)

Tagged releases ship static Linux binaries built by
`.github/workflows/release.yml` (GoReleaser). See GitHub Releases for tagged
versions: https://github.com/trouble-agent/guard/releases (first release:
[v0.1.0](https://github.com/trouble-agent/guard/releases/tag/v0.1.0),
2026-10-02; assets: `guard-linux-amd64.tar.gz`, `guard-linux-arm64.tar.gz`,
`checksums.txt`).

> **Released-artifact notice (2026-10-17, updated 2026-10-20).** The only
> tagged release, **v0.1.0** (2026-10-02), predates four user-facing contracts
> added since it was cut, and the README bundled in its tarball (153 lines)
> documents that older binary, not this one. The released binary lacks the
> `GUARD_EGRESS_ENABLED` classifier egress kill switch, the CLI exit-code
> contract (`rc=2` on a fail-closed error, `rc=0` on a classified run), server
> token precedence (`GUARD_TOKEN` env > `-tokenfile` > `-token`), and the
> normalization-attribution semantics represented by
> `load_bearing_normalizations` (what actually drove the verdict,
> GUARD-DF-002). It does include the default local-only `-serve` binding and
> its channel-selection CLI option; those are not gaps in the released
> artifact. The released binary also predates the Go `net/http` security fix
> GO-2026-6617 (SEC-GO-1269). The tarball's bundled README is a snapshot frozen
> at tag time and cannot be corrected in place — cutting a release is an owner
> decision — so the standing warning lives in the v0.1.0 release notes and in
> this notice until the next tag. **Build from source (Quick Start above)** to
> get the contracts documented here — the notice is removed by the next tagged
> release (v0.2.0 is proposed in
> [release readiness](docs/release-readiness-2026-10-04.md)).

```bash
curl -sL -o guard-linux-amd64.tar.gz \
  https://github.com/trouble-agent/guard/releases/latest/download/guard-linux-amd64.tar.gz
curl -sL -O \
  https://github.com/trouble-agent/guard/releases/latest/download/checksums.txt
sha256sum --ignore-missing -c checksums.txt
tar xzf guard-linux-amd64.tar.gz
./guardd -pretty -content "hello, world"   # verify the binary runs
```

Each tarball contains `guardd`, the README, and the LICENSE. **A released
binary is NOT interchangeable with a source build** — the notice above lists
the contracts it lacks — so prefer building from source (Quick Start above)
for the current contract set, and for other platforms. Running a released
binary when you accept the older contract set: pipe a message through
`guardd`, or serve it with `guardd -serve` (token and bind-address setup in
**Quick Start** above).

Agent-facing usage notes (prerequisites, key load order, the `errored`
verdict contract): see [skills/guard-usage/SKILL.md](skills/guard-usage/SKILL.md).

## Testing

```bash
make test    # the whole suite: go test ./...
```

No external services are required to run it: the suite is plain `go test`
(use `go test ./... -count=1` to bypass the test cache). The unit and policy
tests use a deterministic stub classifier and pass offline. The live end-to-
end test (`TestLiveJevRoutesKnownCases`, `jev_live_test.go`) — the one that
exercises the real Jev classifier — skips itself when no resolved OpenRouter
key is available, under `-short`, or when `HTTP_PROXY` or `HTTPS_PROXY` is
non-empty (the latter is treated as a network cut). Therefore, `make test`
does not prove that the live route test ran. To prove it ran, use
