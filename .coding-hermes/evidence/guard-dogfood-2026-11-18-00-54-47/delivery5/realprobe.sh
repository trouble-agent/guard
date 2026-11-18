#!/usr/bin/env bash
# Real-content probe for guard (dogfood tick guard-dogfood-2026-11-18-00-54-47, delivery 5).
# Surface: guardd -content "<text>" — the same CLI shape data/corpus/run_calibration.sh uses.
# Inputs: REAL user-facing prose from the repo, chunked to realistic agent-message sizes.
set -uo pipefail
REPO=/home/kara/trouble-agent/guard
GUARDD="$1"
OUT="$2"
CHUNK="$3"
mkdir -p "$OUT/chunks"
for doc in README.md docs/user-guide.md docs/operations.md CHANGELOG.md skills/guard-usage/SKILL.md docs/threshold-calibration.md; do
  base=$(basename "$doc" .md)
  awk -v out="$OUT/chunks/${base}-" -v lim="$CHUNK" '
    { buf = buf $0 "\n"; if (length(buf) >= lim) { n++; f = sprintf("%s%02d.txt", out, n); printf "%s", buf > f; close(f); buf = "" } }
    END { if (length(buf) > 0) { n++; f = sprintf("%s%02d.txt", out, n); printf "%s", buf > f; close(f) } }
  ' "$REPO/$doc"
done
: > "$OUT/benign-real-prose.jsonl"
for f in "$OUT"/chunks/*.txt; do
  bytes=$(wc -c < "$f")
  start=$(date +%s%N)
  resp=$("$GUARDD" -content "$(cat "$f")" 2>/dev/null)
  rc=$?
  end=$(date +%s%N)
  ms=$(( (end - start) / 1000000 ))
  printf '%s\n' "$resp" | jq -c --arg f "$(basename "$f")" --argjson b "$bytes" --argjson ms "$ms" --argjson rc "$rc" \
    '{probe:$f, bytes:$b, wall_ms:$ms, rc:$rc, decision:.decision, route:.route, score:.score, reason:.reason, duration_ms:.duration_ms, cost_usd:.cost_usd, effective_block_at:.effective_block_at, effective_review_at:.effective_review_at, errored:(.errored // false)}' >> "$OUT/benign-real-prose.jsonl"
done
echo "rows=$(wc -l < "$OUT/benign-real-prose.jsonl")"
jq -r '[.decision,.route,(.score|tostring)]|@tsv' "$OUT/benign-real-prose.jsonl" | sort | uniq -c
