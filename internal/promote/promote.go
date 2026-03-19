package promote

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// Run executes both promotion phases and returns a structured response.
func Run(ctx context.Context, cfg *config.Config, items []github.ProjectItem, promoter github.ItemPromoter) (*github.PromoteResponse, error) {
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

	planPhaseResult := buildPhaseResult(planResults)
	doingPhaseResult := buildPhaseResult(doingResults)

	return &github.PromoteResponse{
		Summary: github.PhaseSummary{
			Promoted: planPhaseResult.Summary.Promoted + doingPhaseResult.Summary.Promoted,
			Skipped:  planPhaseResult.Summary.Skipped + doingPhaseResult.Summary.Skipped,
			Total:    planPhaseResult.Summary.Total + doingPhaseResult.Summary.Total,
		},
		Phases: github.PromotePhases{
			Plan:  planPhaseResult,
			Doing: doingPhaseResult,
		},
	}, nil
}

// buildPhaseResult creates a PhaseResult from a slice of PromoteResult,
// ensuring the results slice is never nil.
func buildPhaseResult(results []github.PromoteResult) github.PhaseResult {
	if results == nil {
		results = make([]github.PromoteResult, 0)
	}
	promoted := 0
	skipped := 0
	for _, r := range results {
		switch r.Action {
		case "promoted":
			promoted++
		case "skipped":
			skipped++
		}
	}
	return github.PhaseResult{
		Summary: github.PhaseSummary{
			Promoted: promoted,
			Skipped:  skipped,
			Total:    len(results),
		},
		Results: results,
	}
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
