#!/usr/bin/env bash
# Classifier determinism: N identical calls, same text, same default policy. Captures score spread.
set -uo pipefail
BIN="$1"; OUT="$2"; N="${3:-8}"
for tag in user-guide-01 README-02; do
  chunk="$OUT/chunks/$tag.txt"
  [ -f "$chunk" ] || { echo "missing $chunk"; continue; }
  : > "$OUT/var-$tag-scores.txt"
  : > "$OUT/var-$tag-routes.txt"
  for i in $(seq 1 "$N"); do
    resp=$("$BIN" -content "$(cat "$chunk")" 2>/dev/null)
    printf '%s\n' "$resp" | jq -r '.score' >> "$OUT/var-$tag-scores.txt"
    printf '%s\n' "$resp" | jq -r '.route' >> "$OUT/var-$tag-routes.txt"
    printf '%s\n' "$resp" | jq -c --arg tag "$tag" '{tag:$tag, score:.score, route:.route, decision:.decision, duration_ms:.duration_ms, cost_usd:.cost_usd, classified_bytes:.classified_bytes, load_bearing:(.load_bearing_normalizations//[])}' >> "$OUT/var-detail.jsonl"
  done
  echo "$tag n=$N scores: $(paste -sd' ' "$OUT/var-$tag-scores.txt") routes: $(sort "$OUT/var-$tag-routes.txt" | uniq -c | paste -sd' ')"
  echo "$tag spread: $(sort -n "$OUT/var-$tag-scores.txt" | awk 'NR==1{min=$1} {max=$1} END{printf "min=%.2f max=%.2f delta=%.2f\n", min, max, max-min}')"
done
