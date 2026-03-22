package urlutil_test

import (
	"testing"

	"github.com/douhashi/gh-project-promoter/internal/urlutil"
)

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
		{
			name:  "long owner truncated to 5 chars",
			url:   "https://github.com/longownername/repo/issues/1",
			phase: "plan",
			want:  "plan-longo-repo-1",
		},
		{
			name:  "long repository truncated to 10 chars",
			url:   "https://github.com/owner/longreponame123/issues/42",
			phase: "doing",
			want:  "doing-owner-longrepona-42",
		},
		{
			name:  "both owner and repository truncated",
			url:   "https://github.com/longownername/verylongreponame/issues/999",
			phase: "doing",
			want:  "doing-longo-verylongre-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urlutil.ExtractKey(tt.url, tt.phase)
			if got != tt.want {
				t.Errorf("ExtractKey(%q, %q) = %q, want %q", tt.url, tt.phase, got, tt.want)
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
			got := urlutil.ExtractRepo(tt.url)
			if got != tt.want {
				t.Errorf("ExtractRepo(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
