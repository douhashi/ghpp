package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/gh-project-promoter/internal/cache"
	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// mockPromoter implements github.ItemPromoter for testing.
type mockPromoter struct {
	meta      *github.ProjectMeta
	metaErr   error
	updateErr error
}

func (m *mockPromoter) FetchProjectMeta(_ context.Context, _ string, _ int) (*github.ProjectMeta, error) {
	return m.meta, m.metaErr
}

func (m *mockPromoter) UpdateItemStatus(_ context.Context, _ *github.ProjectMeta, _ string, _ string) error {
	return m.updateErr
}

func TestRunPromote(t *testing.T) {
	defaultMeta := &github.ProjectMeta{
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

	tests := []struct {
		name     string
		items    []github.ProjectItem
		promoter *mockPromoter
		wantErr  bool
	}{
		{
			name: "success with items",
			items: []github.ProjectItem{
				{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog"},
			},
			promoter: &mockPromoter{meta: defaultMeta},
			wantErr:  false,
		},
		{
			name:     "success with empty items",
			items:    []github.ProjectItem{},
			promoter: &mockPromoter{meta: defaultMeta},
			wantErr:  false,
		},
		{
			name: "promoter error",
			items: []github.ProjectItem{
				{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "Backlog"},
			},
			promoter: &mockPromoter{metaErr: errors.New("API error")},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			// Store items in cache
			if err := cache.Store(tt.items); err != nil {
				t.Fatalf("failed to store cache: %v", err)
			}

			cfg := &config.Config{
				Owner:         "testowner",
				ProjectNumber: 1,
				StatusInbox:   "Backlog",
				StatusPlan:    "Plan",
				StatusReady:   "Ready",
				StatusDoing:   "In progress",
			}

			err := RunPromote(context.Background(), cfg, tt.promoter)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunPromote_NoCacheFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := &config.Config{
		Owner:         "testowner",
		ProjectNumber: 1,
	}

	mp := &mockPromoter{
		meta: &github.ProjectMeta{
			ProjectID: "PVT_001",
			FieldID:   "PVTSSF_001",
			Options:   map[string]string{},
		},
	}

	err := RunPromote(context.Background(), cfg, mp)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
