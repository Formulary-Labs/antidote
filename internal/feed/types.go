package feed

// Severity is how serious a post-audit finding is.
type Severity string

const (
	SeverityInformational Severity = "informational"
	SeverityLow           Severity = "low"
	SeverityMedium        Severity = "medium"
	SeverityHigh          Severity = "high"
	SeverityCritical      Severity = "critical"
)

// rankSeverity orders severities so "worst wins" is trivial.
func rankSeverity(s Severity) int {
	switch s {
	case SeverityInformational:
		return 0
	case SeverityLow:
		return 1
	case SeverityMedium:
		return 2
	case SeverityHigh:
		return 3
	case SeverityCritical:
		return 4
	default:
		return -1
	}
}

// FindingStatus is the lifecycle state of a finding.
type FindingStatus string

const (
	StatusOpen     FindingStatus = "open"
	StatusClosed   FindingStatus = "closed"
	StatusAccepted FindingStatus = "accepted"
)

// Finding is a single observation from a completed audit.
type Finding struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Severity    Severity        `json:"severity"`
	ControlID   string          `json:"control_id"`
	Category    string          `json:"category,omitempty"`
	Owner       string          `json:"owner,omitempty"`
	Due         string          `json:"due,omitempty"` // ISO8601
	Status      FindingStatus   `json:"status"`
	Evidence    []string        `json:"evidence,omitempty"`
}

// CorrectiveAction is the concrete step that closes a finding.
type CorrectiveAction struct {
	ID         string        `json:"id"`
	FindingID  string        `json:"finding_id"`
	ControlID  string        `json:"control_id"`
	Action     string        `json:"action"`
	Owner      string        `json:"owner,omitempty"`
	Due        string        `json:"due,omitempty"`
	Status     string        `json:"status"`
	Severity   Severity      `json:"severity"`
}

// Lesson is a distilled, reusable insight fed forward into the next cycle.
type Lesson struct {
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Source  []string `json:"source_findings"` // finding IDs that produced this lesson
	Control string `json:"control_id,omitempty"`
}

// FeedReport is the feed-forward output of an antidote run. It is byte-stable
// for identical input: derived purely from the findings + clock, no randomness.
type FeedReport struct {
	Name              string          `json:"name,omitempty"`
	FeedID            string          `json:"feed_id"`
	Now               string          `json:"now,omitempty"`
	TotalFindings     int             `json:"total_findings"`
	OpenFindings      int             `json:"open_findings"`
	ClosedFindings    int             `json:"closed_findings"`
	MaxSeverity       Severity        `json:"max_severity"`
	HasCritical       bool            `json:"has_critical"`
	CorrectiveActions []CorrectiveAction `json:"corrective_actions"`
	Lessons           []Lesson          `json:"lessons"`
	SystemicControls  []SystemicControl `json:"systemic_controls,omitempty"`
	FocusControls     []string          `json:"focus_controls,omitempty"`
	Verdict           string            `json:"verdict"` // "ready" | "attention"
	ExitCode          int               `json:"exit_code"`
}

// SystemicControl is a control that appears in multiple findings (a
// pattern, not a one-off).
type SystemicControl struct {
	ControlID string `json:"control_id"`
	Findings  int    `json:"findings"`
	Severities []Severity `json:"severities"`
}