#!/usr/bin/env bash
# guard dogfood perf arm — delivery 4 (duplicate fire), cheap re-timing of the
# headline /check operation at an unchanged tree. No repo writes.
set -u
BIN=/home/kara/trouble-agent/guard/bin/guardd
S=/home/kara/.hermes/cache/scratch/guardd-perf4
mkdir -p "$S"
printf '%s' 'dogfood-local-token' > "$S/token"; chmod 600 "$S/token"

# 4 KiB contention payload (plain prose-like bytes; normalizes 1:1)
head -c 4096 /dev/urandom | base64 | head -c 4096 > "$S/body4k.txt"
BODY=$(cat "$S/body4k.txt")
PAYLOAD=$(printf '{"source":"inbox","channel":"crier","content":"%s"}' "$BODY" | tr -d '\n')
printf '%s' "$PAYLOAD" > "$S/payload.json"
echo "payload_bytes=$(wc -c < "$S/payload.json")"

arm () { # $1 = label, $2 = port, $3 = egress(false|true)
  local label=$1 port=$2 egress=$3 pid t0 t1
  t0=$(date +%s%N)
  GUARD_TOKEN_FILE_PLACEHOLDER=1 GUARD_EGRESS_ENABLED=$egress \
    "$BIN" -serve 127.0.0.1:$port -tokenfile "$S/token" > "$S/$label.server.log" 2>&1 &
  pid=$!
  # cold: poll until first response lands
  local cold=""
  for _ in $(seq 1 100); do
    out=$(curl -s -o "$S/$label.cold.out" -w '%{http_code}' -m 5 \
      -H "X-Operator-Token: $(cat "$S/token")" -H 'Content-Type: application/json' \
      --data-binary @"$S/payload.json" "http://127.0.0.1:$port/check" 2>/dev/null)
    if [ "$out" = "200" ]; then t1=$(date +%s%N); cold=$(( (t1-t0)/1000000 )); break; fi
    sleep 0.05
  done
  echo "$label cold_ms=${cold:-TIMEOUT} http=$out classified=$(jq -r '.classified_bytes // "n/a"' "$S/$label.cold.out" 2>/dev/null) route=$(jq -r '.route // "n/a"' "$S/$label.cold.out" 2>/dev/null)"
  # warm: 9 timed calls
  : > "$S/$label.times"
  for i in $(seq 1 9); do
    curl -s -o "$S/$label.warm.$i.out" -w '%{time_total}\n' -m 10 \
      -H "X-Operator-Token: $(cat "$S/token")" -H 'Content-Type: application/json' \
      --data-binary @"$S/payload.json" "http://127.0.0.1:$port/check" >> "$S/$label.times" 2>/dev/null
  done
  sort -n "$S/$label.times" > "$S/$label.times.sorted"
  echo "$label n=$(wc -l < "$S/$label.times") min=$(head -1 "$S/$label.times.sorted") median=$(sed -n '5p' "$S/$label.times.sorted") max=$(tail -1 "$S/$label.times.sorted")"
  echo "$label last_classified=$(jq -r '.classified_bytes // "n/a"' "$S/$label.warm.9.out" 2>/dev/null) last_route=$(jq -r '.route // "n/a"' "$S/$label.warm.9.out" 2>/dev/null) last_reason=$(jq -r '.reason // ""' "$S/$label.warm.9.out" 2>/dev/null | cut -c1-90)"
  kill $pid 2>/dev/null
  wait $pid 2>/dev/null
}

arm egressoff 18771 false
arm egresson  18772 true
echo "--- server stderr (egresson) ---"
head -5 "$S/egresson.server.log"
