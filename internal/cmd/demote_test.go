package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

func TestRunDemote(t *testing.T) {
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
		name    string
		demoter *mockPromoter
		wantErr bool
	}{
		{
			name: "success with items",
			demoter: &mockPromoter{
				items: []github.ProjectItem{
					// 24h経過 > StaleThreshold(1h) なので demote が実行される
					{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "In progress", UpdatedAt: time.Now().Add(-24 * time.Hour)},
				},
				meta: defaultMeta,
			},
			wantErr: false,
		},
		{
			name: "success with empty items",
			demoter: &mockPromoter{
				items: []github.ProjectItem{},
				meta:  defaultMeta,
			},
			wantErr: false,
		},
		{
			name: "FetchProjectMeta error",
			demoter: &mockPromoter{
				items:   []github.ProjectItem{{ID: "1", Title: "Issue 1", URL: "https://github.com/owner/repo/issues/1", Status: "In progress", UpdatedAt: time.Now().Add(-24 * time.Hour)}},
				metaErr: errors.New("API error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Owner:          "testowner",
				ProjectNumber:  1,
				StatusInbox:    "Backlog",
				StatusPlan:     "Plan",
				StatusReady:    "Ready",
				StatusDoing:    "In progress",
				StaleThreshold: time.Hour,
			}

			err := RunDemote(context.Background(), cfg, tt.demoter)
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

func TestRunDemote_FetchError(t *testing.T) {
	cfg := &config.Config{
		Owner:         "testowner",
		ProjectNumber: 1,
	}

	mp := &mockPromoter{
		itemsErr: errors.New("API error"),
	}

	err := RunDemote(context.Background(), cfg, mp)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
