package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/douhashi/gh-project-promoter/internal/cache"
	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
	"github.com/douhashi/gh-project-promoter/internal/promote"
)

// RunPromote loads cached items, runs the promotion logic, and prints the results as JSON.
func RunPromote(ctx context.Context, cfg *config.Config, promoter github.ItemPromoter) error {
	data, err := cache.Load()
	if err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	results, err := promote.Run(ctx, cfg, data.Items, promoter)
	if err != nil {
		return fmt.Errorf("failed to run promote: %w", err)
	}

	out, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	fmt.Println(string(out))
	return nil
}
