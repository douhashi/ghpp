package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// stubInitializer is a minimal github.ProjectInitializer for cmd-layer tests.
type stubInitializer struct {
	findErr      error
	templateErr  error
	ownerErr     error
	repoErr      error
	copyErr      error
	linkErr      error
	workflowsErr error
}

func (s *stubInitializer) GetProjectIDByOwnerAndNumber(_ context.Context, _ string, _ int) (string, error) {
	return "PVT_t", s.templateErr
}
func (s *stubInitializer) GetOwnerID(_ context.Context, _ string) (string, error) {
	return "U_o", s.ownerErr
}
func (s *stubInitializer) GetRepositoryID(_ context.Context, _, _ string) (string, error) {
	return "R_r", s.repoErr
}
func (s *stubInitializer) FindProjectByTitle(_ context.Context, _, _ string) (*github.ExistingProject, error) {
	return nil, s.findErr
}
func (s *stubInitializer) CopyProjectV2(_ context.Context, _, _, _ string) (*github.NewProject, error) {
	if s.copyErr != nil {
		return nil, s.copyErr
	}
	return &github.NewProject{ID: "PVT_n", Number: 5, URL: "https://example.com/5", Title: "T"}, nil
}
func (s *stubInitializer) LinkProjectV2ToRepository(_ context.Context, _, _ string) error {
	return s.linkErr
}
func (s *stubInitializer) ListProjectV2Workflows(_ context.Context, _ string) ([]github.Workflow, error) {
	if s.workflowsErr != nil {
		return nil, s.workflowsErr
	}
	return []github.Workflow{{Name: "x", Number: 1, Enabled: true}}, nil
}

func baseCfg() *config.ProjectInitConfig {
	return &config.ProjectInitConfig{
		Token:          "tok",
		Title:          "T",
		Owner:          "o",
		RepoOwner:      "o",
		RepoName:       "r",
		TemplateOwner:  "douhashi",
		TemplateNumber: 25,
	}
}

func TestRunProjectInit_Success(t *testing.T) {
	if err := RunProjectInit(context.Background(), baseCfg(), &stubInitializer{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProjectInit_DryRun(t *testing.T) {
	cfg := baseCfg()
	cfg.DryRun = true
	if err := RunProjectInit(context.Background(), cfg, &stubInitializer{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProjectInit_PropagatesError(t *testing.T) {
	if err := RunProjectInit(context.Background(), baseCfg(), &stubInitializer{templateErr: errors.New("boom")}); err == nil {
		t.Fatal("expected error, got nil")
	}
}
