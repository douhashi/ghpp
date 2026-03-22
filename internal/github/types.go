package github

import "time"

// ProjectItem represents a single item in a GitHub Project.
type ProjectItem struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Status    string    `json:"status"`
	Body      string    `json:"body"`
	Labels    []string  `json:"labels"`
	UpdatedAt time.Time `json:"updated_at"`
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
	DryRun  bool          `json:"dry_run"`
	Summary PhaseSummary  `json:"summary"`
	Phases  PromotePhases `json:"phases"`
}

// DemotedItem represents a single item that was demoted.
type DemotedItem struct {
	Item       ProjectItem `json:"item"`
	Key        string      `json:"key,omitempty"`
	FromStatus string      `json:"from_status"`
	ToStatus   string      `json:"to_status"`
}

// DemotePhaseResults groups demoted and skipped items for one phase.
type DemotePhaseResults struct {
	Demoted []DemotedItem `json:"demoted"`
	Skipped []SkippedItem `json:"skipped"`
}

// DemoteSummary holds counts for a single demote phase or the overall demote response.
type DemoteSummary struct {
	Demoted int `json:"demoted"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

// DemotePhaseResult groups the summary and individual results for one demote phase.
type DemotePhaseResult struct {
	Summary DemoteSummary      `json:"summary"`
	Results DemotePhaseResults `json:"results"`
}

// DemotePhases holds results for each demotion phase as explicit fields
// so that both keys always appear in JSON output, even when empty.
type DemotePhases struct {
	Doing DemotePhaseResult `json:"doing"`
	Plan  DemotePhaseResult `json:"plan"`
}

// DemoteResponse is the top-level JSON output of the demote command.
type DemoteResponse struct {
	DryRun  bool          `json:"dry_run"`
	Summary DemoteSummary `json:"summary"`
	Phases  DemotePhases  `json:"phases"`
}
