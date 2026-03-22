package urlutil

import (
	"fmt"
	"net/url"
	"strings"
)

// ExtractKey builds a key string "{phase}-{owner}-{repo}-{number}" from a GitHub URL and phase name.
// Owner is truncated to 5 characters and repository to 10 characters to keep the key within 32 characters.
func ExtractKey(rawURL string, phase string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 4 {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s-%s", phase, truncate(parts[0], 5), truncate(parts[1], 10), parts[3])
}

// truncate returns s unchanged if its length is within maxLen,
// otherwise returns the first maxLen bytes.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ExtractRepo extracts "owner/repo" from a GitHub issue/PR URL.
func ExtractRepo(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + "/" + parts[1]
}
