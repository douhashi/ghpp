package config

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/douhashi/gh-project-promoter/internal/gitutil"
)

const (
	// DefaultTemplateOwner is the default owner whose project is used as the
	// init template when --template-owner is not provided.
	DefaultTemplateOwner = "douhashi"
	// DefaultTemplateNumber is the project number used together with
	// DefaultTemplateOwner.
	DefaultTemplateNumber = 25
)

// ProjectInitConfig is the resolved configuration for `ghpp project init`.
type ProjectInitConfig struct {
	Token          string
	Title          string
	Owner          string // resolved (defaults to RepoOwner when --owner is not given)
	RepoOwner      string
	RepoName       string
	TemplateOwner  string
	TemplateNumber int
	Force          bool
	DryRun         bool
}

// LoadProjectInitWithArgs parses the flags for `ghpp project init`.
// Resolution priority: flag > env > default.
func LoadProjectInitWithArgs(args []string) (*ProjectInitConfig, error) {
	fs := flag.NewFlagSet("ghpp project init", flag.ContinueOnError)

	token := fs.String("token", "", "GitHub API token (env: GH_TOKEN)")
	title := fs.String("title", "", "Title for the new project (required)")
	owner := fs.String("owner", "", "Owner (user or org) under which to create the project (env: GHPP_OWNER, default: repo owner)")
	repo := fs.String("repo", "", "owner/name of the repository to link (default: detected from current git remote)")
	templateOwner := fs.String("template-owner", "", fmt.Sprintf("Template project owner (env: GHPP_TEMPLATE_OWNER, default: %s)", DefaultTemplateOwner))
	templateNumber := fs.String("template-number", "", fmt.Sprintf("Template project number (env: GHPP_TEMPLATE_NUMBER, default: %d)", DefaultTemplateNumber))
	force := fs.Bool("force", false, "Skip the same-title collision check")
	dryRun := fs.Bool("dry-run", false, "Resolve everything but skip copy/link calls")

	if args != nil {
		if err := fs.Parse(args); err != nil {
			return nil, fmt.Errorf("failed to parse flags: %w", err)
		}
	}

	flagSet := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { flagSet[f.Name] = true })
	resolve := func(flagName, flagVal, envKey, defaultVal string) string {
		if flagSet[flagName] {
			return flagVal
		}
		return getEnvOrDefault(envKey, defaultVal)
	}

	resolvedToken := resolve("token", *token, "GH_TOKEN", "")
	if resolvedToken == "" {
		return nil, fmt.Errorf("failed to load config: GH_TOKEN is required (use --token flag or GH_TOKEN env)")
	}

	if *title == "" {
		return nil, fmt.Errorf("failed to load config: --title is required")
	}

	repoOwner, repoName, err := resolveRepo(flagSet["repo"], *repo)
	if err != nil {
		return nil, err
	}

	resolvedOwner := resolve("owner", *owner, "GHPP_OWNER", "")
	if resolvedOwner == "" {
		resolvedOwner = repoOwner
	}

	resolvedTemplateOwner := resolve("template-owner", *templateOwner, "GHPP_TEMPLATE_OWNER", DefaultTemplateOwner)
	resolvedTemplateNumberStr := resolve("template-number", *templateNumber, "GHPP_TEMPLATE_NUMBER", strconv.Itoa(DefaultTemplateNumber))
	resolvedTemplateNumber, err := strconv.Atoi(resolvedTemplateNumberStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template-number %q: %w", resolvedTemplateNumberStr, err)
	}

	return &ProjectInitConfig{
		Token:          resolvedToken,
		Title:          *title,
		Owner:          resolvedOwner,
		RepoOwner:      repoOwner,
		RepoName:       repoName,
		TemplateOwner:  resolvedTemplateOwner,
		TemplateNumber: resolvedTemplateNumber,
		Force:          *force,
		DryRun:         *dryRun,
	}, nil
}

// resolveRepo returns (owner, name) from the --repo flag or, if absent, from
// the current working directory's git remote.
func resolveRepo(repoFlagSet bool, repoFlag string) (string, string, error) {
	if repoFlagSet && repoFlag != "" {
		parts := strings.Split(repoFlag, "/")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("--repo must be in 'owner/name' form, got %q", repoFlag)
		}
		return parts[0], parts[1], nil
	}

	owner, name, err := gitutil.GetCwdRepoFromGit()
	if err != nil {
		return "", "", fmt.Errorf("failed to detect repository from git: %w. Pass --repo owner/name explicitly", err)
	}
	return owner, name, nil
}
