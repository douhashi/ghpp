package github

// ProjectItem represents a single item in a GitHub Project.
type ProjectItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Status string `json:"status"`
}
