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
		fmt.Println("gh-project-promoter")
		fmt.Println("Usage: gh-project-promoter <command>")
		fmt.Println("Commands: fetch")
		return
	}

	switch os.Args[1] {
	case "fetch":
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		client := github.NewClient(cfg.Token)
		if err := cmd.RunFetch(context.Background(), cfg, client); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
