package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

const (
	DefaultStatusInbox    = "Backlog"
	DefaultStatusPlan     = "Plan"
	DefaultStatusReady    = "Ready"
	DefaultStatusDoing    = "In progress"
	DefaultPlanLimit      = 3
	DefaultStaleThreshold = 2 * time.Hour
	DefaultPlannedLabel   = "planned"
)

// getEnvOrDefault returns the value of the environment variable named by the key,
// or the default value if the variable is not set or empty.
func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// Config holds application configuration loaded from environment variables.
type Config struct {
	Token               string
	Owner               string
	ProjectNumber       int
	StatusInbox         string
	StatusPlan          string
	StatusReady         string
	StatusDoing         string
	PlanLimit           int
	StaleThreshold      time.Duration
	DryRun              bool
	PromoteReadyEnabled bool
	PlannedLabel        string
}

// Load reads environment variables and returns a Config.
// Required variables: GH_TOKEN, GHPP_OWNER, GHPP_PROJECT_NUMBER.
func Load() (*Config, error) {
	return LoadWithArgs(nil)
}

// LoadWithArgs reads configuration from command-line flags and environment variables.
// Priority: flags > environment variables > default values.
// If args is nil, only environment variables and defaults are used.
func LoadWithArgs(args []string) (*Config, error) {
	fs := flag.NewFlagSet("ghpp", flag.ContinueOnError)

	token := fs.String("token", "", "GitHub API token (env: GH_TOKEN)")
	owner := fs.String("owner", "", "GitHub Organization / User name (env: GHPP_OWNER)")
	projectNumber := fs.String("project-number", "", "GitHub Projects number (env: GHPP_PROJECT_NUMBER)")
	statusInbox := fs.String("status-inbox", "", "Status name for inbox (env: GHPP_STATUS_INBOX)")
	statusPlan := fs.String("status-plan", "", "Status name for plan (env: GHPP_STATUS_PLAN)")
	statusReady := fs.String("status-ready", "", "Status name for ready (env: GHPP_STATUS_READY)")
	statusDoing := fs.String("status-doing", "", "Status name for doing (env: GHPP_STATUS_DOING)")
	planLimit := fs.String("plan-limit", "", "Plan promotion limit (env: GHPP_PLAN_LIMIT)")
	staleThreshold := fs.String("stale-threshold", "", "Stale threshold duration for demote (env: GHPP_STALE_THRESHOLD, default: 2h)")
	dryRun := fs.Bool("dry-run", false, "Dry-run mode: do not actually update items")
	promoteReadyEnabled := fs.Bool("promote-ready-enabled", false, "Enable plan→ready auto-promotion (env: GHPP_PROMOTE_READY_ENABLED, default: false)")
	plannedLabel := fs.String("planned-label", "", "Label name that triggers plan→ready promotion (env: GHPP_PLANNED_LABEL, default: planned)")

	if args != nil {
		if err := fs.Parse(args); err != nil {
			return nil, fmt.Errorf("failed to parse flags: %w", err)
		}
	}

	// Track which flags were explicitly set
	flagSet := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		flagSet[f.Name] = true
	})

	// Helper: resolve value with priority flag > env > default
	resolve := func(flagName, flagVal, envKey, defaultVal string) string {
		if flagSet[flagName] {
			return flagVal
		}
		return getEnvOrDefault(envKey, defaultVal)
	}

	// Resolve token
	resolvedToken := resolve("token", *token, "GH_TOKEN", "")
	if resolvedToken == "" {
		return nil, fmt.Errorf("failed to load config: GH_TOKEN is required (use --token flag or GH_TOKEN env)")
	}

	// Resolve owner
	resolvedOwner := resolve("owner", *owner, "GHPP_OWNER", "")
	if resolvedOwner == "" {
		return nil, fmt.Errorf("failed to load config: GHPP_OWNER is required (use --owner flag or GHPP_OWNER env)")
	}

	// Resolve project number
	resolvedProjectNumberStr := resolve("project-number", *projectNumber, "GHPP_PROJECT_NUMBER", "")
	if resolvedProjectNumberStr == "" {
		return nil, fmt.Errorf("failed to load config: GHPP_PROJECT_NUMBER is required (use --project-number flag or GHPP_PROJECT_NUMBER env)")
	}
	resolvedProjectNumber, err := strconv.Atoi(resolvedProjectNumberStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse GHPP_PROJECT_NUMBER: %w", err)
	}

	// Resolve optional string values
	resolvedStatusInbox := resolve("status-inbox", *statusInbox, "GHPP_STATUS_INBOX", DefaultStatusInbox)
	resolvedStatusPlan := resolve("status-plan", *statusPlan, "GHPP_STATUS_PLAN", DefaultStatusPlan)
	resolvedStatusReady := resolve("status-ready", *statusReady, "GHPP_STATUS_READY", DefaultStatusReady)
	resolvedStatusDoing := resolve("status-doing", *statusDoing, "GHPP_STATUS_DOING", DefaultStatusDoing)

	// Resolve plan limit
	resolvedPlanLimit := DefaultPlanLimit
	resolvedPlanLimitStr := resolve("plan-limit", *planLimit, "GHPP_PLAN_LIMIT", "")
	if resolvedPlanLimitStr != "" {
		resolvedPlanLimit, err = strconv.Atoi(resolvedPlanLimitStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GHPP_PLAN_LIMIT: %w", err)
		}
	}

	// Resolve stale threshold
	resolvedStaleThreshold := DefaultStaleThreshold
	resolvedStaleThresholdStr := resolve("stale-threshold", *staleThreshold, "GHPP_STALE_THRESHOLD", "")
	if resolvedStaleThresholdStr != "" {
		resolvedStaleThreshold, err = time.ParseDuration(resolvedStaleThresholdStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GHPP_STALE_THRESHOLD: %w", err)
		}
	}

	// Resolve dry-run (flag only, no env var)
	resolvedDryRun := *dryRun

	// Resolve promote-ready-enabled (flag > env > default false)
	resolvedPromoteReadyEnabled := *promoteReadyEnabled
	if !flagSet["promote-ready-enabled"] {
		envVal := os.Getenv("GHPP_PROMOTE_READY_ENABLED")
		if envVal != "" {
			resolvedPromoteReadyEnabled, err = strconv.ParseBool(envVal)
			if err != nil {
				return nil, fmt.Errorf("failed to parse GHPP_PROMOTE_READY_ENABLED: %w", err)
			}
		}
	}

	// Resolve planned-label (flag > env > default "planned")
	resolvedPlannedLabel := resolve("planned-label", *plannedLabel, "GHPP_PLANNED_LABEL", DefaultPlannedLabel)

	// Validate: promote-ready-enabled=true requires a non-empty planned-label
	if resolvedPromoteReadyEnabled && resolvedPlannedLabel == "" {
		return nil, fmt.Errorf("failed to load config: --planned-label (or GHPP_PLANNED_LABEL) must not be empty when --promote-ready-enabled is true")
	}

	return &Config{
		Token:               resolvedToken,
		Owner:               resolvedOwner,
		ProjectNumber:       resolvedProjectNumber,
		StatusInbox:         resolvedStatusInbox,
		StatusPlan:          resolvedStatusPlan,
		StatusReady:         resolvedStatusReady,
		StatusDoing:         resolvedStatusDoing,
		PlanLimit:           resolvedPlanLimit,
		StaleThreshold:      resolvedStaleThreshold,
		DryRun:              resolvedDryRun,
		PromoteReadyEnabled: resolvedPromoteReadyEnabled,
		PlannedLabel:        resolvedPlannedLabel,
	}, nil
}
