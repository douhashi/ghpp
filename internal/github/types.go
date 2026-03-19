package github

// ProjectItem represents a single item in a GitHub Project.
type ProjectItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

// ProjectMeta holds project-level metadata needed for mutations.
type ProjectMeta struct {
	ProjectID string
	FieldID   string            // Status フィールドの ID
	Options   map[string]string // ステータス名 -> Option ID のマップ
}

// PromoteResult represents the outcome of a single promote attempt.
type PromoteResult struct {
	Item     ProjectItem `json:"item"`
	Action   string      `json:"action"` // "promoted" or "skipped"
	Reason   string      `json:"reason,omitempty"`
	ToStatus string      `json:"to_status,omitempty"`
}

// PhaseSummary holds counts for a single phase or the overall response.
type PhaseSummary struct {
	Promoted int `json:"promoted"`
	Skipped  int `json:"skipped"`
	Total    int `json:"total"`
}

// PhaseResult groups the summary and individual results for one phase.
type PhaseResult struct {
	Summary PhaseSummary    `json:"summary"`
	Results []PromoteResult `json:"results"`
}

// PromotePhases holds results for each promotion phase as explicit fields
// so that both keys always appear in JSON output, even when empty.
type PromotePhases struct {
	Plan  PhaseResult `json:"plan"`
	Doing PhaseResult `json:"doing"`
}

// PromoteResponse is the top-level JSON output of the promote command.
type PromoteResponse struct {
	Summary PhaseSummary  `json:"summary"`
	Phases  PromotePhases `json:"phases"`
}
