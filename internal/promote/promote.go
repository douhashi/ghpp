package promote

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// Run executes both promotion phases and returns all results.
func Run(ctx context.Context, cfg *config.Config, items []github.ProjectItem, promoter github.ItemPromoter) ([]github.PromoteResult, error) {
	meta, err := promoter.FetchProjectMeta(ctx, cfg.Owner, cfg.ProjectNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project meta: %w", err)
	}

	planResults, err := planPhase(ctx, cfg, items, meta, promoter)
	if err != nil {
		return nil, fmt.Errorf("plan phase failed: %w", err)
	}

	doingResults, err := doingPhase(ctx, cfg, items, meta, promoter)
	if err != nil {
		return nil, fmt.Errorf("doing phase failed: %w", err)
	}

	return append(planResults, doingResults...), nil
}

func planPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, promoter github.ItemPromoter) ([]github.PromoteResult, error) {
	var results []github.PromoteResult
	promoted := 0

	for _, item := range items {
		if item.Status != cfg.StatusInbox {
			continue
		}

		if cfg.PlanLimit > 0 && promoted >= cfg.PlanLimit {
			results = append(results, github.PromoteResult{
				Item:   item,
				Action: "skipped",
				Reason: "plan limit reached",
			})
			continue
		}

		if err := promoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusPlan); err != nil {
			return nil, fmt.Errorf("failed to promote item %s to %s: %w", item.ID, cfg.StatusPlan, err)
		}

		results = append(results, github.PromoteResult{
			Item:     item,
			Action:   "promoted",
			ToStatus: cfg.StatusPlan,
		})
		promoted++
	}

	return results, nil
}

func doingPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, promoter github.ItemPromoter) ([]github.PromoteResult, error) {
	var results []github.PromoteResult

	// Build set of repos that already have a doing item.
	doingRepos := make(map[string]bool)
	for _, item := range items {
		if item.Status != cfg.StatusDoing {
			continue
		}
		repo := extractRepo(item.URL)
		if repo != "" {
			doingRepos[repo] = true
		}
	}

	for _, item := range items {
		if item.Status != cfg.StatusReady {
			continue
		}

		repo := extractRepo(item.URL)
		if doingRepos[repo] {
			results = append(results, github.PromoteResult{
				Item:   item,
				Action: "skipped",
				Reason: "repository already has doing issue",
			})
			continue
		}

		if err := promoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusDoing); err != nil {
			return nil, fmt.Errorf("failed to promote item %s to %s: %w", item.ID, cfg.StatusDoing, err)
		}

		results = append(results, github.PromoteResult{
			Item:     item,
			Action:   "promoted",
			ToStatus: cfg.StatusDoing,
		})
		doingRepos[repo] = true
	}

	return results, nil
}

// extractRepo extracts "owner/repo" from a GitHub issue/PR URL.
func extractRepo(rawURL string) string {
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
