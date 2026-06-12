package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/enough/enough/backend/agent"
	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/skills"
)

func runCuratorCLI() {
	if len(os.Args) < 3 {
		printCuratorUsage()
		os.Exit(1)
	}

	cmd := strings.ToLower(os.Args[2])
	args := os.Args[3:]

	switch cmd {
	case "status":
		cfg, err := config.LoadRuntime()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(skills.CuratorStatusString(cfg.Curator))
	case "run":
		dryRun := false
		sync := true
		for _, a := range args {
			switch strings.ToLower(a) {
			case "dry-run", "--dry-run":
				dryRun = true
			case "--background":
				sync = false
			case "--sync", "--synchronous":
				sync = true
			}
		}
		cfg, err := config.LoadRuntime()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		res := agent.RunCuratorReview(cfg, dryRun, sync, func(msg string) {
			fmt.Fprintf(os.Stderr, "note: %s\n", msg)
		})
		fmt.Printf("curator: started at %s\n", res.StartedAt.Format(time.RFC3339))
		if res.AutoSummary != "" {
			fmt.Printf("  auto: %s\n", res.AutoSummary)
		}
		if dryRun {
			fmt.Println("dry-run: no changes applied. Run `enough curator run` to apply.")
		} else if !sync {
			fmt.Println("LLM pass running in background — check `enough curator status` later.")
		}
	case "pause":
		skills.SetCuratorPaused(true)
		fmt.Println("curator: paused")
	case "resume":
		skills.SetCuratorPaused(false)
		fmt.Println("curator: resumed")
	case "pin":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: enough curator pin <skill>")
			os.Exit(1)
		}
		name := args[0]
		if !skills.IsAgentCreated(name) {
			fmt.Fprintf(os.Stderr, "curator: '%s' is bundled or hub-installed — only agent-created skills can be pinned\n", name)
			os.Exit(1)
		}
		skills.PinSkill(name)
		fmt.Printf("curator: pinned '%s'\n", name)
	case "unpin":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: enough curator unpin <skill>")
			os.Exit(1)
		}
		name := args[0]
		if !skills.IsAgentCreated(name) {
			fmt.Fprintf(os.Stderr, "curator: '%s' is not an agent-created skill\n", name)
			os.Exit(1)
		}
		skills.UnpinSkill(name)
		fmt.Printf("curator: unpinned '%s'\n", name)
	case "restore":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: enough curator restore <skill>")
			os.Exit(1)
		}
		ok, msg := skills.RestoreSkill(args[0])
		fmt.Printf("curator: %s\n", msg)
		if !ok {
			os.Exit(1)
		}
	case "archive":
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: enough curator archive <skill>")
			os.Exit(1)
		}
		name := args[0]
		um := skills.LoadUsage()
		if rec, ok := um[name]; ok && rec.Pinned {
			fmt.Fprintf(os.Stderr, "curator: '%s' is pinned — unpin first with `enough curator unpin %s`\n", name, name)
			os.Exit(1)
		}
		ok, msg := skills.ArchiveSkill(name)
		fmt.Printf("curator: %s\n", msg)
		if !ok {
			os.Exit(1)
		}
	case "list-archived":
		names := skills.ListArchivedSkillNames()
		if len(names) == 0 {
			fmt.Println("curator: no archived skills")
			return
		}
		for _, n := range names {
			fmt.Println(n)
		}
	default:
		fmt.Printf("Unknown curator command: %s\n", cmd)
		printCuratorUsage()
		os.Exit(1)
	}
}

func printCuratorUsage() {
	fmt.Println("Usage: enough curator <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  status                    Show curator state and agent-created skill stats")
	fmt.Println("  run [dry-run]             Run curator now (--background to return immediately)")
	fmt.Println("  pause | resume            Pause or resume the inactivity scheduler")
	fmt.Println("  pin <skill>               Pin an agent-created skill")
	fmt.Println("  unpin <skill>             Unpin a skill")
	fmt.Println("  archive <skill>           Manually archive a skill")
	fmt.Println("  restore <skill>           Restore from ~/.enough/skills/.archive/")
	fmt.Println("  list-archived             List archived skill names")
}
