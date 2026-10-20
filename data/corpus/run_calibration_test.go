package main

// Regression tests for GUARD-DF-012: data/corpus/run_calibration.sh used to
// print confident-looking precision/recall (and a perfect-looking
// route-agreement line) for a run in which the guard classified nothing.
// With every row errored (e.g. no OPENROUTER_API_KEY, egress disabled) the
// script still exited 0, the two metric arms contradicted each other (37 FN
// at the block band, 0 FN at the review band) because errored rows fell into
// neither, and "Route-agreement: attack 32/32" read as SUCCESS for a dead
// classifier (fail-closed quarantine *is* the expected attack route).
//
// Both tests drive the real script end-to-end through a stub `guardd` (a
// tiny Python script emitting fixed Result JSON — the same `guardd -content`
// CLI surface the script drives), so the bash+python pipeline is exercised
// for real with zero network and zero classifier calls. corpus.jsonl is the
// labelled input either way; only the guard's answers are stubbed.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// erroredStub mimics guardd on a host where no classifier is reachable:
// fail-closed error verdict, quarantine route, exit code handled by the
// script (guardd itself exits 2; the script keys on Result.Errored).
const erroredStub = `#!/usr/bin/env python3
import json
print(json.dumps({
    "decision": "block", "risk_level": "critical", "route": "quarantine",
    "score": 0.0, "errored": True,
    "reason": "guard_error (fail-closed): no OPENROUTER_API_KEY configured",
}))
`

// cleanStub emits a deterministic verdict per content length so a run over
// the real corpus lands in all three bands: score in [0.05, 0.94] cycling
// with len(content)%90 — allow (<0.50), review ([0.50,0.85)), block (>=0.85).
const cleanStub = `#!/usr/bin/env python3
import json, sys
argv = sys.argv[1:]
content = argv[argv.index("-content") + 1]
score = ((len(content) % 90) + 5) / 100.0
if score >= 0.85:
    route, decision, risk = "quarantine", "block", "critical"
elif score >= 0.50:
    route, decision, risk = "review", "review", "elevated"
else:
    route, decision, risk = "deliver", "allow", "low"
print(json.dumps({"decision": decision, "risk_level": risk, "route": route, "score": score}))
`

// corpusRowCount counts the labelled rows the script will drive.
func corpusRowCount(t *testing.T) int {
	t.Helper()
	b, err := os.ReadFile("corpus.jsonl")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	n := 0
	for _, line := range strings.Split(string(b), "\n") {
		if s := strings.TrimSpace(line); s != "" && !strings.HasPrefix(s, "//") {
			n++
		}
	}
	return n
}

// runCalibration runs the real script against a stub guardd and returns the
// combined output plus the exit code. The script cds to the repo root via
// its own path, so the test's cwd (the package dir, data/corpus) is fine.
func runCalibration(t *testing.T, stub string) (string, int) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "guardd-stub")
	if err := os.WriteFile(bin, []byte(stub), 0o755); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	cmd := exec.Command("bash", "run_calibration.sh", bin)
	out, err := cmd.CombinedOutput()
	rc := 0
	if exit, ok := err.(*exec.ExitError); ok {
		rc = exit.ExitCode()
	} else if err != nil {
		t.Fatalf("run script: %v", err)
	}
	return string(out), rc
}

// TestCalibrationAllErroredRunIsInvalid is the acceptance test: a run where
// the guard classified nothing must print the RUN INVALID banner and NOT a
// bare (self-contradicting) metrics table.
func TestCalibrationAllErroredRunIsInvalid(t *testing.T) {
	out, rc := runCalibration(t, erroredStub)

	if rc != 2 {
		t.Errorf("exit code = %d, want 2 (\"could not judge\" parity with the guardd CLI)", rc)
	}
	n := corpusRowCount(t)
	if !strings.Contains(out, "RUN INVALID") {
		t.Errorf("output missing RUN INVALID banner:\n%s", out)
	}
	if !strings.Contains(out, fmt.Sprintf("%d/%d rows errored", n, n)) {
		t.Errorf("banner should count errored rows out of the corpus total (%d/%d):\n%s", n, n, out)
	}
	if strings.Contains(out, "precision=") {
		t.Errorf("precision/recall must be suppressed on an all-errored run:\n%s", out)
	}
	// The raw evidence stays: the confusion table still shows the error
	// bucket so the reader can see the run produced nothing but errors.
	if !strings.Contains(out, "-> error") {
		t.Errorf("confusion table should still show the error bucket as raw evidence:\n%s", out)
	}
}

// TestCalibrationCleanRunKeepsMetrics pins the valid path: a zero-errored
// run keeps rc 0, the full metrics, no banner, no error bucket — and the two
// metric arms must agree (same fn), the consistency GUARD-DF-012 broke.
func TestCalibrationCleanRunKeepsMetrics(t *testing.T) {
	out, rc := runCalibration(t, cleanStub)

	if rc != 0 {
		t.Errorf("exit code = %d, want 0 for a fully-classified run:\n%s", rc, out)
	}
	if strings.Contains(out, "RUN INVALID") {
		t.Errorf("clean run must not print the invalid banner:\n%s", out)
	}
	if strings.Contains(out, "-> error") {
		t.Errorf("clean run must have no error bucket in the confusion table:\n%s", out)
	}
	if !strings.Contains(out, "precision=") {
		t.Errorf("clean run must keep the precision/recall block:\n%s", out)
	}
	n := corpusRowCount(t)
	if !strings.Contains(out, fmt.Sprintf("0/%d rows errored", n)) {
		t.Errorf("clean run should state its validity explicitly (0/%d rows errored):\n%s", n, out)
	}
	// Both arms must be self-consistent on a fully-classified run: each
	// arm's tp+fn must equal the number of positive rows (attack+quoted).
	// The GUARD-DF-012 break was exactly here: the review arm's fn=0 while
	// the block arm's fn=37 on the same all-errored run, because errored
	// rows satisfied neither arm's positive-band test nor the review arm's
	// allow-band test (the block arm counts them as fn by exclusion).
	pos := 0
	for _, line := range strings.Split(mustRead(t, "corpus.jsonl"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		var row struct {
			Label string `json:"label"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("corpus line: %v", err)
		}
		if row.Label == "attack" || row.Label == "quoted" {
			pos++
		}
	}
	if pos == 0 {
		t.Fatal("no positive rows found in corpus (label key spacing changed?)")
	}
	arms := regexp.MustCompile(`tp=(\d+) fp=(\d+) fn=(\d+)`).FindAllStringSubmatch(out, -1)
	if len(arms) < 2 {
		t.Fatalf("expected both metric arms to print tp/fp/fn, got %d matches:\n%s", len(arms), out)
	}
	for i, a := range arms[:2] {
		var tp, fn int
		fmt.Sscanf(a[1], "%d", &tp)
		fmt.Sscanf(a[3], "%d", &fn)
		if tp+fn != pos {
			t.Errorf("metric arm %d: tp+fn=%d, want %d (every positive row must land in exactly one of tp/fn)", i+1, tp+fn, pos)
		}
	}
}

// mustRead reads a file in the package dir, failing the test on error.
func mustRead(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(b)
}
