package demote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// mockDemoter implements github.ItemPromoter for testing.
type mockDemoter struct {
	meta      *github.ProjectMeta
	metaErr   error
	updateErr error
	updated   []updateCall
}

type updateCall struct {
	ItemID     string
	StatusName string
}

func (m *mockDemoter) FetchProjectItems(_ context.Context, _ string, _ int) ([]github.ProjectItem, error) {
	return nil, nil
}

func (m *mockDemoter) FetchProjectMeta(_ context.Context, _ string, _ int) (*github.ProjectMeta, error) {
	return m.meta, m.metaErr
}

func (m *mockDemoter) UpdateItemStatus(_ context.Context, _ *github.ProjectMeta, itemID string, statusName string) error {
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
		Owner:          "testowner",
		ProjectNumber:  1,
		StatusInbox:    "Backlog",
		StatusReady:    "Ready",
		StatusDoing:    "In progress",
		StaleThreshold: 2 * time.Hour,
		DryRun:         false,
	}
}

// staleTime は Status 遷移から stale 閾値（2h）を超えた時刻を返す。
func staleTime() time.Time {
	return time.Now().Add(-3 * time.Hour) // Status 遷移から3時間経過 = stale
}

// freshTime は Status 遷移から stale 閾値（2h）以内の時刻を返す。
func freshTime() time.Time {
	return time.Now().Add(-1 * time.Hour) // Status 遷移から1時間経過 = fresh
}

// TestDoingPhase_StaleItemDemotedToReady: stale な doing アイテムが ready に降格
func TestDoingPhase_StaleItemDemotedToReady(t *testing.T) {
	md := &mockDemoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Stale Doing", URL: "https://github.com/owner/repo/issues/1", Status: "In progress", UpdatedAt: staleTime(), Labels: []string{}},
	}

	resp, err := Run(context.Background(), cfg, items, md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	demoted := resp.Phases.Doing.Results.Demoted
	if len(demoted) != 1 {
		t.Fatalf("expected 1 demoted, got %d", len(demoted))
	}
	if demoted[0].Item.ID != "1" {
		t.Errorf("demoted item ID = %q, want %q", demoted[0].Item.ID, "1")
	}
	if demoted[0].FromStatus != "In progress" {
		t.Errorf("FromStatus = %q, want %q", demoted[0].FromStatus, "In progress")
	}
	if demoted[0].ToStatus != "Ready" {
		t.Errorf("ToStatus = %q, want %q", demoted[0].ToStatus, "Ready")
	}
	if demoted[0].Key != "doing-owner-repo-1" {
		t.Errorf("Key = %q, want %q", demoted[0].Key, "doing-owner-repo-1")
	}
	if resp.Phases.Doing.Summary.Demoted != 1 {
		t.Errorf("doing summary demoted = %d, want 1", resp.Phases.Doing.Summary.Demoted)
	}
	if len(md.updated) != 1 {
		t.Fatalf("expected 1 UpdateItemStatus call, got %d", len(md.updated))
	}
	if md.updated[0].StatusName != "Ready" {
		t.Errorf("UpdateItemStatus called with %q, want %q", md.updated[0].StatusName, "Ready")
	}
}

// TestDoingPhase_FreshItemSkipped: 新しい doing アイテムはスキップ
func TestDoingPhase_FreshItemSkipped(t *testing.T) {
	md := &mockDemoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Fresh Doing", URL: "https://github.com/owner/repo/issues/1", Status: "In progress", UpdatedAt: freshTime(), Labels: []string{}},
	}

	resp, err := Run(context.Background(), cfg, items, md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	demoted := resp.Phases.Doing.Results.Demoted
	skipped := resp.Phases.Doing.Results.Skipped
	if len(demoted) != 0 {
		t.Fatalf("expected 0 demoted, got %d", len(demoted))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(skipped))
	}
	if len(md.updated) != 0 {
		t.Errorf("expected 0 UpdateItemStatus calls, got %d", len(md.updated))
	}
}

// TestDryRun_UpdateItemStatusNotCalled: dry-run では UpdateItemStatus が呼ばれない
func TestDryRun_UpdateItemStatusNotCalled(t *testing.T) {
	md := &mockDemoter{meta: defaultMeta}
	cfg := defaultCfg()
	cfg.DryRun = true
	items := []github.ProjectItem{
		{ID: "1", Title: "Stale Doing", URL: "https://github.com/owner/repo/issues/1", Status: "In progress", UpdatedAt: staleTime(), Labels: []string{}},
	}

	resp, err := Run(context.Background(), cfg, items, md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(md.updated) != 0 {
		t.Errorf("dry-run: expected 0 UpdateItemStatus calls, got %d", len(md.updated))
	}

	// dry-run でも demoted に記録される
	if len(resp.Phases.Doing.Results.Demoted) != 1 {
		t.Errorf("dry-run doing demoted = %d, want 1", len(resp.Phases.Doing.Results.Demoted))
	}

	// DryRun フィールドが true にセットされている
	if !resp.DryRun {
		t.Errorf("DryRun = false, want true")
	}
}

// TestRun_SummaryAggregation: 全体サマリの集計が正確
func TestRun_SummaryAggregation(t *testing.T) {
	md := &mockDemoter{meta: defaultMeta}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Stale Doing", URL: "https://github.com/owner/repo-a/issues/1", Status: "In progress", UpdatedAt: staleTime(), Labels: []string{}},
		{ID: "2", Title: "Fresh Doing", URL: "https://github.com/owner/repo-b/issues/2", Status: "In progress", UpdatedAt: freshTime(), Labels: []string{}},
	}

	resp, err := Run(context.Background(), cfg, items, md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Phases.Doing.Summary.Demoted != 1 {
		t.Errorf("doing demoted = %d, want 1", resp.Phases.Doing.Summary.Demoted)
	}
	if resp.Phases.Doing.Summary.Skipped != 1 {
		t.Errorf("doing skipped = %d, want 1", resp.Phases.Doing.Summary.Skipped)
	}
	if resp.Summary.Demoted != 1 {
		t.Errorf("total demoted = %d, want 1", resp.Summary.Demoted)
	}
	if resp.Summary.Skipped != 1 {
		t.Errorf("total skipped = %d, want 1", resp.Summary.Skipped)
	}
	if resp.Summary.Total != 2 {
		t.Errorf("total = %d, want 2", resp.Summary.Total)
	}
}

// TestRun_EmptyItems_ResultsNotNil: 空配列でも nil にならない
func TestRun_EmptyItems_ResultsNotNil(t *testing.T) {
	md := &mockDemoter{meta: defaultMeta}
	cfg := defaultCfg()

	resp, err := Run(context.Background(), cfg, []github.ProjectItem{}, md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Phases.Doing.Results.Demoted == nil {
		t.Error("doing demoted should not be nil")
	}
	if resp.Phases.Doing.Results.Skipped == nil {
		t.Error("doing skipped should not be nil")
	}
}

// TestRun_APIError: エラーハンドリング
func TestRun_APIError(t *testing.T) {
	md := &mockDemoter{
		meta:      defaultMeta,
		updateErr: errors.New("API error"),
	}
	cfg := defaultCfg()
	items := []github.ProjectItem{
		{ID: "1", Title: "Stale Doing", URL: "https://github.com/owner/repo/issues/1", Status: "In progress", UpdatedAt: staleTime(), Labels: []string{}},
	}

	_, err := Run(context.Background(), cfg, items, md)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// TestRun_FetchMetaError: FetchProjectMeta エラーハンドリング
func TestRun_FetchMetaError(t *testing.T) {
	md := &mockDemoter{
		metaErr: errors.New("meta error"),
	}
	cfg := defaultCfg()

	_, err := Run(context.Background(), cfg, nil, md)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
