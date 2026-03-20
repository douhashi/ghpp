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

func (m *mockPromoter) FetchProjectItems(_ context.Context, _ string, _ int) ([]github.ProjectItem, error) {
	return nil, nil
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

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Plan.Results.Promoted
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if promoted[0].Item.ID != "1" {
		t.Errorf("promoted item ID = %q, want %q", promoted[0].Item.ID, "1")
	}
	if promoted[0].ToStatus != "Plan" {
		t.Errorf("ToStatus = %q, want %q", promoted[0].ToStatus, "Plan")
	}
	if promoted[0].Key != "plan-owner-repo-1" {
		t.Errorf("Key = %q, want %q", promoted[0].Key, "plan-owner-repo-1")
	}
	if resp.Phases.Plan.Summary.Promoted != 1 {
		t.Errorf("plan summary promoted = %d, want 1", resp.Phases.Plan.Summary.Promoted)
	}
	if resp.Phases.Plan.Summary.Total != 1 {
		t.Errorf("plan summary total = %d, want 1", resp.Phases.Plan.Summary.Total)
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

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Plan.Results.Promoted
	skipped := resp.Phases.Plan.Results.Skipped
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if len(skipped) != 2 {
		t.Fatalf("expected 2 skipped, got %d", len(skipped))
	}
	if skipped[0].Reason != "plan limit reached" {
		t.Errorf("Reason = %q, want %q", skipped[0].Reason, "plan limit reached")
	}
	if promoted[0].Key != "plan-owner-repo-1" {
		t.Errorf("promoted Key = %q, want %q", promoted[0].Key, "plan-owner-repo-1")
	}
	if resp.Phases.Plan.Summary.Promoted != 1 {
		t.Errorf("plan summary promoted = %d, want 1", resp.Phases.Plan.Summary.Promoted)
	}
	if resp.Phases.Plan.Summary.Skipped != 2 {
		t.Errorf("plan summary skipped = %d, want 2", resp.Phases.Plan.Summary.Skipped)
	}
	if resp.Phases.Plan.Summary.Total != 3 {
		t.Errorf("plan summary total = %d, want 3", resp.Phases.Plan.Summary.Total)
	}
}

func TestPlanPhase_NoInboxItems(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Ready"},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Phases.Plan.Summary.Promoted != 0 {
		t.Errorf("expected 0 plan promotions, got %d", resp.Phases.Plan.Summary.Promoted)
	}
	if len(resp.Phases.Plan.Results.Promoted) != 0 {
		t.Errorf("expected 0 plan promoted, got %d", len(resp.Phases.Plan.Results.Promoted))
	}
	if len(resp.Phases.Plan.Results.Skipped) != 0 {
		t.Errorf("expected 0 plan skipped, got %d", len(resp.Phases.Plan.Results.Skipped))
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

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Plan.Results.Promoted
	if len(promoted) != 3 {
		t.Fatalf("expected 3 promoted, got %d", len(promoted))
	}
	if resp.Phases.Plan.Summary.Promoted != 3 {
		t.Errorf("plan summary promoted = %d, want 3", resp.Phases.Plan.Summary.Promoted)
	}
}

func TestDoingPhase_ReadyToDoing(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready"},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Doing.Results.Promoted
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if promoted[0].ToStatus != "In progress" {
		t.Errorf("ToStatus = %q, want %q", promoted[0].ToStatus, "In progress")
	}
	if promoted[0].Key != "doing-owner-repo-a-1" {
		t.Errorf("Key = %q, want %q", promoted[0].Key, "doing-owner-repo-a-1")
	}
	if resp.Phases.Doing.Summary.Promoted != 1 {
		t.Errorf("doing summary promoted = %d, want 1", resp.Phases.Doing.Summary.Promoted)
	}
}

func TestDoingPhase_SameRepoSecondSkipped(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready"},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo-a/issues/2", Status: "Ready"},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Doing.Results.Promoted
	skipped := resp.Phases.Doing.Results.Skipped
	if len(promoted) != 1 {
		t.Fatalf("expected 1 promoted, got %d", len(promoted))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(skipped))
	}
	if skipped[0].Reason != "repository already has doing issue" {
		t.Errorf("Reason = %q, want %q", skipped[0].Reason, "repository already has doing issue")
	}
	if promoted[0].Key != "doing-owner-repo-a-1" {
		t.Errorf("promoted Key = %q, want %q", promoted[0].Key, "doing-owner-repo-a-1")
	}
	if resp.Phases.Doing.Summary.Promoted != 1 {
		t.Errorf("doing summary promoted = %d, want 1", resp.Phases.Doing.Summary.Promoted)
	}
	if resp.Phases.Doing.Summary.Skipped != 1 {
		t.Errorf("doing summary skipped = %d, want 1", resp.Phases.Doing.Summary.Skipped)
	}
}

func TestDoingPhase_ExistingDoingRepoSkipped(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Doing Issue", URL: "https://github.com/owner/repo-a/issues/1", Status: "In progress"},
		{ID: "2", Title: "Ready Issue", URL: "https://github.com/owner/repo-a/issues/2", Status: "Ready"},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	skipped := resp.Phases.Doing.Results.Skipped
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

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Doing.Results.Promoted
	if len(promoted) != 2 {
		t.Fatalf("expected 2 promoted, got %d", len(promoted))
	}
	if resp.Phases.Doing.Summary.Promoted != 2 {
		t.Errorf("doing summary promoted = %d, want 2", resp.Phases.Doing.Summary.Promoted)
	}
}

func TestExtractKey(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		phase string
		want  string
	}{
		{
			name:  "issue URL with plan phase",
			url:   "https://github.com/owner/repo/issues/123",
			phase: "plan",
			want:  "plan-owner-repo-123",
		},
		{
			name:  "pull request URL with doing phase",
			url:   "https://github.com/org/project/pull/456",
			phase: "doing",
			want:  "doing-org-project-456",
		},
		{
			name:  "too short path",
			url:   "https://github.com/owner/repo",
			phase: "plan",
			want:  "",
		},
		{
			name:  "empty URL",
			url:   "",
			phase: "plan",
			want:  "",
		},
		{
			name:  "invalid URL",
			url:   "not-a-url",
			phase: "doing",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKey(tt.url, tt.phase)
			if got != tt.want {
				t.Errorf("extractKey(%q, %q) = %q, want %q", tt.url, tt.phase, got, tt.want)
			}
		})
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

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan: 1 promoted (limit=1), 1 skipped. Doing: 1 promoted.
	if resp.Phases.Plan.Summary.Promoted != 1 {
		t.Errorf("plan promoted = %d, want 1", resp.Phases.Plan.Summary.Promoted)
	}
	if resp.Phases.Plan.Summary.Skipped != 1 {
		t.Errorf("plan skipped = %d, want 1", resp.Phases.Plan.Summary.Skipped)
	}
	if resp.Phases.Doing.Summary.Promoted != 1 {
		t.Errorf("doing promoted = %d, want 1", resp.Phases.Doing.Summary.Promoted)
	}
	if resp.Summary.Promoted != 2 {
		t.Errorf("total promoted = %d, want 2", resp.Summary.Promoted)
	}
	if resp.Summary.Skipped != 1 {
		t.Errorf("total skipped = %d, want 1", resp.Summary.Skipped)
	}
	if resp.Summary.Total != 3 {
		t.Errorf("total = %d, want 3", resp.Summary.Total)
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

func TestRun_EmptyItems_ResultsNotNil(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()

	resp, err := Run(context.Background(), cfg, []github.ProjectItem{}, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Phases.Plan.Results.Promoted == nil {
		t.Error("plan promoted should not be nil")
	}
	if resp.Phases.Plan.Results.Skipped == nil {
		t.Error("plan skipped should not be nil")
	}
	if resp.Phases.Doing.Results.Promoted == nil {
		t.Error("doing promoted should not be nil")
	}
	if resp.Phases.Doing.Results.Skipped == nil {
		t.Error("doing skipped should not be nil")
	}
	if len(resp.Phases.Plan.Results.Promoted) != 0 {
		t.Errorf("plan promoted length = %d, want 0", len(resp.Phases.Plan.Results.Promoted))
	}
	if len(resp.Phases.Doing.Results.Promoted) != 0 {
		t.Errorf("doing promoted length = %d, want 0", len(resp.Phases.Doing.Results.Promoted))
	}
	if len(resp.Phases.Plan.Results.Skipped) != 0 {
		t.Errorf("plan skipped length = %d, want 0", len(resp.Phases.Plan.Results.Skipped))
	}
	if len(resp.Phases.Doing.Results.Skipped) != 0 {
		t.Errorf("doing skipped length = %d, want 0", len(resp.Phases.Doing.Results.Skipped))
	}
}
