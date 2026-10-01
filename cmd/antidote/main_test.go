package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Formulary-Labs/substrate/provenance"
)

func moduleRoot() string {
	h, _ := os.Getwd()
	dir := h
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return h
		}
		dir = next
	}
}

func buildBinary(t *testing.T) string {
	tmp := t.TempDir()
	out := filepath.Join(tmp, "antidote.exe")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/antidote")
	cmd.Dir = moduleRoot()
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v: %s", err, b)
	}
	return out
}

// runBinary runs the binary with an explicit working dir and returns the
// process exit code and combined output. A returned error means the process
// could not be launched; exit codes are a value.
func runBinary(bin string, workdir string, args ...string) (int, string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = workdir
	b, err := cmd.CombinedOutput()
	if err == nil {
		return 0, string(b), nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode(), string(b), nil
	}
	return -1, string(b), err
}

func TestCLI_Attention(t *testing.T) {
	b := buildBinary(t)
	root := moduleRoot()
	flags := []string{
		"--findings", filepath.Join(root, "data/sample/findings.json"),
		"--name", "ISO27001-Q3",
	}
	ec, out, err := runBinary(b, t.TempDir(), flags...)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	t.Logf("exit=%d out=%s", ec, out)
	if ec != 1 {
		t.Fatalf("expected exit 1, got %d", ec)
	}
}

func TestCLI_Ready(t *testing.T) {
	b := buildBinary(t)
	tmp := t.TempDir()
	clean := filepath.Join(tmp, "clean.json")
	findings := []map[string]string{
		{"id": "F-1", "title": "old", "severity": "low",
			"control_id": "A.1.1", "status": "closed"},
	}
	jb, _ := json.Marshal(findings)
	if err := os.WriteFile(clean, jb, 0o644); err != nil {
		t.Fatal(err)
	}
	ec, _, err := runBinary(b, tmp, "--findings", clean, "--name", "X")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ec != 0 {
		t.Fatalf("expected exit 0 for all-closed, got %d", ec)
	}
}

func TestCLI_ToolError(t *testing.T) {
	b := buildBinary(t)
	ec, _, err := runBinary(b, t.TempDir())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ec != 2 {
		t.Fatalf("expected exit 2 (missing findings), got %d", ec)
	}
}

func TestCLI_ProvenanceWritten(t *testing.T) {
	b := buildBinary(t)
	root := moduleRoot()
	workdir := t.TempDir()
	ec, _, err := runBinary(b, workdir,
		"--findings", filepath.Join(root, "data/sample/findings.json"),
		"--name", "Prove",
	)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	t.Logf("exit=%d", ec)
	_, err = provenance.Verify(filepath.Join(workdir, "logs/provenance.jsonl"))
	if err != nil {
		t.Fatalf("provenance verify failed: %v", err)
	}
}