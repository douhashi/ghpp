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

// buildPhaseResult creates a PhaseResult from PhaseResults,
// ensuring the slices are never nil.
func buildPhaseResult(results github.PhaseResults) github.PhaseResult {
	if results.Promoted == nil {
		results.Promoted = make([]github.PromotedItem, 0)
	}
	if results.Skipped == nil {
		results.Skipped = make([]github.SkippedItem, 0)
	}
	return github.PhaseResult{
		Summary: github.PhaseSummary{
			Promoted: len(results.Promoted),
			Skipped:  len(results.Skipped),
			Total:    len(results.Promoted) + len(results.Skipped),
		},
		Results: results,
	}
}

func planPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, promoter github.ItemPromoter) (github.PhaseResults, error) {
	var results github.PhaseResults
	promoted := 0

	for _, item := range items {
		if item.Status != cfg.StatusInbox {
			continue
		}

		if cfg.PlanLimit > 0 && promoted >= cfg.PlanLimit {
			results.Skipped = append(results.Skipped, github.SkippedItem{
				Item:   item,
				Reason: "plan limit reached",
			})
			continue
		}

		if err := promoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusPlan); err != nil {
			return results, fmt.Errorf("failed to promote item %s to %s: %w", item.ID, cfg.StatusPlan, err)
		}

		results.Promoted = append(results.Promoted, github.PromotedItem{
			Item:     item,
			Key:      extractKey(item.URL, "plan"),
			ToStatus: cfg.StatusPlan,
		})
		promoted++
	}

	return results, nil
}

func doingPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, promoter github.ItemPromoter) (github.PhaseResults, error) {
	var results github.PhaseResults

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
			results.Skipped = append(results.Skipped, github.SkippedItem{
				Item:   item,
				Reason: "repository already has doing issue",
			})
			continue
		}

		if err := promoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusDoing); err != nil {
			return results, fmt.Errorf("failed to promote item %s to %s: %w", item.ID, cfg.StatusDoing, err)
		}

		results.Promoted = append(results.Promoted, github.PromotedItem{
			Item:     item,
			Key:      extractKey(item.URL, "doing"),
			ToStatus: cfg.StatusDoing,
		})
		doingRepos[repo] = true
	}

	return results, nil
}

// extractKey builds a key string "{phase}-{owner}-{repo}-{number}" from a GitHub URL and phase name.
func extractKey(rawURL string, phase string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) < 4 {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s-%s", phase, parts[0], parts[1], parts[3])
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
