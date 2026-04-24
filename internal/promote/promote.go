package promote

import (
	"context"
	"fmt"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
	"github.com/douhashi/gh-project-promoter/internal/urlutil"
)

// Run executes all promotion phases and returns a structured response.
func Run(ctx context.Context, cfg *config.Config, items []github.ProjectItem, promoter github.ItemPromoter) (*github.PromoteResponse, error) {
	meta, err := promoter.FetchProjectMeta(ctx, cfg.Owner, cfg.ProjectNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project meta: %w", err)
	}

	planResults, err := planPhase(ctx, cfg, items, meta, promoter)
	if err != nil {
		return nil, fmt.Errorf("plan phase failed: %w", err)
	}

	// カスケード昇格: planPhase で昇格したアイテムのステータスをインメモリで更新して次フェーズへ渡す
	itemsAfterPlan := applyPromotions(items, planResults.Promoted, cfg.StatusPlan)

	readyResults, err := readyPhase(ctx, cfg, itemsAfterPlan, meta, promoter)
	if err != nil {
		return nil, fmt.Errorf("ready phase failed: %w", err)
	}

	// カスケード昇格: readyPhase で昇格したアイテムのステータスをインメモリで更新して次フェーズへ渡す
	itemsAfterReady := applyPromotions(itemsAfterPlan, readyResults.Promoted, cfg.StatusReady)

	// itemsAfterReady には ready フェーズで昇格済みのアイテムも Ready ステータスとして含まれており、doing フェーズでカスケード昇格される
	doingResults, err := doingPhase(ctx, cfg, itemsAfterReady, meta, promoter)
	if err != nil {
		return nil, fmt.Errorf("doing phase failed: %w", err)
	}

	planPhaseResult := buildPhaseResult(planResults)
	readyPhaseResult := buildPhaseResult(readyResults)
	doingPhaseResult := buildPhaseResult(doingResults)

	return &github.PromoteResponse{
		DryRun: cfg.DryRun,
		Summary: github.PhaseSummary{
			Promoted: planPhaseResult.Summary.Promoted + readyPhaseResult.Summary.Promoted + doingPhaseResult.Summary.Promoted,
			Skipped:  planPhaseResult.Summary.Skipped + readyPhaseResult.Summary.Skipped + doingPhaseResult.Summary.Skipped,
			Total:    planPhaseResult.Summary.Total + readyPhaseResult.Summary.Total + doingPhaseResult.Summary.Total,
		},
		Phases: github.PromotePhases{
			Plan:  planPhaseResult,
			Ready: readyPhaseResult,
			Doing: doingPhaseResult,
		},
	}, nil
}

// applyPromotions returns a new slice of items with the promoted items' statuses updated to newStatus.
func applyPromotions(items []github.ProjectItem, promoted []github.PromotedItem, newStatus string) []github.ProjectItem {
	if len(promoted) == 0 {
		return items
	}
	promotedIDs := make(map[string]bool, len(promoted))
	for _, p := range promoted {
		promotedIDs[p.Item.ID] = true
	}
	result := make([]github.ProjectItem, len(items))
	for i, item := range items {
		if promotedIDs[item.ID] {
			item.Status = newStatus
		}
		result[i] = item
	}
	return result
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

	// Plan カラムの現在の件数をカウントする（WIP 上限の基準）
	currentPlanCount := 0
	for _, item := range items {
		if item.Status == cfg.StatusPlan {
			currentPlanCount++
		}
	}

	promoted := 0

	for _, item := range items {
		if item.Status != cfg.StatusInbox {
			continue
		}

		if cfg.PlanLimit > 0 && (currentPlanCount+promoted) >= cfg.PlanLimit {
			results.Skipped = append(results.Skipped, github.SkippedItem{
				Item:   item,
				Reason: fmt.Sprintf("plan limit reached (currently %d/%d in Plan)", currentPlanCount, cfg.PlanLimit),
			})
			continue
		}

		if !cfg.DryRun {
			if err := promoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusPlan); err != nil {
				return results, fmt.Errorf("failed to promote item %s to %s: %w", item.ID, cfg.StatusPlan, err)
			}
		}

		results.Promoted = append(results.Promoted, github.PromotedItem{
			Item:     item,
			Key:      urlutil.ExtractKey(item.URL, "plan"),
			ToStatus: cfg.StatusPlan,
		})
		promoted++
	}

	return results, nil
}

func readyPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, promoter github.ItemPromoter) (github.PhaseResults, error) {
	var results github.PhaseResults

	if !cfg.PromoteReadyEnabled {
		return results, nil
	}

	for _, item := range items {
		if item.Status != cfg.StatusPlan {
			continue
		}
		if !hasLabel(item.Labels, cfg.PlannedLabel) {
			results.Skipped = append(results.Skipped, github.SkippedItem{
				Item:   item,
				Reason: "label not matched",
			})
			continue
		}

		if !cfg.DryRun {
			if err := promoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusReady); err != nil {
				return results, fmt.Errorf("failed to promote item %s to %s: %w", item.ID, cfg.StatusReady, err)
			}
		}

		results.Promoted = append(results.Promoted, github.PromotedItem{
			Item:     item,
			Key:      urlutil.ExtractKey(item.URL, "ready"),
			ToStatus: cfg.StatusReady,
		})
	}

	return results, nil
}

// hasLabel reports whether the label slice contains the target label.
func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}

func doingPhase(ctx context.Context, cfg *config.Config, items []github.ProjectItem, meta *github.ProjectMeta, promoter github.ItemPromoter) (github.PhaseResults, error) {
	var results github.PhaseResults

	// Build set of repos that already have a doing item.
	doingRepos := make(map[string]bool)
	for _, item := range items {
		if item.Status != cfg.StatusDoing {
			continue
		}
		repo := urlutil.ExtractRepo(item.URL)
		if repo != "" {
			doingRepos[repo] = true
		}
	}

	for _, item := range items {
		if item.Status != cfg.StatusReady {
			continue
		}

		repo := urlutil.ExtractRepo(item.URL)
		if doingRepos[repo] {
			results.Skipped = append(results.Skipped, github.SkippedItem{
				Item:   item,
				Reason: "repository already has doing issue",
			})
			continue
		}

		if !cfg.DryRun {
			if err := promoter.UpdateItemStatus(ctx, meta, item.ID, cfg.StatusDoing); err != nil {
				return results, fmt.Errorf("failed to promote item %s to %s: %w", item.ID, cfg.StatusDoing, err)
			}
		}

		results.Promoted = append(results.Promoted, github.PromotedItem{
			Item:     item,
			Key:      urlutil.ExtractKey(item.URL, "doing"),
			ToStatus: cfg.StatusDoing,
		})
		doingRepos[repo] = true
	}

	return results, nil
}
