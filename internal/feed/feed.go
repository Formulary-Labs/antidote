package feed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Feed runs the full feed-forward pass: findings become corrective actions,
// lessons, and systemic-control patterns, then a report ready to be consumed
// by the next cycle (intake / monitoring). Deterministic: identical findings
// + name give identical output. No randomness.
func Feed(name string, findings []Finding) FeedReport {
	r := FeedReport{}
	r.TotalFindings = len(findings)

	open, closed := 0, 0
	var max Severity
	hasCritical := false
	byControl := map[string][]Finding{}

	for _, f := range findings {
		switch f.Status {
		case StatusOpen:
			open++
		default:
			closed++
		}
		if rankSeverity(f.Severity) > rankSeverity(max) {
			max = f.Severity
		}
		if f.Severity == SeverityCritical {
			hasCritical = true
		}
		byControl[f.ControlID] = append(byControl[f.ControlID], f)
	}
	r.OpenFindings = open
	r.ClosedFindings = closed
	r.MaxSeverity = max
	r.HasCritical = hasCritical

	// --- corrective actions: one per OPEN finding ---------------------------
	for _, f := range findings {
		if f.Status != StatusOpen {
			continue
		}
		owner := f.Owner
		if owner == "" {
			owner = "unassigned"
		}
		r.CorrectiveActions = append(r.CorrectiveActions, CorrectiveAction{
			ID:        "CA-"+f.ID,
			FindingID: f.ID,
			ControlID: f.ControlID,
			Action:    fmt.Sprintf("Remediate %q — %s", f.Title, f.Description),
			Owner:     owner,
			Due:       f.Due,
			Status:    "open",
			Severity:  f.Severity,
		})
	}
	if r.CorrectiveActions == nil {
		r.CorrectiveActions = []CorrectiveAction{}
	}

	// --- lessons + systemic controls: deterministic by sorted control ID ----
	controls := make([]string, 0, len(byControl))
	for k := range byControl {
		controls = append(controls, k)
	}
	sort.Strings(controls)

	r.Lessons = []Lesson{}
	r.SystemicControls = []SystemicControl{}
	var focus []string
	for _, cid := range controls {
		fs := byControl[cid]
		cats := map[string]bool{}
		for _, f := range fs {
			cats[f.Category] = true
		}
		catList := make([]string, 0, len(cats))
		for c := range cats {
			catList = append(catList, c)
		}
		sort.Strings(catList)
		r.Lessons = append(r.Lessons, Lesson{
			Title:   fmt.Sprintf("Strengthen %s", cid),
			Detail:  fmt.Sprintf("%d findings on control %s across categories: %s", len(fs), cid, strings.Join(catList, ", ")),
			Source:  findingIDs(fs),
			Control: cid,
		})
		r.SystemicControls = append(r.SystemicControls, SystemicControl{
			ControlID:  cid,
			Findings:   len(fs),
			Severities: severities(fs),
		})
		if len(fs) >= 2 {
			focus = append(focus, cid)
		}
	}
	if r.Lessons == nil {
		r.Lessons = []Lesson{}
	}
	if r.SystemicControls == nil {
		r.SystemicControls = []SystemicControl{}
	}
	if focus == nil {
		focus = []string{}
	}
	r.FocusControls = focus

	// --- verdict + exit code -----------------------------------------------
	r.FeedID = feedID(name, findings)
	if hasCritical || open >= 5 {
		r.Verdict = "attention"
		r.ExitCode = 1
	} else {
		r.Verdict = "ready"
		r.ExitCode = 0
	}
	return r
}

func feedID(name string, findings []Finding) string {
	b, _ := json.Marshal(struct {
		Name     string    `json:"name"`
		Findings []Finding `json:"findings"`
	}{name, findings})
	h := sha256.Sum256(b)
	return fmt.Sprintf("antidote/%s/%s", name, hex.EncodeToString(h[:8]))
}

func findingIDs(fs []Finding) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.ID
	}
	return out
}

func severities(fs []Finding) []Severity {
	out := make([]Severity, len(fs))
	for i, f := range fs {
		out[i] = f.Severity
	}
	return out
}