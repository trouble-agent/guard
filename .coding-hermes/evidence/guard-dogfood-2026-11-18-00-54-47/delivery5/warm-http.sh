#!/usr/bin/env bash
# Warm HTTP path: long-lived guardd -serve, keep-alive connections, real doc chunks.
set -uo pipefail
OUT="$1"; PORT="$2"; N="${3:-6}"
TOK=$(cat "$OUT/tok.txt")
URL="http://127.0.0.1:$PORT"
curl -s -o /dev/null -w 'healthz=%{http_code} %{time_total}s\n' "$URL/healthz"
: > "$OUT/http-warm.jsonl"
for tag in tiny user-guide-01 README-02; do
  if [ "$tag" = tiny ]; then body="hello, world"; else body=$(cat "$OUT/chunks/$tag.txt"); fi
  jq -n --arg c "$body" --arg s "channel_message" '{content:$c, source:$s, channel:"df5-warm"}' > "$OUT/payload-$tag.json"
  bytes=$(wc -c < "$OUT/payload-$tag.json")
  for i in $(seq 1 "$N"); do
    read -r code total < <(curl -s -o "$OUT/resp.json" -w '%{http_code} %{time_total}\n' \
      -H "X-Operator-Token: $TOK" -H 'Content-Type: application/json' \
      --data-binary "@$OUT/payload-$tag.json" "$URL/check")
    ms=$(awk -v t="$total" 'BEGIN{printf "%d", t*1000}')
    jq -c --arg tag "$tag" --argjson ms "$ms" --arg code "$code" --argjson pb "$bytes" \
      '{tag:$tag, http:$code, payload_bytes:$pb, wall_ms:$ms, decision:.decision, route:.route, score:.score, bytes_seen:.bytes_seen, classified_bytes:.classified_bytes, duration_ms:.duration_ms, cost_usd:.cost_usd, load_bearing:(.load_bearing_normalizations//[])}' "$OUT/resp.json" >> "$OUT/http-warm.jsonl"
  done
done
echo "=== warm /check medians ==="
jq -s 'group_by(.tag)[] | {tag:.[0].tag, n:length, wall_ms_median:(map(.wall_ms)|sort|.[length/2|floor]), wall_ms_min:(map(.wall_ms)|min), wall_ms_max:(map(.wall_ms)|max), route:.[0].route, score_mean:((map(.score)|add)/length*1000|round/1000), classified_bytes:.[0].classified_bytes, bytes_seen:.[0].bytes_seen, cost_mean:((map(.cost_usd)|add)/length)}' "$OUT/http-warm.jsonl"
