// Package projectinit implements the business logic for `ghpp project init`:
// copying a template ProjectV2 into a target owner and linking it to a repo.
package projectinit

import (
	"context"
	"fmt"
	"strings"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// autoAddManualHint is always returned in ManualSetupNeeded because the
// "Auto-add to project" workflow is not transferred by copyProjectV2.
const autoAddManualHint = "Auto-add to project: this workflow is NOT carried over by copyProjectV2. Open the new project's Workflows page and enable 'Auto-add to project' for the linked repository."

// dryRunPlaceholder marks fields whose real values would only be known after
// the API mutations run. It is visually distinct so JSON consumers cannot
// confuse it with a real GraphQL node ID.
const dryRunPlaceholder = "<dry-run>"

// Run executes the project init workflow and returns the response that
// callers will marshal to JSON.
func Run(ctx context.Context, cfg *config.ProjectInitConfig, init github.ProjectInitializer) (*github.ProjectInitResponse, error) {
	repoFullName := cfg.RepoOwner + "/" + cfg.RepoName

	// 1. Pre-flight collision check (skippable with --force).
	if !cfg.Force {
		existing, err := init.FindProjectByTitle(ctx, cfg.Owner, cfg.Title)
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing project with title %q: %w", cfg.Title, err)
		}
		if existing != nil {
			return nil, fmt.Errorf("project with title %q already exists at %s (#%d). Use --force to create a new project with the same title", cfg.Title, existing.URL, existing.Number)
		}
	}

	// 2. Resolve template / owner / repository node IDs in parallel sequence.
	templateID, err := init.GetProjectIDByOwnerAndNumber(ctx, cfg.TemplateOwner, cfg.TemplateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to access template project (%s/%d): %w", cfg.TemplateOwner, cfg.TemplateNumber, err)
	}

	ownerID, err := init.GetOwnerID(ctx, cfg.Owner)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve owner %q: %w", cfg.Owner, err)
	}

	repoID, err := init.GetRepositoryID(ctx, cfg.RepoOwner, cfg.RepoName)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repository %s: %w", repoFullName, err)
	}

	// 3. Dry-run short-circuit: skip all mutations and list calls but keep
	//    the JSON shape stable so consumers can branch on `dry_run` only.
	if cfg.DryRun {
		return &github.ProjectInitResponse{
			DryRun: true,
			Project: github.NewProject{
				ID:     dryRunPlaceholder,
				Number: 0,
				URL:    "",
				Title:  cfg.Title,
			},
			LinkedRepository:  repoFullName,
			Workflows:         []github.Workflow{},
			ManualSetupNeeded: []string{autoAddManualHint},
		}, nil
	}

	// 4. Copy the template project.
	newProj, err := init.CopyProjectV2(ctx, templateID, ownerID, cfg.Title)
	if err != nil {
		return nil, augmentScopeError(fmt.Errorf("failed to copy template project: %w", err))
	}

	// 5. Link the new project to the target repository. If this fails, the
	//    project is already created, so we surface its URL/number for recovery.
	if err := init.LinkProjectV2ToRepository(ctx, newProj.ID, repoID); err != nil {
		return nil, fmt.Errorf("project created (#%d, %s) but failed to link repository %s: %w",
			newProj.Number, newProj.URL, repoFullName, err)
	}

	// 6. Fetch workflows so callers can verify that the expected automations
	//    came over with the copy.
	workflows, err := init.ListProjectV2Workflows(ctx, newProj.ID)
	if err != nil {
		return nil, fmt.Errorf("project created (#%d) and linked, but failed to fetch workflows: %w", newProj.Number, err)
	}

	return &github.ProjectInitResponse{
		DryRun:            false,
		Project:           *newProj,
		LinkedRepository:  repoFullName,
		Workflows:         workflows,
		ManualSetupNeeded: []string{autoAddManualHint},
	}, nil
}

// augmentScopeError detects GraphQL error strings that hint at missing OAuth
// scopes and rewrites them with a concrete remediation step.
func augmentScopeError(err error) error {
	msg := err.Error()
	low := strings.ToLower(msg)
	scopeHint := strings.Contains(low, "insufficient_scopes") ||
		strings.Contains(low, "required scope") ||
		strings.Contains(low, "403") ||
		strings.Contains(low, "not accessible")
	if scopeHint {
		return fmt.Errorf("%w. Token requires the 'project' (write) scope. Update your token at https://github.com/settings/tokens", err)
	}
	return err
}
