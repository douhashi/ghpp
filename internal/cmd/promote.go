package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
	"github.com/douhashi/gh-project-promoter/internal/promote"
)

// RunPromote fetches project items via the API, runs the promotion logic, and prints the results as JSON.
func RunPromote(ctx context.Context, cfg *config.Config, promoter github.ItemPromoter) error {
	items, err := promoter.FetchProjectItems(ctx, cfg.Owner, cfg.ProjectNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch project items: %w", err)
	}

	resp, err := promote.Run(ctx, cfg, items, promoter)
	if err != nil {
		return fmt.Errorf("failed to run promote: %w", err)
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	fmt.Println(string(out))
	return nil
}
