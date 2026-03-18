package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/douhashi/gh-project-promoter/internal/github"
)

func TestStoreAndLoad(t *testing.T) {
	// Override HOME to use a temp directory
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	items := []github.ProjectItem{
		{ID: "PVTI_001", Title: "Issue 1", URL: "https://example.com/1", Status: "Ready"},
		{ID: "PVTI_002", Title: "Issue 2", URL: "https://example.com/2", Status: "Backlog"},
	}

	if err := Store(items); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	// Verify file was created
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	path := filepath.Join(dir, fileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("cache file was not created")
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got.FetchedAt.IsZero() {
		t.Error("FetchedAt should not be zero")
	}

	if len(got.Items) != len(items) {
		t.Fatalf("got %d items, want %d", len(got.Items), len(items))
	}

	for i, item := range got.Items {
		if item.ID != items[i].ID {
			t.Errorf("items[%d].ID = %q, want %q", i, item.ID, items[i].ID)
		}
		if item.Title != items[i].Title {
			t.Errorf("items[%d].Title = %q, want %q", i, item.Title, items[i].Title)
		}
		if item.URL != items[i].URL {
			t.Errorf("items[%d].URL = %q, want %q", i, item.URL, items[i].URL)
		}
		if item.Status != items[i].Status {
			t.Errorf("items[%d].Status = %q, want %q", i, item.Status, items[i].Status)
		}
	}
}

func TestDirectoryAutoCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	items := []github.ProjectItem{
		{ID: "PVTI_001", Title: "Test", URL: "https://example.com/1", Status: "Todo"},
	}

	// Directory should not exist yet
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("cache directory should not exist before Store")
	}

	if err := Store(items); err != nil {
		t.Fatalf("Store() error: %v", err)
	}

	// Directory should now exist
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("cache directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("cache path should be a directory")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("Load() should return error for invalid JSON")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() should return error when file does not exist")
	}
}
