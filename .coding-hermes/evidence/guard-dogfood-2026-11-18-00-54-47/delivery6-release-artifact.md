# Delivery 6 — released-artifact notice, measured against the actual v0.1.0 tarball

Tick `guard-dogfood-2026-11-18-00-54-47`, sixth inbound dispatch (2026-11-17 21:20:23 -0500).
Host: this box, x86_64 Linux. Nothing in the product tree changed since the delivery-4 walk
(`98dba4e`), so no product surface was re-driven. What had never been done: take the ONLY
artifact a stranger can install today and check the five factual claims the HEAD notice
(README.md lines 119-137) makes about it.

## Artifact and integrity

```
GET https://github.com/trouble-agent/guard/releases/download/v0.1.0/guard-linux-amd64.tar.gz
    -> HTTP 200, 2,935,335 bytes
sha256sum guard-linux-amd64.tar.gz
    -> b75cb31a21ade9b2f322642d8906a2e0d5d4b734579c3ced70591cbb01caf4e3
checksums.txt (release asset) says the same -> sha256sum -c: OK (rc=0)
tarball contents: guardd (6,938,786 bytes, 2026-10-02), README.md (6,737 bytes), LICENSE
```

## The five claims, each measured (not read off the docs)

| README notice claim | Probe | Result |
|---|---|---|
| lacks `GUARD_EGRESS_ENABLED` classifier egress kill switch | `GUARD_EGRESS_ENABLED=off ./guardd -content "hello world, benign" -source github_pr` | `decision=allow provider=jev cost_usd=0.000026502` — the provider was still called, the kill switch is inert. **Claim true.** |
| lacks the CLI exit-code contract (`rc=2` fail-closed error, `rc=0` classified) | `./guardd -serve :18814` with no token; then `./guardd -content "summarize this changelog entry"` | `rc=1` on the fail-closed error (not 2); `rc=0` on a classified run. **Claim true.** |
| lacks server token precedence (`GUARD_TOKEN` > `-tokenfile` > `-token`) | `./guardd -h` | flags: `-channel -content -fail-closed -pretty -serve -source -token` only — no `-tokenfile`, no env surface. **Claim true.** |
| lacks `load_bearing_normalizations` attribution | injection and benign inputs, full JSON keys | keys end at `normalizations:["rot13"]`; no `load_bearing_normalizations` on either verdict. **Claim true.** |
| *includes* the default local-only `-serve` binding (the 2026-10-20 correction) | `./guardd -serve :18813 -token ...` then `ss -ltnp`; POST from LAN 192.168.123.147 | `LISTEN 127.0.0.1:18813` only; LAN POST `http=000` (refused). **Correction accurate.** |

Also confirmed: `-version` does not exist in v0.1.0 (`flag provided but not defined`), which is
the released-artifact half of GUARD-DF-023 (pending) — no separate row filed.

## What the released artifact does do

```
injection prose, -source github_pr  -> decision=block risk_level=high route=quarantine
                                       score=0.98 attack_class=instruction_injection
                                       duration_ms=393 cost_usd=0.000029316
benign changelog line               -> decision=allow risk_level=low route=deliver
POST /check with X-Operator-Token   -> HTTP 200, block verdict, same shape
```

## Verdict

5/5 notice claims measured true; core block/allow path works on the released binary; the
tarball is checksum-clean. The only install path a stranger has today is therefore *functional
but 194 product commits behind* — stale by an owner gate (REL-GUARD-007, blocked), not by
neglect or breakage. No product defect found, so no board row was filed from this delivery.

Raw probe transcripts: this file (commands and outputs above are verbatim from the run).
