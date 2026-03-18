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
