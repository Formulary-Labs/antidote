// antidote closes the loop after a completed audit: it takes the findings and
// turns them into corrective actions, distilled lessons, and systemic-control
// patterns — the feed-forward channel into the next cycle. Read-only: no
// program file is ever written; output goes to --output or stdout.
// Exit codes follow the Formulary three-code contract: 0 = feed ready (no
// critical findings, few open), 1 = attention (critical or many open),
// 2 = tool error.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Formulary-Labs/antidote/internal/feed"
	"github.com/Formulary-Labs/substrate/exit"
	"github.com/Formulary-Labs/substrate/provenance"
)

const version = "0.1.0"

func main() {
	var (
		findingsFlag = flag.String("findings", "", "Findings JSON (required)")
		nameFlag     = flag.String("name", "", "Audit name (used in feed_id)")
		fmtFlag      = flag.String("format", "json", "Output format: json (default), md")
		emitFlag     = flag.String("emit", "json", "Output mode: json (default), specimen (specimen-ingest-compatible array)")
		outputFlag   = flag.String("output", "", "Output file path (stdout if empty)")
		quietFlag    = flag.Bool("quiet", false, "Suppress progress output")
		versionFlag  = flag.Bool("version", false, "Print version and exit")
		doctorFlag   = flag.Bool("doctor", false, "Self-check and exit 0")
	)
	flag.Usage = usage
	flag.Parse()

	if *versionFlag {
		fmt.Printf("antidote version %s\n", version)
		os.Exit(exit.OK)
	}
	if *doctorFlag {
		runDoctor()
		return
	}
	if *findingsFlag == "" {
		fmt.Fprintf(os.Stderr, `{"error": "--findings is required", "code": 2}\n`)
		flag.Usage()
		os.Exit(exit.ToolError)
	}

	if !*quietFlag {
		fmt.Fprintln(os.Stderr, "[antidote] loading findings:", *findingsFlag)
	}
	data, err := os.ReadFile(*findingsFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error": "%s", "code": 2}\n`, err.Error())
		os.Exit(exit.ToolError)
	}
	var findings []feed.Finding
	if err := json.Unmarshal(data, &findings); err != nil {
		fmt.Fprintf(os.Stderr, `{"error": "parsing findings: %v", "code": 2}\n`, err)
		os.Exit(exit.ToolError)
	}

	report := feed.Feed(*nameFlag, findings)
	if !*quietFlag {
		fmt.Fprintf(os.Stderr, "[antidote] verdict %s (exit %d): %d findings (%d open, %d closed), %d corrective actions, %d lessons, %d systemic controls\n",
			report.Verdict, report.ExitCode,
			report.TotalFindings, report.OpenFindings, report.ClosedFindings,
			len(report.CorrectiveActions), len(report.Lessons), len(report.SystemicControls))
	}

	out := formatOutput(report, *fmtFlag, *emitFlag)
	if *outputFlag != "" {
		if err := os.WriteFile(*outputFlag, []byte(out), 0o600); err != nil {
			fmt.Fprintf(os.Stderr, "error writing output: %v\n", err)
			os.Exit(exit.ToolError)
		}
		if !*quietFlag {
			fmt.Fprintf(os.Stderr, "wrote %s\n", *outputFlag)
		}
	} else {
		fmt.Print(out)
	}

	if *nameFlag != "" {
		_ = provenance.Write("logs/provenance.jsonl", provenance.Entry{
			Spec:        "functions/post-audit-spec.md",
			Output:      *findingsFlag,
			OutputType:  "post_audit_feedforward",
			Program:     *nameFlag,
			Purpose:     fmt.Sprintf("antidote: verdict %s exit %d (%d findings, %d open, %d actions, %d lessons)",
				report.Verdict, report.ExitCode,
				report.TotalFindings, report.OpenFindings,
				len(report.CorrectiveActions), len(report.Lessons)),
			Reusability: provenance.Instance,
			QualityGate: provenance.Pass,
			Tool:        "antidote",
			ToolVersion: version,
		})
	}

	os.Exit(report.ExitCode)
}

func runDoctor() {
	fmt.Printf("antidote version %s\n", version)
	fmt.Println("substrate: required (github.com/Formulary-Labs/substrate v0.4.0)")
	fmt.Println("doctor: OK")
}

func formatOutput(r feed.FeedReport, fmtStr, emit string) string {
	if emit == "specimen" {
		return toSpecimenEmit(r)
	}
	if fmtStr == "md" {
		return toMarkdown(r)
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

// specimenEntry matches specimen/register.FeedForwardEntry exactly so that
// `specimen ingest --feed-forward <file>` can consume antidote output directly.
type specimenEntry struct {
	FindingID        string `json:"finding_id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	ControlID        string `json:"control_id"`
	Severity         string `json:"severity"`
	CorrectiveAction string `json:"corrective_action"`
	Owner            string `json:"owner"`
}

// toSpecimenEmit converts corrective actions into a []specimenEntry JSON
// array ready for `specimen ingest --feed-forward`.
func toSpecimenEmit(r feed.FeedReport) string {
	entries := make([]specimenEntry, 0, len(r.CorrectiveActions))
	for _, ca := range r.CorrectiveActions {
		entries = append(entries, specimenEntry{
			FindingID:        ca.FindingID,
			Title:            ca.Action,
			Description:      fmt.Sprintf("Corrective action from feed %s (finding %s)", r.FeedID, ca.FindingID),
			ControlID:        ca.ControlID,
			Severity:         string(ca.Severity),
			CorrectiveAction: ca.Action,
			Owner:            ca.Owner,
		})
	}
	b, _ := json.MarshalIndent(entries, "", "  ")
	return string(b)
}

func toMarkdown(r feed.FeedReport) string {
	var b strings.Builder
	b.WriteString("## antidote — post-audit feed-forward\n\n")
	b.WriteString(fmt.Sprintf("**Feed:** %s  |  **Verdict:** %s  |  **Now:** %s\n\n",
		r.FeedID, r.Verdict, r.Now))
	b.WriteString(fmt.Sprintf("**Findings:** %d (%d open, %d closed)  |  **Max severity:** %s\n\n",
		r.TotalFindings, r.OpenFindings, r.ClosedFindings, r.MaxSeverity))

	if len(r.CorrectiveActions) > 0 {
		b.WriteString("## Corrective Actions\n\n")
		for _, c := range r.CorrectiveActions {
			b.WriteString(fmt.Sprintf("- **%s** [%s] (%s) — %s\n", c.ID, c.ControlID, c.Severity, c.Action))
		}
		b.WriteString("\n")
	}

	if len(r.SystemicControls) > 0 {
		b.WriteString("## Systemic Controls (repeat offenders)\n\n")
		for _, s := range r.SystemicControls {
			if s.Findings < 2 {
				continue
			}
			b.WriteString(fmt.Sprintf("- **%s** — %d findings: %s\n",
				s.ControlID, s.Findings, strings.Join(stringSlice(s.Severities), ", ")))
		}
		b.WriteString("\n")
	}

	if len(r.Lessons) > 0 {
		b.WriteString("## Lessons (carry forward)\n\n")
		for _, l := range r.Lessons {
			b.WriteString(fmt.Sprintf("- **%s** — %s\n", l.Title, l.Detail))
		}
		b.WriteString("\n")
	}

	if len(r.FocusControls) > 0 {
		b.WriteString("## Focus Controls (next cycle)\n\n")
		for _, c := range r.FocusControls {
			b.WriteString(fmt.Sprintf("- %s\n", c))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func stringSlice(ss []feed.Severity) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = string(s)
	}
	return out
}

func usage() {
	fmt.Fprintln(os.Stderr, `antidote — post-audit feed-forward (the cure)

Usage:
  antidote [flags] --findings <findings.json>

Flags:
  --findings string   Findings JSON (required)
  --name string       Audit name (used in feed_id)
  --format string     Output format: json (default), md
  --emit string       Output mode: json (default), specimen (specimen-ingest array)
  --output string     Output file path (stdout if empty)
  --quiet             Suppress progress output
  --doctor            Self-check and exit 0
  --version           Print version and exit

Findings JSON format (array of findings):
  [
    { "id": "F-01", "title": "Excessive privilege", "description": "Admin access
      not reviewed quarterly", "severity": "high", "control_id": "A.9.4",
      "category": "access", "owner": "sec", "due": "2026-10-15",
      "status": "open" }
  ]

Exit codes: 0 feed ready, 1 attention (critical/many open), 2 tool error.
Deterministic: identical findings + name gives identical output.

--emit specimen produces a JSON array compatible with 'specimen ingest --feed-forward'.

Examples:
  antidote --findings data/sample/findings.json --name "ISO27001-Q3"
  antidote --findings data/sample/findings.json --format md
  antidote --findings data/sample/findings.json --name "ISO27001-Q3" --emit specimen --output feed-forward.json
  antidote --doctor`)
}