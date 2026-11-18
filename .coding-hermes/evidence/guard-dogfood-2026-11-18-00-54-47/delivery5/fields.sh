#!/usr/bin/env bash
# Full-field pass over the same 21 real doc chunks: replay + capture normalization/cost/latency attribution.
set -uo pipefail
BIN="$1"; OUT="$2"
: > "$OUT/benign-fields.jsonl"
for f in "$OUT"/chunks/*.txt; do
  bytes=$(wc -c < "$f")
  s=$(date +%s%N)
  resp=$("$BIN" -content "$(cat "$f")" 2>/dev/null)
  rc=$?
  e=$(date +%s%N)
  ms=$(( (e-s)/1000000 ))
  printf '%s\n' "$resp" | jq -c --arg f "$(basename "$f")" --argjson b "$bytes" --argjson ms "$ms" --argjson rc "$rc" \
    '{probe:$f, bytes:$b, wall_ms:$ms, rc:$rc, decision:.decision, route:.route, score:.score, attack_class:(.attack_class//""), risk:.risk_level, bytes_seen:.bytes_seen, classified_bytes:.classified_bytes, inflation:((.classified_bytes/.bytes_seen)|.*100|round/100), normalizations:.normalizations, load_bearing:(.load_bearing_normalizations//[]), duration_ms:.duration_ms, cost_usd:.cost_usd}' >> "$OUT/benign-fields.jsonl"
done
echo "rows=$(wc -l < "$OUT/benign-fields.jsonl")"
echo "=== route counts ==="
jq -r '.route' "$OUT/benign-fields.jsonl" | sort | uniq -c
echo "=== by route: median wall_ms, mean cost, mean inflation ==="
jq -s 'group_by(.route)[] | {route:.[0].route, n:length, wall_ms_median:(map(.wall_ms)|sort|.[length/2|floor]), cost_mean:(map(.cost_usd)|add/length), infl_mean:(map(.inflation)|add/length), lb_rot13:(map(select(.load_bearing|index("rot13")!=null))|length)}' "$OUT/benign-fields.jsonl"
