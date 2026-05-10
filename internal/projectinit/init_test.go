package projectinit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// fakeInitializer is a configurable test double for github.ProjectInitializer
// that records call counts and returns canned values.
type fakeInitializer struct {
	// canned responses
	templateID    string
	templateIDErr error

	ownerID    string
	ownerIDErr error

	repoID    string
	repoIDErr error

	existing    *github.ExistingProject
	existingErr error

	newProject    *github.NewProject
	newProjectErr error

	linkErr error

	workflows    []github.Workflow
	workflowsErr error

	// call counters
	calls struct {
		getProjectID  int
		getOwnerID    int
		getRepoID     int
		findByTitle   int
		copyProject   int
		linkRepo      int
		listWorkflows int
	}
}

func (f *fakeInitializer) GetProjectIDByOwnerAndNumber(_ context.Context, _ string, _ int) (string, error) {
	f.calls.getProjectID++
	return f.templateID, f.templateIDErr
}

func (f *fakeInitializer) GetOwnerID(_ context.Context, _ string) (string, error) {
	f.calls.getOwnerID++
	return f.ownerID, f.ownerIDErr
}

func (f *fakeInitializer) GetRepositoryID(_ context.Context, _, _ string) (string, error) {
	f.calls.getRepoID++
	return f.repoID, f.repoIDErr
}

func (f *fakeInitializer) FindProjectByTitle(_ context.Context, _, _ string) (*github.ExistingProject, error) {
	f.calls.findByTitle++
	return f.existing, f.existingErr
}

func (f *fakeInitializer) CopyProjectV2(_ context.Context, _, _, _ string) (*github.NewProject, error) {
	f.calls.copyProject++
	return f.newProject, f.newProjectErr
}

func (f *fakeInitializer) LinkProjectV2ToRepository(_ context.Context, _, _ string) error {
	f.calls.linkRepo++
	return f.linkErr
}

func (f *fakeInitializer) ListProjectV2Workflows(_ context.Context, _ string) ([]github.Workflow, error) {
	f.calls.listWorkflows++
	return f.workflows, f.workflowsErr
}

func defaultCfg() *config.ProjectInitConfig {
	return &config.ProjectInitConfig{
		Token:          "tok",
		Title:          "My Project",
		Owner:          "myorg",
		RepoOwner:      "myorg",
		RepoName:       "myrepo",
		TemplateOwner:  "douhashi",
		TemplateNumber: 25,
	}
}

func happyInitializer() *fakeInitializer {
	return &fakeInitializer{
		templateID: "PVT_template",
		ownerID:    "U_owner",
		repoID:     "R_repo",
		existing:   nil,
		newProject: &github.NewProject{
			ID:     "PVT_new",
			Number: 99,
			URL:    "https://github.com/users/myorg/projects/99",
			Title:  "My Project",
		},
		workflows: []github.Workflow{
			{Name: "Item closed", Number: 1, Enabled: true},
			{Name: "Auto-add to project", Number: 2, Enabled: false},
		},
	}
}

func TestRun_Happy(t *testing.T) {
	f := happyInitializer()
	resp, err := Run(context.Background(), defaultCfg(), f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.DryRun {
		t.Errorf("DryRun = true, want false")
	}
	if resp.Project.ID != "PVT_new" || resp.Project.Number != 99 {
		t.Errorf("unexpected project: %+v", resp.Project)
	}
	if resp.LinkedRepository != "myorg/myrepo" {
		t.Errorf("LinkedRepository = %q, want %q", resp.LinkedRepository, "myorg/myrepo")
	}
	if len(resp.Workflows) != 2 {
		t.Errorf("got %d workflows, want 2", len(resp.Workflows))
	}
	if len(resp.ManualSetupNeeded) == 0 {
		t.Fatalf("ManualSetupNeeded must not be empty")
	}
	// V12: Auto-add hint must always be present.
	joined := strings.Join(resp.ManualSetupNeeded, "\n")
	if !strings.Contains(joined, "Auto-add to project") {
		t.Errorf("ManualSetupNeeded missing Auto-add hint: %v", resp.ManualSetupNeeded)
	}
	// Verify call counts (V10 baseline: real run hits all 6 surfaces).
	if f.calls.findByTitle != 1 {
		t.Errorf("findByTitle calls = %d, want 1", f.calls.findByTitle)
	}
	if f.calls.copyProject != 1 || f.calls.linkRepo != 1 || f.calls.listWorkflows != 1 {
		t.Errorf("expected one of each (copy/link/list), got copy=%d link=%d list=%d",
			f.calls.copyProject, f.calls.linkRepo, f.calls.listWorkflows)
	}
}

func TestRun_DryRunSkipsMutations(t *testing.T) {
	cfg := defaultCfg()
	cfg.DryRun = true
	f := happyInitializer()

	resp, err := Run(context.Background(), cfg, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// V10: copy/link/list must not be called.
	if f.calls.copyProject != 0 {
		t.Errorf("copyProject called %d times in dry-run", f.calls.copyProject)
	}
	if f.calls.linkRepo != 0 {
		t.Errorf("linkRepo called %d times in dry-run", f.calls.linkRepo)
	}
	if f.calls.listWorkflows != 0 {
		t.Errorf("listWorkflows called %d times in dry-run", f.calls.listWorkflows)
	}

	if !resp.DryRun {
		t.Errorf("DryRun = false, want true")
	}
	if resp.Project.Title != "My Project" {
		t.Errorf("Project.Title = %q, want %q", resp.Project.Title, "My Project")
	}
	if resp.Project.ID != "<dry-run>" {
		t.Errorf("Project.ID = %q, want %q", resp.Project.ID, "<dry-run>")
	}
	if resp.Workflows == nil {
		t.Errorf("Workflows must be non-nil empty slice for stable JSON")
	}
	if len(resp.ManualSetupNeeded) == 0 {
		t.Errorf("ManualSetupNeeded must include guidance even in dry-run")
	}
}

func TestRun_NameCollisionWithoutForce(t *testing.T) {
	f := happyInitializer()
	f.existing = &github.ExistingProject{
		ID:     "PVT_existing",
		Number: 7,
		URL:    "https://github.com/users/myorg/projects/7",
		Title:  "My Project",
	}

	_, err := Run(context.Background(), defaultCfg(), f)
	if err == nil {
		t.Fatal("expected error on name collision, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention already exists: %v", err)
	}
	if !strings.Contains(err.Error(), "#7") {
		t.Errorf("error should include existing project number: %v", err)
	}
	if f.calls.copyProject != 0 {
		t.Errorf("copyProject must not be called on collision")
	}
}

func TestRun_ForceSkipsCollisionCheck(t *testing.T) {
	cfg := defaultCfg()
	cfg.Force = true
	f := happyInitializer()
	// Even if existing returned, force skips the lookup entirely.
	f.existing = &github.ExistingProject{ID: "PVT_existing", Number: 7, URL: "", Title: "My Project"}

	_, err := Run(context.Background(), cfg, f)
	if err != nil {
		t.Fatalf("unexpected error with --force: %v", err)
	}
	if f.calls.findByTitle != 0 {
		t.Errorf("findByTitle should not be called when force=true (got %d)", f.calls.findByTitle)
	}
	if f.calls.copyProject != 1 {
		t.Errorf("copyProject calls = %d, want 1", f.calls.copyProject)
	}
}

func TestRun_TemplateNotFound(t *testing.T) {
	f := happyInitializer()
	f.templateID = ""
	f.templateIDErr = errors.New("not found")

	_, err := Run(context.Background(), defaultCfg(), f)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "template") {
		t.Errorf("error should mention template: %v", err)
	}
}

func TestRun_CopyFailsWithInsufficientScope(t *testing.T) {
	// V5: when the API hints at insufficient scopes, surface a helpful message.
	cases := []string{
		"INSUFFICIENT_SCOPES: 403 Resource not accessible by integration",
		// Real-world GitHub GraphQL error string (token without 'project' scope).
		"Your token has not been granted the required scopes to execute this query. The 'copyProjectV2' field requires one of the following scopes: ['project']",
	}
	for _, msg := range cases {
		f := happyInitializer()
		f.newProject = nil
		f.newProjectErr = errors.New(msg)

		_, err := Run(context.Background(), defaultCfg(), f)
		if err == nil {
			t.Fatalf("expected error, got nil (msg=%q)", msg)
		}
		if !strings.Contains(err.Error(), "project") || !strings.Contains(err.Error(), "scope") {
			t.Errorf("error should mention project scope (msg=%q): %v", msg, err)
		}
		if !strings.Contains(err.Error(), "https://github.com/settings/tokens") {
			t.Errorf("error should include token settings URL (msg=%q): %v", msg, err)
		}
	}
}

func TestRun_LinkFailsAfterCopy(t *testing.T) {
	f := happyInitializer()
	f.linkErr = errors.New("link failed")

	_, err := Run(context.Background(), defaultCfg(), f)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Must mention that the project was created so the user can recover.
	if !strings.Contains(err.Error(), "#99") || !strings.Contains(err.Error(), "link") {
		t.Errorf("error should reference created project number and link failure: %v", err)
	}
}

func TestRun_OrgFallbackForOwner(t *testing.T) {
	// V11: ownerID resolution failures bubble up; both user/org are tried inside
	// the GitHub client, so here we just assert that an owner lookup error
	// surfaces cleanly.
	f := happyInitializer()
	f.ownerID = ""
	f.ownerIDErr = errors.New("owner not found in user nor org")

	_, err := Run(context.Background(), defaultCfg(), f)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "owner") {
		t.Errorf("error should mention owner: %v", err)
	}
}

func TestRun_ManualSetupAlwaysPresent(t *testing.T) {
	// V12: Auto-add hint must be present in both real and dry-run flows.
	f := happyInitializer()
	resp, err := Run(context.Background(), defaultCfg(), f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hasHint := false
	for _, m := range resp.ManualSetupNeeded {
		if strings.Contains(m, "Auto-add") {
			hasHint = true
			break
		}
	}
	if !hasHint {
		t.Errorf("ManualSetupNeeded missing auto-add hint: %v", resp.ManualSetupNeeded)
	}
}
