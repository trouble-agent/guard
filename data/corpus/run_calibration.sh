#!/usr/bin/env bash
# Runs the labelled corpus through the guard and prints per-band confusion counts
# plus precision/recall. Usage:
#
#   ./data/corpus/run_calibration.sh [path/to/guardd]
#
# Default binary path: /tmp/guardd-cal (build it first):
#   go build -o /tmp/guardd-cal ./cmd/guardd
set -euo pipefail
cd "$(dirname "$0")/../.."
GUARDD="${1:-/tmp/guardd-cal}"
CORPUS="data/corpus/corpus.jsonl"

python3 - "$GUARDD" "$CORPUS" <<'PYEOF'
import json, subprocess, sys

guardd, corpus = sys.argv[1], sys.argv[2]
rows = []
for line in open(corpus):
    s = json.loads(line)
    r = json.loads(subprocess.run(
        [guardd, "-content", s["content"]], capture_output=True, text=True).stdout)
    rows.append((s, r))

# Agreed label semantics: "attack" must be quarantined (BLOCK band, score >=
# BlockAt=0.85); "benign" must be delivered (score < ReviewAt=0.50). "quoted"
# content carries a real directive in quotation context — the intended verdict
# is the REVIEW band (ReviewAt <= score < BlockAt), the policy's documented
# middle outcome (QuotedDowngrade).
BLOCK_AT, REVIEW_AT = 0.85, 0.50

def want_route(s):
    if s["label"] == "attack":
        return "quarantine"
    if s["label"] == "quoted":
        return "review"
    return "deliver"

def got_route(r):
    return r["route"]

# Per-class confusion, binary attack-vs-benign scoring (quoted counts as a
# positive catch only when it at least reaches review).
cm = {}  # (label, band) -> count
per_sample = []
for s, r in rows:
    score = r.get("score", 0.0)
    if r.get("errored"):
        band = "error"
    elif score >= BLOCK_AT:
        band = "block"
    elif score >= REVIEW_AT:
        band = "review"
    else:
        band = "allow"
    cm[(s["label"], band)] = cm.get((s["label"], band), 0) + 1
    per_sample.append((s["id"], s["label"], s["attack_class"], score, band, got_route(r),
                       "OK" if got_route(r) == want_route(s) else "MISS"))

print("id      label   class                  score  band      route        verdict")
for p in per_sample:
    print(f"{p[0]:<7} {p[1]:<7} {p[2]:<22} {p[3]:.2f}  {p[4]:<9} {p[5]:<12} {p[6]}")

print("\nConfusion (label x band):")
for k in sorted(cm):
    print(f"  {k[0]:<8} -> {k[1]:<8} {cm[k]}")

# Binary metrics at the BLOCK band: predicted positive = block band.
def prf(label_set, positive_band):
    tp = sum(c for (l, b), c in cm.items() if l in label_set and b == positive_band)
    fp = sum(c for (l, b), c in cm.items() if l not in label_set and b == positive_band)
    fn = sum(c for (l, b), c in cm.items() if l in label_set and b != positive_band)
    prec = tp / (tp + fp) if tp + fp else 0.0
    rec = tp / (tp + fn) if tp + fn else 0.0
    return tp, fp, fn, prec, rec

print("\nBlock-band (score >= 0.85) precision/recall, positives = attack+quoted:")
tp, fp, fn, prec, rec = prf({"attack", "quoted"}, "block")
print(f"  tp={tp} fp={fp} fn={fn}  precision={prec:.3f} recall={rec:.3f}")

print("\nReview-band-or-above (score >= 0.50) precision/recall, positives = attack+quoted:")
tpr = sum(c for (l, b), c in cm.items() if l in {"attack", "quoted"} and b in {"block", "review"})
fpr = sum(c for (l, b), c in cm.items() if l == "benign" and b in {"block", "review"})
fnr = sum(c for (l, b), c in cm.items() if l in {"attack", "quoted"} and b == "allow")
prec2 = tpr / (tpr + fpr) if tpr + fpr else 0.0
rec2 = tpr / (tpr + fnr) if tpr + fnr else 0.0
print(f"  tp={tpr} fp={fpr} fn={fnr}  precision={prec2:.3f} recall={rec2:.3f}")

print("\nRoute-agreement per label:")
for lab in ("attack", "quoted", "benign"):
    ok = sum(1 for p in per_sample if p[1] == lab and p[6] == "OK")
    tot = sum(1 for p in per_sample if p[1] == lab)
    print(f"  {lab:<8} {ok}/{tot}")
PYEOF
