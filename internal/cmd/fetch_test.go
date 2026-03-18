package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// mockFetcher implements github.ItemFetcher for testing.
type mockFetcher struct {
	items []github.ProjectItem
	err   error
}

func (m *mockFetcher) FetchProjectItems(_ context.Context, _ string, _ int) ([]github.ProjectItem, error) {
	return m.items, m.err
}

func TestRunFetch(t *testing.T) {
	tests := []struct {
		name    string
		fetcher *mockFetcher
		wantErr bool
	}{
		{
			name: "success",
			fetcher: &mockFetcher{
				items: []github.ProjectItem{
					{ID: "PVTI_001", Title: "Issue 1", URL: "https://example.com/1", Status: "Ready"},
					{ID: "PVTI_002", Title: "Issue 2", URL: "https://example.com/2", Status: "Backlog"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty items",
			fetcher: &mockFetcher{
				items: []github.ProjectItem{},
			},
			wantErr: false,
		},
		{
			name: "fetch error",
			fetcher: &mockFetcher{
				err: errors.New("API error"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use temp HOME for cache
			tmpDir := t.TempDir()
			t.Setenv("HOME", tmpDir)

			cfg := &config.Config{
				Owner:         "testowner",
				ProjectNumber: 1,
			}

			err := RunFetch(context.Background(), cfg, tt.fetcher)
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
