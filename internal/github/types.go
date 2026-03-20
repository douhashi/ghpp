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

// PromotedItem represents a single item that was promoted.
type PromotedItem struct {
	Item     ProjectItem `json:"item"`
	Key      string      `json:"key,omitempty"`
	ToStatus string      `json:"to_status"`
}

// SkippedItem represents a single item that was skipped.
type SkippedItem struct {
	Item   ProjectItem `json:"item"`
	Reason string      `json:"reason"`
}

// PhaseResults groups promoted and skipped items for one phase.
type PhaseResults struct {
	Promoted []PromotedItem `json:"promoted"`
	Skipped  []SkippedItem  `json:"skipped"`
}

// PhaseSummary holds counts for a single phase or the overall response.
type PhaseSummary struct {
	Promoted int `json:"promoted"`
	Skipped  int `json:"skipped"`
	Total    int `json:"total"`
}

// PhaseResult groups the summary and individual results for one phase.
type PhaseResult struct {
	Summary PhaseSummary `json:"summary"`
	Results PhaseResults `json:"results"`
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
