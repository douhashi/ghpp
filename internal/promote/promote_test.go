package promote

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// mockPromoter implements github.ItemPromoter for testing.
type mockPromoter struct {
	meta      *github.ProjectMeta
	metaErr   error
	updateErr error
	updated   []updateCall
}

type updateCall struct {
	ItemID     string
	StatusName string
}

func (m *mockPromoter) FetchProjectMeta(_ context.Context, _ string, _ int) (*github.ProjectMeta, error) {
	return m.meta, m.metaErr
}

func (m *mockPromoter) UpdateItemStatus(_ context.Context, _ *github.ProjectMeta, itemID string, statusName string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, updateCall{ItemID: itemID, StatusName: statusName})
	return nil
}

var defaultMeta = &github.ProjectMeta{
	ProjectID: "PVT_001",
	FieldID:   "PVTSSF_001",
	Options: map[string]string{
		"Backlog":     "opt1",
		"Plan":        "opt2",
		"Ready":       "opt3",
		"In progress": "opt4",
		"Done":        "opt5",
	},
}

func defaultCfg() *config.Config {
	return &config.Config{
		Owner:         "testowner",
		ProjectNumber: 1,
		StatusInbox:   "Backlog",
		StatusPlan:    "Plan",
		StatusReady:   "Ready",
		StatusDoing:   "In progress",
		PlanLimit:     0,
	}
}

func TestPlanPhase_InboxToPlan(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog"},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo/issues/2", Status: "Done"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := filterByAction(results, "promoted")
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if promoted[0].Item.ID != "1" {
		t.Errorf("promoted item ID = %q, want %q", promoted[0].Item.ID, "1")
	}
	if promoted[0].ToStatus != "Plan" {
		t.Errorf("ToStatus = %q, want %q", promoted[0].ToStatus, "Plan")
	}
}

func TestPlanPhase_PlanLimitExceeded(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PlanLimit = 1
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog"},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo/issues/2", Status: "Backlog"},
		{ID: "3", Title: "Issue 3", URL: "https://github.com/owner/repo/issues/3", Status: "Backlog"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := filterByAction(results, "promoted")
	skipped := filterByAction(results, "skipped")
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if len(skipped) != 2 {
		t.Fatalf("expected 2 skipped, got %d", len(skipped))
	}
	if skipped[0].Reason != "plan limit reached" {
		t.Errorf("Reason = %q, want %q", skipped[0].Reason, "plan limit reached")
	}
}

func TestPlanPhase_NoInboxItems(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Ready"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	planPromoted := 0
	for _, r := range results {
		if r.ToStatus == "Plan" {
			planPromoted++
		}
	}
	if planPromoted != 0 {
		t.Errorf("expected 0 plan promotions, got %d", planPromoted)
	}
}

func TestPlanPhase_PlanLimitZeroPromotesAll(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PlanLimit = 0
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog"},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo/issues/2", Status: "Backlog"},
		{ID: "3", Title: "Issue 3", URL: "https://github.com/owner/repo/issues/3", Status: "Backlog"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := filterByAction(results, "promoted")
	if len(promoted) != 3 {
		t.Fatalf("expected 3 promoted, got %d", len(promoted))
	}
}

func TestDoingPhase_ReadyToDoing(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := filterByAction(results, "promoted")
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if promoted[0].ToStatus != "In progress" {
		t.Errorf("ToStatus = %q, want %q", promoted[0].ToStatus, "In progress")
	}
}

func TestDoingPhase_SameRepoSecondSkipped(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready"},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo-a/issues/2", Status: "Ready"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := filterByAction(results, "promoted")
	skipped := filterByAction(results, "skipped")
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(skipped))
	}
	if skipped[0].Reason != "repository already has doing issue" {
		t.Errorf("Reason = %q, want %q", skipped[0].Reason, "repository already has doing issue")
	}
}

func TestDoingPhase_ExistingDoingRepoSkipped(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Doing Issue", URL: "https://github.com/owner/repo-a/issues/1", Status: "In progress"},
		{ID: "2", Title: "Ready Issue", URL: "https://github.com/owner/repo-a/issues/2", Status: "Ready"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skipped := filterByAction(results, "skipped")
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(skipped))
	}
	if skipped[0].Item.ID != "2" {
		t.Errorf("skipped item ID = %q, want %q", skipped[0].Item.ID, "2")
	}
}

func TestDoingPhase_DifferentReposBothPromoted(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready"},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo-b/issues/1", Status: "Ready"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := filterByAction(results, "promoted")
	if len(promoted) != 2 {
		t.Fatalf("expected 2 promoted, got %d", len(promoted))
	}
}

func TestExtractRepo(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "normal issue URL",
			url:  "https://github.com/owner/repo/issues/123",
			want: "owner/repo",
		},
		{
			name: "pull request URL",
			url:  "https://github.com/org/project/pull/456",
			want: "org/project",
		},
		{
			name: "invalid URL with no slash",
			url:  "not-a-url",
			want: "",
		},
		{
			name: "too short path",
			url:  "https://github.com/owner",
			want: "",
		},
		{
			name: "empty string",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRepo(tt.url)
			if got != tt.want {
				t.Errorf("extractRepo(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestRun_BothPhasesCombined(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PlanLimit = 1
	items := []github.ProjectItem{
		{ID: "1", Title: "Inbox 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Backlog"},
		{ID: "2", Title: "Inbox 2", URL: "https://github.com/owner/repo-b/issues/1", Status: "Backlog"},
		{ID: "3", Title: "Ready 1", URL: "https://github.com/owner/repo-c/issues/1", Status: "Ready"},
		{ID: "4", Title: "Doing 1", URL: "https://github.com/owner/repo-d/issues/1", Status: "In progress"},
	}

	results, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := filterByAction(results, "promoted")
	skipped := filterByAction(results, "skipped")
	// Plan: 1 promoted (limit=1), 1 skipped. Doing: 1 promoted.
	if len(promoted) != 2 {
		t.Fatalf("expected 2 promoted, got %d", len(promoted))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(skipped))
	}
}

func TestRun_APIError(t *testing.T) {
	mp := &mockPromoter{
		meta:      defaultMeta,
		updateErr: errors.New("API error"),
	}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog"},
	}

	_, err := Run(context.Background(), cfg, items, mp)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestRun_FetchMetaError(t *testing.T) {
	mp := &mockPromoter{
		metaErr: errors.New("meta error"),
	}
	cfg := defaultCfg()

	_, err := Run(context.Background(), cfg, nil, mp)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func filterByAction(results []github.PromoteResult, action string) []github.PromoteResult {
	var filtered []github.PromoteResult
	for _, r := range results {
		if r.Action == action {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
