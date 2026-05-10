package gitutil

import "testing"

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{
			name:      "https without .git",
			input:     "https://github.com/owner/repo",
			wantOwner: "owner",
			wantName:  "repo",
		},
		{
			name:      "https with .git suffix",
			input:     "https://github.com/owner/repo.git",
			wantOwner: "owner",
			wantName:  "repo",
		},
		{
			name:      "https with trailing newline",
			input:     "https://github.com/owner/repo.git\n",
			wantOwner: "owner",
			wantName:  "repo",
		},
		{
			name:      "scp-like ssh",
			input:     "git@github.com:owner/repo.git",
			wantOwner: "owner",
			wantName:  "repo",
		},
		{
			name:      "scp-like ssh without .git",
			input:     "git@github.com:owner/repo",
			wantOwner: "owner",
			wantName:  "repo",
		},
		{
			name:      "ssh:// form",
			input:     "ssh://git@github.com/owner/repo.git",
			wantOwner: "owner",
			wantName:  "repo",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			input:   "ftp://github.com/owner/repo",
			wantErr: true,
		},
		{
			name:    "missing owner/repo",
			input:   "https://github.com/",
			wantErr: true,
		},
		{
			name:    "owner only",
			input:   "https://github.com/owner",
			wantErr: true,
		},
		{
			name:      "extra path segments are ignored beyond owner/repo",
			input:     "https://github.com/owner/repo/extra",
			wantOwner: "owner",
			wantName:  "repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, name, err := ParseRemoteURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got owner=%q name=%q", owner, name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner || name != tt.wantName {
				t.Errorf("got (%q, %q), want (%q, %q)", owner, name, tt.wantOwner, tt.wantName)
			}
		})
	}
}
