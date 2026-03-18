package cmd

import (
	"context"
	"fmt"

	"github.com/douhashi/gh-project-promoter/internal/cache"
	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

// RunFetch fetches project items via the GitHub API and stores them in the local cache.
func RunFetch(ctx context.Context, cfg *config.Config, fetcher github.ItemFetcher) error {
	items, err := fetcher.FetchProjectItems(ctx, cfg.Owner, cfg.ProjectNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch project items: %w", err)
	}

	if err := cache.Store(items); err != nil {
		return fmt.Errorf("failed to store cache: %w", err)
	}

	fmt.Printf("Fetched %d items\n", len(items))
	return nil
}
