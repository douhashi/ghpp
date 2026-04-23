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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog", Labels: []string{}},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo/issues/2", Status: "Done", Labels: []string{}},
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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog", Labels: []string{}},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo/issues/2", Status: "Backlog", Labels: []string{}},
		{ID: "3", Title: "Issue 3", URL: "https://github.com/owner/repo/issues/3", Status: "Backlog", Labels: []string{}},
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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Ready", Labels: []string{}},
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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog", Labels: []string{}},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo/issues/2", Status: "Backlog", Labels: []string{}},
		{ID: "3", Title: "Issue 3", URL: "https://github.com/owner/repo/issues/3", Status: "Backlog", Labels: []string{}},
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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready", Labels: []string{}},
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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready", Labels: []string{}},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo-a/issues/2", Status: "Ready", Labels: []string{}},
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
		{ID: "1", Title: "Doing Issue", URL: "https://github.com/owner/repo-a/issues/1", Status: "In progress", Labels: []string{}},
		{ID: "2", Title: "Ready Issue", URL: "https://github.com/owner/repo-a/issues/2", Status: "Ready", Labels: []string{}},
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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready", Labels: []string{}},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo-b/issues/1", Status: "Ready", Labels: []string{}},
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

func TestRun_BothPhasesCombined(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PlanLimit = 1
	items := []github.ProjectItem{
		{ID: "1", Title: "Inbox 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Backlog", Labels: []string{}},
		{ID: "2", Title: "Inbox 2", URL: "https://github.com/owner/repo-b/issues/1", Status: "Backlog", Labels: []string{}},
		{ID: "3", Title: "Ready 1", URL: "https://github.com/owner/repo-c/issues/1", Status: "Ready", Labels: []string{}},
		{ID: "4", Title: "Doing 1", URL: "https://github.com/owner/repo-d/issues/1", Status: "In progress", Labels: []string{}},
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
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog", Labels: []string{}},
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

func TestPlanPhase_DryRun(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.DryRun = true
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog", Labels: []string{}},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo/issues/2", Status: "Backlog", Labels: []string{}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// dry-run では UpdateItemStatus が呼ばれない
	if len(mp.updated) != 0 {
		t.Errorf("expected 0 UpdateItemStatus calls, got %d", len(mp.updated))
	}

	// promoted スライスにはアイテムが含まれる（出力は通常と同じ）
	promoted := resp.Phases.Plan.Results.Promoted
	if len(promoted) != 2 {
		t.Fatalf("expected 2 promoted in output, got %d", len(promoted))
	}

	// DryRun フィールドが true
	if !resp.DryRun {
		t.Error("expected DryRun = true, got false")
	}
}

func TestDoingPhase_DryRun(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.DryRun = true
	items := []github.ProjectItem{
		{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Ready", Labels: []string{}},
		{ID: "2", Title: "Issue 2", URL: "https://github.com/owner/repo-b/issues/1", Status: "Ready", Labels: []string{}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// dry-run では UpdateItemStatus が呼ばれない
	if len(mp.updated) != 0 {
		t.Errorf("expected 0 UpdateItemStatus calls, got %d", len(mp.updated))
	}

	// promoted スライスにはアイテムが含まれる（出力は通常と同じ）
	promoted := resp.Phases.Doing.Results.Promoted
	if len(promoted) != 2 {
		t.Fatalf("expected 2 promoted in output, got %d", len(promoted))
	}

	// DryRun フィールドが true
	if !resp.DryRun {
		t.Error("expected DryRun = true, got false")
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
	if resp.Phases.Ready.Results.Promoted == nil {
		t.Error("ready promoted should not be nil")
	}
	if resp.Phases.Ready.Results.Skipped == nil {
		t.Error("ready skipped should not be nil")
	}
}

func TestReadyPhase_Disabled(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = false
	items := []github.ProjectItem{
		{ID: "1", Title: "Plan Issue", URL: "https://github.com/owner/repo/issues/1", Status: "Plan", Labels: []string{"planned"}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 機能無効時: ready フェーズで昇格しない
	if len(resp.Phases.Ready.Results.Promoted) != 0 {
		t.Errorf("expected 0 ready promoted, got %d", len(resp.Phases.Ready.Results.Promoted))
	}
	if resp.Phases.Ready.Summary.Promoted != 0 {
		t.Errorf("ready summary promoted = %d, want 0", resp.Phases.Ready.Summary.Promoted)
	}
	// API 呼び出しなし
	if len(mp.updated) != 0 {
		t.Errorf("expected 0 UpdateItemStatus calls, got %d", len(mp.updated))
	}
}

func TestReadyPhase_LabelMatch(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = true
	cfg.PlannedLabel = "planned"
	items := []github.ProjectItem{
		{ID: "1", Title: "Plan Issue", URL: "https://github.com/owner/repo/issues/1", Status: "Plan", Labels: []string{"planned"}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Ready.Results.Promoted
	if len(promoted) != 1 {
		t.Fatalf("expected 1 ready promoted, got %d", len(promoted))
	}
	if promoted[0].Item.ID != "1" {
		t.Errorf("promoted item ID = %q, want %q", promoted[0].Item.ID, "1")
	}
	if promoted[0].ToStatus != cfg.StatusReady {
		t.Errorf("ToStatus = %q, want %q", promoted[0].ToStatus, cfg.StatusReady)
	}
	if resp.Phases.Ready.Summary.Promoted != 1 {
		t.Errorf("ready summary promoted = %d, want 1", resp.Phases.Ready.Summary.Promoted)
	}
}

func TestReadyPhase_LabelMismatch(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = true
	cfg.PlannedLabel = "planned"
	items := []github.ProjectItem{
		{ID: "1", Title: "Plan Issue", URL: "https://github.com/owner/repo/issues/1", Status: "Plan", Labels: []string{"other-label"}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Phases.Ready.Results.Promoted) != 0 {
		t.Errorf("expected 0 ready promoted, got %d", len(resp.Phases.Ready.Results.Promoted))
	}
	// ラベルが一致しないアイテムは Skipped に記録される
	if resp.Phases.Ready.Summary.Promoted != 0 {
		t.Errorf("ready summary promoted = %d, want 0", resp.Phases.Ready.Summary.Promoted)
	}
	if len(resp.Phases.Ready.Results.Skipped) != 1 {
		t.Errorf("expected 1 ready skipped, got %d", len(resp.Phases.Ready.Results.Skipped))
	}
	if len(mp.updated) != 0 {
		t.Errorf("expected 0 UpdateItemStatus calls, got %d", len(mp.updated))
	}
}

func TestReadyPhase_CustomLabel(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = true
	cfg.PlannedLabel = "my-custom-label"
	items := []github.ProjectItem{
		{ID: "1", Title: "Plan Issue with custom", URL: "https://github.com/owner/repo/issues/1", Status: "Plan", Labels: []string{"my-custom-label"}},
		{ID: "2", Title: "Plan Issue without", URL: "https://github.com/owner/repo/issues/2", Status: "Plan", Labels: []string{"planned"}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	promoted := resp.Phases.Ready.Results.Promoted
	if len(promoted) != 1 {
		t.Fatalf("expected 1 ready promoted, got %d", len(promoted))
	}
	if promoted[0].Item.ID != "1" {
		t.Errorf("promoted item ID = %q, want %q", promoted[0].Item.ID, "1")
	}
}

func TestReadyPhase_DryRun(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = true
	cfg.PlannedLabel = "planned"
	cfg.DryRun = true
	items := []github.ProjectItem{
		{ID: "1", Title: "Plan Issue", URL: "https://github.com/owner/repo/issues/1", Status: "Plan", Labels: []string{"planned"}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// dry-run では UpdateItemStatus が呼ばれない
	if len(mp.updated) != 0 {
		t.Errorf("expected 0 UpdateItemStatus calls, got %d", len(mp.updated))
	}

	// promoted スライスにはアイテムが含まれる
	promoted := resp.Phases.Ready.Results.Promoted
	if len(promoted) != 1 {
		t.Fatalf("expected 1 ready promoted in output, got %d", len(promoted))
	}
}

func TestCascade_PlanToReadyToDoing(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = true
	cfg.PlannedLabel = "planned"
	// plan フェーズ: Backlog -> Plan
	// ready フェーズ: Plan + "planned" ラベル -> Ready
	// doing フェーズ: Ready -> In progress
	items := []github.ProjectItem{
		{
			ID:     "1",
			Title:  "Cascade Issue",
			URL:    "https://github.com/owner/repo-cascade/issues/1",
			Status: "Plan",
			Labels: []string{"planned"},
		},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ready フェーズで Plan -> Ready に昇格
	if resp.Phases.Ready.Summary.Promoted != 1 {
		t.Errorf("ready promoted = %d, want 1", resp.Phases.Ready.Summary.Promoted)
	}

	// doing フェーズで Ready -> In progress に昇格（カスケード）
	if resp.Phases.Doing.Summary.Promoted != 1 {
		t.Errorf("doing promoted = %d, want 1", resp.Phases.Doing.Summary.Promoted)
	}

	// summary の合算に ready フェーズが含まれる
	if resp.Summary.Promoted != 2 {
		t.Errorf("total promoted = %d, want 2", resp.Summary.Promoted)
	}
}

func TestRun_ReadyPhaseInSummary(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = true
	cfg.PlannedLabel = "planned"
	items := []github.ProjectItem{
		{ID: "1", Title: "Plan Issue 1", URL: "https://github.com/owner/repo-a/issues/1", Status: "Plan", Labels: []string{"planned"}},
		{ID: "2", Title: "Plan Issue 2", URL: "https://github.com/owner/repo-b/issues/2", Status: "Plan", Labels: []string{"planned"}},
		{ID: "3", Title: "Inbox Issue", URL: "https://github.com/owner/repo-c/issues/3", Status: "Backlog", Labels: []string{}},
	}

	resp, err := Run(context.Background(), cfg, items, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// plan: 1 (Backlog→Plan), ready: 2 (Plan→Ready), doing: 2 (Ready→Doing, cascade)
	if resp.Phases.Plan.Summary.Promoted != 1 {
		t.Errorf("plan promoted = %d, want 1", resp.Phases.Plan.Summary.Promoted)
	}
	if resp.Phases.Ready.Summary.Promoted != 2 {
		t.Errorf("ready promoted = %d, want 2", resp.Phases.Ready.Summary.Promoted)
	}
	if resp.Phases.Doing.Summary.Promoted != 2 {
		t.Errorf("doing promoted = %d, want 2", resp.Phases.Doing.Summary.Promoted)
	}
	// summary には全フェーズ合算
	if resp.Summary.Promoted != resp.Phases.Plan.Summary.Promoted+resp.Phases.Ready.Summary.Promoted+resp.Phases.Doing.Summary.Promoted {
		t.Errorf("summary promoted mismatch: got %d", resp.Summary.Promoted)
	}
}

func TestRun_ReadyFieldAlwaysPresent(t *testing.T) {
	mp := &mockPromoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.PromoteReadyEnabled = false // 機能無効

	resp, err := Run(context.Background(), cfg, []github.ProjectItem{}, mp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 機能無効時も phases.ready が存在すること（nil でないこと）
	if resp.Phases.Ready.Results.Promoted == nil {
		t.Error("phases.ready.results.promoted should not be nil even when disabled")
	}
	if resp.Phases.Ready.Results.Skipped == nil {
		t.Error("phases.ready.results.skipped should not be nil even when disabled")
	}
}
