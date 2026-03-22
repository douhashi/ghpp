package demote

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// Run executes both demotion phases and returns a structured response.
func Run(ctx context.Context, cfg *config.Config, items []github.ProjectItem, demoter github.ItemPromoter) (*github.DemoteResponse, error) {
	meta, err := demoter.FetchProjectMeta(ctx, cfg.Owner, cfg.ProjectNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project meta: %w", err)
	}

	now := time.Now()

	doingResults, err := doingPhase(ctx, cfg, items, meta, demoter, now)
	if err != nil {
		return nil, fmt.Errorf("doing phase failed: %w", err)
	}

	planResults, err := planPhase(ctx, cfg, items, meta, demoter, now)
	if err != nil {
		return nil, fmt.Errorf("plan phase failed: %w", err)
	}

	doingPhaseResult := buildPhaseResult(doingResults)
	planPhaseResult := buildPhaseResult(planResults)

	return &github.DemoteResponse{
		DryRun: cfg.DryRun,
		Summary: github.DemoteSummary{
			Demoted: doingPhaseResult.Summary.Demoted + planPhaseResult.Summary.Demoted,
			Skipped: doingPhaseResult.Summary.Skipped + planPhaseResult.Summary.Skipped,
			Total:   doingPhaseResult.Summary.Total + planPhaseResult.Summary.Total,
		},
		Phases: github.DemotePhases{
			Doing: doingPhaseResult,
			Plan:  planPhaseResult,
		},
	}, nil
}

// buildPhaseResult creates a DemotePhaseResult from DemotePhaseResults,
// ensuring the slices are never nil.
func buildPhaseResult(results github.DemotePhaseResults) github.DemotePhaseResult {
	if results.Demoted == nil {
		results.Demoted = make([]github.DemotedItem, 0)
	}
	if results.Skipped == nil {
		results.Skipped = make([]github.SkippedItem, 0)
	}
	return github.DemotePhaseResult{
		Summary: github.DemoteSummary{
			Demoted: len(results.Demoted),
			Skipped: len(results.Skipped),
			Total:   len(results.Demoted) + len(results.Skipped),
		},
		Results: results,
	}
}

func doingPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, demoter github.ItemPromoter, now time.Time) (github.DemotePhaseResults, error) {
	var results github.DemotePhaseResults

	for _, item := range items {
		if item.Status != cfg.StatusDoing {
			continue
		}

		if now.Sub(item.UpdatedAt) < cfg.StaleThreshold {
			results.Skipped = append(results.Skipped, github.SkippedItem{
				Item:   item,
				Reason: "not stale",
			})
			continue
		}

		if !cfg.DryRun {
			if err := demoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusReady); err != nil {
				return results, fmt.Errorf("failed to demote item %s to %s: %w", item.ID, cfg.StatusReady, err)
			}
		}

		results.Demoted = append(results.Demoted, github.DemotedItem{
			Item:       item,
			Key:        extractKey(item.URL, "doing"),
			FromStatus: cfg.StatusDoing,
			ToStatus:   cfg.StatusReady,
		})
	}

	return results, nil
}

func planPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, demoter github.ItemPromoter, now time.Time) (github.DemotePhaseResults, error) {
	var results github.DemotePhaseResults

	for _, item := range items {
		if item.Status != cfg.StatusPlan {
			continue
		}

		if now.Sub(item.UpdatedAt) < cfg.StaleThreshold {
			results.Skipped = append(results.Skipped, github.SkippedItem{
				Item:   item,
				Reason: "not stale",
			})
			continue
		}

		if !cfg.DryRun {
			if err := demoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusInbox); err != nil {
				return results, fmt.Errorf("failed to demote item %s to %s: %w", item.ID, cfg.StatusInbox, err)
			}
		}

		results.Demoted = append(results.Demoted, github.DemotedItem{
			Item:       item,
			Key:        extractKey(item.URL, "plan"),
			FromStatus: cfg.StatusPlan,
			ToStatus:   cfg.StatusInbox,
		})
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
