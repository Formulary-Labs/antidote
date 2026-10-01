package feed

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// loadSample reads the sample findings file relative to the test.
func loadSample(t *testing.T) []Finding {
	b, err := os.ReadFile(samplePath(t, "findings.json"))
	t.Helper()
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	var f []Finding
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return f
}

func samplePath(t *testing.T, name string) string {
	root := moduleRoot(t)
	return filepath.Join(root, "data/sample", name)
}

func moduleRoot(t *testing.T) string {
	h, _ := os.Getwd()
	dir := h
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		p := filepath.Dir(dir)
		if p == dir {
			return h
		}
		dir = p
	}
}

func TestFeed_Sample(t *testing.T) {
	f := loadSample(t)
	r := Feed("ISO27001-Q3", f)
	t.Logf("verdict=%s exit=%d open=%d closed=%d actions=%d lessons=%d systemic=%d",
		r.Verdict, r.ExitCode, r.OpenFindings, r.ClosedFindings,
		len(r.CorrectiveActions), len(r.Lessons), len(r.SystemicControls))

	if r.OpenFindings != 3 || r.ClosedFindings != 2 {
		t.Fatalf("open/closed mismatch: got %d/%d", r.OpenFindings, r.ClosedFindings)
	}
	if !r.HasCritical {
		t.Fatalf("expected critical finding")
	}
	if r.Verdict != "attention" || r.ExitCode != 1 {
		t.Fatalf("expected attention/1, got %s/%d", r.Verdict, r.ExitCode)
	}
	if len(r.CorrectiveActions) != 3 {
		t.Fatalf("expected 3 corrective actions, got %d", len(r.CorrectiveActions))
	}
	found := false
	for _, sc := range r.SystemicControls {
		if sc.ControlID == "A.5.11" && sc.Findings == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected A.5.11 systemic control with 2 findings")
	}
	if len(r.Lessons) != 4 {
		t.Fatalf("expected 4 lessons, got %d", len(r.Lessons))
	}
}

func TestFeed_Determinism(t *testing.T) {
	f := loadSample(t)
	a := Feed("X", f)
	b := Feed("X", f)
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	if !bytes.Equal(ab, bb) {
		t.Fatalf("determinism violated")
	}
}

func TestFeed_AllClosed_Ready(t *testing.T) {
	f := []Finding{{
		ID:        "F-1",
		Title:     "old finding",
		Severity:  SeverityHigh,
		ControlID: "A.1.1",
		Status:    StatusClosed,
	}}
	r := Feed("X", f)
	if r.Verdict != "ready" || r.ExitCode != 0 {
		t.Fatalf("expected ready/0, got %s/%d", r.Verdict, r.ExitCode)
	}
	if len(r.CorrectiveActions) != 0 {
		t.Fatalf("expected no corrective actions for all-closed")
	}
}