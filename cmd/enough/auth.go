package main

import (
	"context"
	"fmt"
	"os"

	"github.com/enough/enough/backend/auth"
	"github.com/enough/enough/backend/config"
)

func runAuthCLI() {
	if len(os.Args) < 3 {
		printAuthUsage()
		os.Exit(1)
	}
	switch os.Args[2] {
	case "add":
		if len(os.Args) < 4 {
			printAuthUsage()
			os.Exit(1)
		}
		switch os.Args[3] {
		case "openai-codex":
			runAuthAddCodex()
		default:
			fmt.Fprintf(os.Stderr, "Unknown provider: %s\n", os.Args[3])
			printAuthUsage()
			os.Exit(1)
		}
	default:
		printAuthUsage()
		os.Exit(1)
	}
}

func runAuthAddCodex() {
	ctx := context.Background()
	start, err := auth.StartCodexDeviceAuth(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Codex auth failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Open %s\nEnter code: %s\nWaiting for sign-in...\n", start.VerifyURL, start.UserCode)
	if err := auth.PollCodexDeviceAuth(ctx, start); err != nil {
		fmt.Fprintf(os.Stderr, "Codex auth failed: %v\n", err)
		os.Exit(1)
	}
	if err := config.EnableCodexProvider(); err != nil {
		fmt.Fprintf(os.Stderr, "Saved tokens but failed to update config: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OpenAI Codex connected.")
}

func printAuthUsage() {
	fmt.Println("Usage: enough auth add openai-codex")
	fmt.Println("\nAuthenticate with OpenAI Codex via browser OAuth (ChatGPT subscription).")
}
