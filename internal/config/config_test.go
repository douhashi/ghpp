package config

import (
	"testing"
)

// allEnvKeys lists all environment variable keys used by the config package.
var allEnvKeys = []string{
	"GH_TOKEN", "GHPP_OWNER", "GHPP_PROJECT_NUMBER",
	"GHPP_STATUS_INBOX", "GHPP_STATUS_PLAN",
	"GHPP_STATUS_READY", "GHPP_STATUS_DOING",
	"GHPP_PLAN_LIMIT",
	"GHPP_PROMOTE_READY_ENABLED", "GHPP_PLANNED_LABEL",
}

// clearEnv clears all config-related environment variables for test isolation.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range allEnvKeys {
		t.Setenv(key, "")
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "all environment variables set correctly",
			env: map[string]string{
				"GH_TOKEN":            "ghp_test_token",
				"GHPP_OWNER":          "my-org",
				"GHPP_PROJECT_NUMBER": "42",
				"GHPP_STATUS_INBOX":   "Inbox",
				"GHPP_STATUS_PLAN":    "Plan",
				"GHPP_STATUS_READY":   "Ready",
				"GHPP_STATUS_DOING":   "Doing",
				"GHPP_PLAN_LIMIT":     "5",
			},
			want: &Config{
				Token:         "ghp_test_token",
				Owner:         "my-org",
				ProjectNumber: 42,
				StatusInbox:   "Inbox",
				StatusPlan:    "Plan",
				StatusReady:   "Ready",
				StatusDoing:   "Doing",
				PlanLimit:     5,
				PlannedLabel:  DefaultPlannedLabel,
			},
		},
		{
			name: "only required variables set",
			env: map[string]string{
				"GH_TOKEN":            "ghp_token",
				"GHPP_OWNER":          "owner",
				"GHPP_PROJECT_NUMBER": "1",
			},
			want: &Config{
				Token:         "ghp_token",
				Owner:         "owner",
				ProjectNumber: 1,
				StatusInbox:   "Backlog",
				StatusPlan:    "Plan",
				StatusReady:   "Ready",
				StatusDoing:   "In progress",
				PlanLimit:     3,
				PlannedLabel:  DefaultPlannedLabel,
			},
		},
		{
			name:    "missing GH_TOKEN",
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "missing GHPP_OWNER",
			env: map[string]string{
				"GH_TOKEN": "ghp_token",
			},
			wantErr: true,
		},
		{
			name: "missing GHPP_PROJECT_NUMBER",
			env: map[string]string{
				"GH_TOKEN":   "ghp_token",
				"GHPP_OWNER": "owner",
			},
			wantErr: true,
		},
		{
			name: "GHPP_PROJECT_NUMBER is not a number",
			env: map[string]string{
				"GH_TOKEN":            "ghp_token",
				"GHPP_OWNER":          "owner",
				"GHPP_PROJECT_NUMBER": "abc",
			},
			wantErr: true,
		},
		{
			name: "GHPP_PLAN_LIMIT is not a number",
			env: map[string]string{
				"GH_TOKEN":            "ghp_token",
				"GHPP_OWNER":          "owner",
				"GHPP_PROJECT_NUMBER": "1",
				"GHPP_PLAN_LIMIT":     "xyz",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertConfig(t, got, tt.want)
		})
	}
}

func TestLoadWithArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "all parameters via flags only",
			args: []string{
				"--token", "flag_token",
				"--owner", "flag-org",
				"--project-number", "10",
				"--status-inbox", "FlagInbox",
				"--status-plan", "FlagPlan",
				"--status-ready", "FlagReady",
				"--status-doing", "FlagDoing",
				"--plan-limit", "7",
			},
			env: map[string]string{},
			want: &Config{
				Token:         "flag_token",
				Owner:         "flag-org",
				ProjectNumber: 10,
				StatusInbox:   "FlagInbox",
				StatusPlan:    "FlagPlan",
				StatusReady:   "FlagReady",
				StatusDoing:   "FlagDoing",
				PlanLimit:     7,
				PlannedLabel:  DefaultPlannedLabel,
			},
		},
		{
			name: "environment variables only (backward compatibility)",
			args: []string{},
			env: map[string]string{
				"GH_TOKEN":            "env_token",
				"GHPP_OWNER":          "env-org",
				"GHPP_PROJECT_NUMBER": "20",
				"GHPP_STATUS_INBOX":   "EnvInbox",
				"GHPP_STATUS_PLAN":    "EnvPlan",
				"GHPP_STATUS_READY":   "EnvReady",
				"GHPP_STATUS_DOING":   "EnvDoing",
				"GHPP_PLAN_LIMIT":     "4",
			},
			want: &Config{
				Token:         "env_token",
				Owner:         "env-org",
				ProjectNumber: 20,
				StatusInbox:   "EnvInbox",
				StatusPlan:    "EnvPlan",
				StatusReady:   "EnvReady",
				StatusDoing:   "EnvDoing",
				PlanLimit:     4,
				PlannedLabel:  DefaultPlannedLabel,
			},
		},
		{
			name: "flags override environment variables",
			args: []string{
				"--token", "flag_token",
				"--owner", "flag-org",
				"--project-number", "99",
				"--plan-limit", "10",
			},
			env: map[string]string{
				"GH_TOKEN":            "env_token",
				"GHPP_OWNER":          "env-org",
				"GHPP_PROJECT_NUMBER": "1",
				"GHPP_STATUS_INBOX":   "EnvInbox",
				"GHPP_STATUS_PLAN":    "EnvPlan",
				"GHPP_STATUS_READY":   "EnvReady",
				"GHPP_STATUS_DOING":   "EnvDoing",
				"GHPP_PLAN_LIMIT":     "2",
			},
			want: &Config{
				Token:         "flag_token",
				Owner:         "flag-org",
				ProjectNumber: 99,
				StatusInbox:   "EnvInbox",
				StatusPlan:    "EnvPlan",
				StatusReady:   "EnvReady",
				StatusDoing:   "EnvDoing",
				PlanLimit:     10,
				PlannedLabel:  DefaultPlannedLabel,
			},
		},
		{
			name:    "missing required parameters",
			args:    []string{},
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "invalid plan-limit flag",
			args: []string{
				"--token", "t",
				"--owner", "o",
				"--project-number", "1",
				"--plan-limit", "not-a-number",
			},
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "invalid project-number flag",
			args: []string{
				"--token", "t",
				"--owner", "o",
				"--project-number", "abc",
			},
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "defaults applied when no optional flags or env",
			args: []string{
				"--token", "t",
				"--owner", "o",
				"--project-number", "5",
			},
			env: map[string]string{},
			want: &Config{
				Token:         "t",
				Owner:         "o",
				ProjectNumber: 5,
				StatusInbox:   DefaultStatusInbox,
				StatusPlan:    DefaultStatusPlan,
				StatusReady:   DefaultStatusReady,
				StatusDoing:   DefaultStatusDoing,
				PlanLimit:     DefaultPlanLimit,
				PlannedLabel:  DefaultPlannedLabel,
			},
		},
		{
			name: "--dry-run flag",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
				"--dry-run",
			},
			env: map[string]string{},
			want: &Config{
				Token:          "tok",
				Owner:          "owner",
				ProjectNumber:  1,
				StatusInbox:    DefaultStatusInbox,
				StatusPlan:     DefaultStatusPlan,
				StatusReady:    DefaultStatusReady,
				StatusDoing:    DefaultStatusDoing,
				PlanLimit:      DefaultPlanLimit,
				StaleThreshold: DefaultStaleThreshold,
				DryRun:         true,
				PlannedLabel:   DefaultPlannedLabel,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := LoadWithArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertConfig(t, got, tt.want)
		})
	}
}

func assertConfig(t *testing.T, got, want *Config) {
	t.Helper()
	if got.Token != want.Token {
		t.Errorf("Token = %q, want %q", got.Token, want.Token)
	}
	if got.Owner != want.Owner {
		t.Errorf("Owner = %q, want %q", got.Owner, want.Owner)
	}
	if got.ProjectNumber != want.ProjectNumber {
		t.Errorf("ProjectNumber = %d, want %d", got.ProjectNumber, want.ProjectNumber)
	}
	if got.StatusInbox != want.StatusInbox {
		t.Errorf("StatusInbox = %q, want %q", got.StatusInbox, want.StatusInbox)
	}
	if got.StatusPlan != want.StatusPlan {
		t.Errorf("StatusPlan = %q, want %q", got.StatusPlan, want.StatusPlan)
	}
	if got.StatusReady != want.StatusReady {
		t.Errorf("StatusReady = %q, want %q", got.StatusReady, want.StatusReady)
	}
	if got.StatusDoing != want.StatusDoing {
		t.Errorf("StatusDoing = %q, want %q", got.StatusDoing, want.StatusDoing)
	}
	if got.PlanLimit != want.PlanLimit {
		t.Errorf("PlanLimit = %d, want %d", got.PlanLimit, want.PlanLimit)
	}
	if got.DryRun != want.DryRun {
		t.Errorf("DryRun = %v, want %v", got.DryRun, want.DryRun)
	}
	if got.PromoteReadyEnabled != want.PromoteReadyEnabled {
		t.Errorf("PromoteReadyEnabled = %v, want %v", got.PromoteReadyEnabled, want.PromoteReadyEnabled)
	}
	if got.PlannedLabel != want.PlannedLabel {
		t.Errorf("PlannedLabel = %q, want %q", got.PlannedLabel, want.PlannedLabel)
	}
}

func TestLoadWithArgs_PromoteReadyEnabled(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		env     map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "--promote-ready-enabled flag sets field to true",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
				"--promote-ready-enabled",
				"--planned-label", "planned",
			},
			env: map[string]string{},
			want: &Config{
				Token:               "tok",
				Owner:               "owner",
				ProjectNumber:       1,
				StatusInbox:         DefaultStatusInbox,
				StatusPlan:          DefaultStatusPlan,
				StatusReady:         DefaultStatusReady,
				StatusDoing:         DefaultStatusDoing,
				PlanLimit:           DefaultPlanLimit,
				StaleThreshold:      DefaultStaleThreshold,
				PromoteReadyEnabled: true,
				PlannedLabel:        "planned",
			},
		},
		{
			name: "GHPP_PROMOTE_READY_ENABLED env var sets field to true",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
			},
			env: map[string]string{
				"GHPP_PROMOTE_READY_ENABLED": "true",
				"GHPP_PLANNED_LABEL":         "my-label",
			},
			want: &Config{
				Token:               "tok",
				Owner:               "owner",
				ProjectNumber:       1,
				StatusInbox:         DefaultStatusInbox,
				StatusPlan:          DefaultStatusPlan,
				StatusReady:         DefaultStatusReady,
				StatusDoing:         DefaultStatusDoing,
				PlanLimit:           DefaultPlanLimit,
				StaleThreshold:      DefaultStaleThreshold,
				PromoteReadyEnabled: true,
				PlannedLabel:        "my-label",
			},
		},
		{
			name: "flag overrides env var for promote-ready-enabled",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
				"--promote-ready-enabled=false",
				"--planned-label", "flag-label",
			},
			env: map[string]string{
				"GHPP_PROMOTE_READY_ENABLED": "true",
				"GHPP_PLANNED_LABEL":         "env-label",
			},
			want: &Config{
				Token:               "tok",
				Owner:               "owner",
				ProjectNumber:       1,
				StatusInbox:         DefaultStatusInbox,
				StatusPlan:          DefaultStatusPlan,
				StatusReady:         DefaultStatusReady,
				StatusDoing:         DefaultStatusDoing,
				PlanLimit:           DefaultPlanLimit,
				StaleThreshold:      DefaultStaleThreshold,
				PromoteReadyEnabled: false,
				PlannedLabel:        "flag-label",
			},
		},
		{
			name: "default: promote-ready-enabled=false, planned-label=planned",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
			},
			env: map[string]string{},
			want: &Config{
				Token:               "tok",
				Owner:               "owner",
				ProjectNumber:       1,
				StatusInbox:         DefaultStatusInbox,
				StatusPlan:          DefaultStatusPlan,
				StatusReady:         DefaultStatusReady,
				StatusDoing:         DefaultStatusDoing,
				PlanLimit:           DefaultPlanLimit,
				StaleThreshold:      DefaultStaleThreshold,
				PromoteReadyEnabled: false,
				PlannedLabel:        DefaultPlannedLabel,
			},
		},
		{
			name: "promote-ready-enabled=true with empty planned-label flag returns error",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
				"--promote-ready-enabled",
				"--planned-label", "",
			},
			env:     map[string]string{},
			wantErr: true,
		},
		{
			name: "GHPP_PROMOTE_READY_ENABLED invalid value returns error",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
			},
			env: map[string]string{
				"GHPP_PROMOTE_READY_ENABLED": "notabool",
			},
			wantErr: true,
		},
		{
			name: "GHPP_PLANNED_LABEL overrides default",
			args: []string{
				"--token", "tok",
				"--owner", "owner",
				"--project-number", "1",
			},
			env: map[string]string{
				"GHPP_PLANNED_LABEL": "custom-label",
			},
			want: &Config{
				Token:               "tok",
				Owner:               "owner",
				ProjectNumber:       1,
				StatusInbox:         DefaultStatusInbox,
				StatusPlan:          DefaultStatusPlan,
				StatusReady:         DefaultStatusReady,
				StatusDoing:         DefaultStatusDoing,
				PlanLimit:           DefaultPlanLimit,
				StaleThreshold:      DefaultStaleThreshold,
				PromoteReadyEnabled: false,
				PlannedLabel:        "custom-label",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			got, err := LoadWithArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertConfig(t, got, tt.want)
		})
	}
}
