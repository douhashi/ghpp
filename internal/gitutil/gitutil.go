// Package gitutil provides helpers for parsing git remote URLs and detecting
// the current repository owner/name.
package gitutil

import (
	"fmt"
	"os/exec"
	"strings"
)

// ParseRemoteURL extracts owner and repository name from a git remote URL.
// Supported forms:
//   - https://github.com/owner/repo(.git)?
//   - git@github.com:owner/repo(.git)?
//   - ssh://git@github.com/owner/repo(.git)?
func ParseRemoteURL(rawURL string) (owner, name string, err error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return "", "", fmt.Errorf("empty remote URL")
	}

	// Strip protocol/user prefix down to "host[:/]owner/repo"
	var path string
	switch {
	case strings.HasPrefix(trimmed, "https://"):
		path = strings.TrimPrefix(trimmed, "https://")
	case strings.HasPrefix(trimmed, "http://"):
		path = strings.TrimPrefix(trimmed, "http://")
	case strings.HasPrefix(trimmed, "ssh://"):
		path = strings.TrimPrefix(trimmed, "ssh://")
		if at := strings.Index(path, "@"); at >= 0 {
			path = path[at+1:]
		}
	case strings.HasPrefix(trimmed, "git@"):
		// SCP-like syntax: git@host:owner/repo
		path = strings.TrimPrefix(trimmed, "git@")
	default:
		return "", "", fmt.Errorf("unsupported remote URL format: %q", rawURL)
	}

	// After this point `path` is "host[:/]owner/repo[.git]"
	// Normalize the host separator (':' for SCP-like) into '/'.
	path = strings.Replace(path, ":", "/", 1)

	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("could not extract owner/repo from URL: %q", rawURL)
	}
	owner = parts[1]
	name = strings.TrimSuffix(parts[2], ".git")
	if owner == "" || name == "" {
		return "", "", fmt.Errorf("could not extract owner/repo from URL: %q", rawURL)
	}
	return owner, name, nil
}

// GetCwdRepoFromGit reads `git config --get remote.origin.url` for the
// current working directory and returns the parsed owner/repo.
func GetCwdRepoFromGit() (owner, name string, err error) {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	out, runErr := cmd.Output()
	if runErr != nil {
		// Distinguish "git not installed" from "ran but failed".
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			return "", "", fmt.Errorf("git command not found: %w", lookErr)
		}
		return "", "", fmt.Errorf("failed to read git remote (not a git repository?): %w", runErr)
	}
	return ParseRemoteURL(string(out))
}
