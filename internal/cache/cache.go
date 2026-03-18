package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/douhashi/gh-project-promoter/internal/github"
)

const (
	dirName  = "ghpp"
	fileName = "items.json"
)

// CacheData holds cached project items with a timestamp.
type CacheData struct {
	FetchedAt time.Time            `json:"fetched_at"`
	Items     []github.ProjectItem `json:"items"`
}

// Dir returns the cache directory path (~/.local/share/ghpp/).
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", dirName), nil
}

// Store writes the given items to the cache file as JSON.
func Store(items []github.ProjectItem) error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data := CacheData{
		FetchedAt: time.Now(),
		Items:     items,
	}

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Load reads the cache file and returns the cached data.
func Load() (*CacheData, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, fileName)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var data CacheData
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return &data, nil
}
