package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
	"github.com/douhashi/gh-project-promoter/internal/projectinit"
)

// RunProjectInit executes the project init workflow and prints the JSON
// response to stdout.
func RunProjectInit(ctx context.Context, cfg *config.ProjectInitConfig, init github.ProjectInitializer) error {
	resp, err := projectinit.Run(ctx, cfg, init)
	if err != nil {
		return fmt.Errorf("failed to run project init: %w", err)
	}

	out, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	fmt.Println(string(out))
	return nil
}
