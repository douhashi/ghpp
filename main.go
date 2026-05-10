package main

import (
	"context"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	"github.com/douhashi/gh-project-promoter/internal/cmd"
	"github.com/douhashi/gh-project-promoter/internal/config"
	"github.com/douhashi/gh-project-promoter/internal/github"
)

func main() {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "failed to load .env: %v\n", err)
			os.Exit(1)
		}
	}

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "promote":
		runPromote(os.Args[2:])
	case "demote":
		runDemote(os.Args[2:])
	case "project":
		runProject(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("ghpp - GitHub Project Promoter")
	fmt.Println("Usage: ghpp <command> [flags]")
	fmt.Println("Commands: promote, demote, project init")
}

func runPromote(args []string) {
	cfg, err := config.LoadWithArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	client := github.NewClient(cfg.Token)
	if err := cmd.RunPromote(context.Background(), cfg, client); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runDemote(args []string) {
	cfg, err := config.LoadWithArgs(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	client := github.NewClient(cfg.Token)
	if err := cmd.RunDemote(context.Background(), cfg, client); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runProject(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: ghpp project <subcommand> [flags]")
		fmt.Fprintln(os.Stderr, "Subcommands: init")
		os.Exit(1)
	}
	switch args[0] {
	case "init":
		cfg, err := config.LoadProjectInitWithArgs(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		client := github.NewClient(cfg.Token)
		if err := cmd.RunProjectInit(context.Background(), cfg, client); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown project subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
