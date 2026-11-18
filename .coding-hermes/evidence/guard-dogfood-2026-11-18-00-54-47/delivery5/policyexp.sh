#!/usr/bin/env bash
# Same benign content, two policies: deliver vs forced-review. Isolates the cost of the
# attribution re-classification (guard.go attribution block) from content/provider effects.
set -uo pipefail
BIN="$1"; OUT="$2"; N="${3:-3}"
chunk="$OUT/chunks/user-guide-01.txt"
[ -f "$chunk" ] || chunk=$(ls "$OUT"/chunks/*-01.txt | head -1)
echo "chunk=$chunk bytes=$(wc -c < "$chunk")"
printf '{"id":"df5-reviewed-source","block_at":0.99,"review_at":0.10,"fail_closed":true}\n' > "$OUT/policy-tight.json"
printf '{"id":"df5-default-bands","block_at":0.85,"review_at":0.50,"fail_closed":true}\n' > "$OUT/policy-default.json"
for pol in default tight; do
  : > "$OUT/pol-$pol-times.txt"
  for i in $(seq 1 "$N"); do
    s=$(date +%s%N)
    resp=$("$BIN" -content "$(cat "$chunk")" -policy "$OUT/policy-$pol.json" 2>/dev/null)
    e=$(date +%s%N)
    ms=$(( (e-s)/1000000 ))
    echo "$ms" >> "$OUT/pol-$pol-times.txt"
    printf '%s\n' "$resp" | jq -c --arg pol "$pol" --argjson ms "$ms" \
      '{policy:$pol, wall_ms:$ms, decision:.decision, route:.route, score:.score, effective_block_at:.effective_block_at, effective_review_at:.effective_review_at, classified_bytes:.classified_bytes, bytes_seen:.bytes_seen, load_bearing:(.load_bearing_normalizations//[]), duration_ms:.duration_ms, cost_usd:.cost_usd, override_source:(.override_source//"")}' >> "$OUT/pol-detail.jsonl"
  done
  med=$(sort -n "$OUT/pol-$pol-times.txt" | awk '{a[NR]=$1} END{ if(NR%2) print a[(NR+1)/2]; else print (a[NR/2]+a[NR/2+1])/2 }')
  echo "policy=$pol n=$N median_ms=$med"
done
echo "=== detail ==="
cat "$OUT/pol-detail.jsonl"
