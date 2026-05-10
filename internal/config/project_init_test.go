package config

import (
	"strings"
	"testing"
)

// projectInitEnvKeys lists env keys touched by LoadProjectInitWithArgs.
var projectInitEnvKeys = []string{
	"GH_TOKEN",
	"GHPP_OWNER",
	"GHPP_TEMPLATE_OWNER",
	"GHPP_TEMPLATE_NUMBER",
}

func clearProjectInitEnv(t *testing.T) {
	t.Helper()
	for _, k := range projectInitEnvKeys {
		t.Setenv(k, "")
	}
}

func TestLoadProjectInitWithArgs_AllFlags(t *testing.T) {
	clearProjectInitEnv(t)
	cfg, err := LoadProjectInitWithArgs([]string{
		"--token", "tok",
		"--title", "My Project",
		"--owner", "myorg",
		"--repo", "myorg/myrepo",
		"--template-owner", "tplowner",
		"--template-number", "100",
		"--force",
		"--dry-run",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "tok" {
		t.Errorf("Token = %q", cfg.Token)
	}
	if cfg.Title != "My Project" {
		t.Errorf("Title = %q", cfg.Title)
	}
	if cfg.Owner != "myorg" || cfg.RepoOwner != "myorg" || cfg.RepoName != "myrepo" {
		t.Errorf("owner/repo wrong: owner=%q repoOwner=%q repoName=%q", cfg.Owner, cfg.RepoOwner, cfg.RepoName)
	}
	if cfg.TemplateOwner != "tplowner" || cfg.TemplateNumber != 100 {
		t.Errorf("template wrong: %s/%d", cfg.TemplateOwner, cfg.TemplateNumber)
	}
	if !cfg.Force || !cfg.DryRun {
		t.Errorf("force/dry-run flags not picked up: force=%v dry=%v", cfg.Force, cfg.DryRun)
	}
}

func TestLoadProjectInitWithArgs_TemplateDefaults(t *testing.T) {
	clearProjectInitEnv(t)
	cfg, err := LoadProjectInitWithArgs([]string{
		"--token", "tok",
		"--title", "X",
		"--repo", "o/r",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.TemplateOwner != "douhashi" || cfg.TemplateNumber != 25 {
		t.Errorf("default template wrong: %s/%d", cfg.TemplateOwner, cfg.TemplateNumber)
	}
}

func TestLoadProjectInitWithArgs_OwnerDefaultsToRepoOwner(t *testing.T) {
	clearProjectInitEnv(t)
	cfg, err := LoadProjectInitWithArgs([]string{
		"--token", "tok",
		"--title", "X",
		"--repo", "alice/proj",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Owner != "alice" {
		t.Errorf("Owner default = %q, want %q", cfg.Owner, "alice")
	}
}

func TestLoadProjectInitWithArgs_OwnerOverridesRepoOwner(t *testing.T) {
	clearProjectInitEnv(t)
	cfg, err := LoadProjectInitWithArgs([]string{
		"--token", "tok",
		"--title", "X",
		"--owner", "other-org",
		"--repo", "alice/proj",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Owner != "other-org" {
		t.Errorf("Owner = %q, want %q", cfg.Owner, "other-org")
	}
	if cfg.RepoOwner != "alice" || cfg.RepoName != "proj" {
		t.Errorf("repo split wrong: %s/%s", cfg.RepoOwner, cfg.RepoName)
	}
}

func TestLoadProjectInitWithArgs_EnvFallback(t *testing.T) {
	clearProjectInitEnv(t)
	t.Setenv("GH_TOKEN", "envtok")
	t.Setenv("GHPP_OWNER", "envowner")
	t.Setenv("GHPP_TEMPLATE_OWNER", "envtpl")
	t.Setenv("GHPP_TEMPLATE_NUMBER", "77")
	cfg, err := LoadProjectInitWithArgs([]string{
		"--title", "X",
		"--repo", "o/r",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Token != "envtok" {
		t.Errorf("Token = %q", cfg.Token)
	}
	if cfg.Owner != "envowner" {
		t.Errorf("Owner = %q", cfg.Owner)
	}
	if cfg.TemplateOwner != "envtpl" || cfg.TemplateNumber != 77 {
		t.Errorf("template env not picked: %s/%d", cfg.TemplateOwner, cfg.TemplateNumber)
	}
}

func TestLoadProjectInitWithArgs_MissingToken(t *testing.T) {
	clearProjectInitEnv(t)
	_, err := LoadProjectInitWithArgs([]string{"--title", "X", "--repo", "o/r"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "GH_TOKEN") {
		t.Errorf("error should mention GH_TOKEN: %v", err)
	}
}

func TestLoadProjectInitWithArgs_MissingTitle(t *testing.T) {
	clearProjectInitEnv(t)
	_, err := LoadProjectInitWithArgs([]string{"--token", "t", "--repo", "o/r"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "title") && !strings.Contains(err.Error(), "Title") {
		t.Errorf("error should mention title: %v", err)
	}
}

func TestLoadProjectInitWithArgs_BadRepoFormat(t *testing.T) {
	clearProjectInitEnv(t)
	_, err := LoadProjectInitWithArgs([]string{
		"--token", "t",
		"--title", "X",
		"--repo", "no-slash",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "repo") {
		t.Errorf("error should mention repo: %v", err)
	}
}

func TestLoadProjectInitWithArgs_BadTemplateNumber(t *testing.T) {
	clearProjectInitEnv(t)
	_, err := LoadProjectInitWithArgs([]string{
		"--token", "t",
		"--title", "X",
		"--repo", "o/r",
		"--template-number", "abc",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadProjectInitWithArgs_RepoMissingOutsideGitDir(t *testing.T) {
	// When --repo is omitted, gitutil tries `git config --get remote.origin.url`.
	// Inside this test process the working directory IS a git repo, so we can't
	// reliably test the failure here; instead we just ensure that the function
	// either succeeds with the discovered repo or returns a clear error.
	clearProjectInitEnv(t)
	cfg, err := LoadProjectInitWithArgs([]string{"--token", "t", "--title", "X"})
	if err != nil {
		// In a non-git dir we expect a clear, actionable error.
		if !strings.Contains(err.Error(), "--repo") && !strings.Contains(err.Error(), "repo") {
			t.Errorf("error should mention repo: %v", err)
		}
		return
	}
	// In a git dir, owner/name must be populated.
	if cfg.RepoOwner == "" || cfg.RepoName == "" {
		t.Errorf("repo not auto-detected: %+v", cfg)
	}
}
