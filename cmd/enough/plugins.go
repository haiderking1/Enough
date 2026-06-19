package main

import (
	"fmt"
	"os"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/plugins"
)

func runPluginsCLI() {
	if len(os.Args) < 3 {
		printPluginsUsage()
		os.Exit(1)
	}

	cmd := os.Args[2]
	switch cmd {
	case "list":
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dir := plugins.PluginsDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No plugins installed under", dir)
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading plugins directory: %v\n", err)
			os.Exit(1)
		}

		var installed []string
		for _, entry := range entries {
			if entry.IsDir() && plugins.IsValidNamespace(entry.Name()) {
				installed = append(installed, entry.Name())
			}
		}

		if len(installed) == 0 {
			fmt.Println("No plugins installed.")
			return
		}

		fmt.Println("Installed plugins:")
		disabledMap := make(map[string]bool)
		if cfg.Plugins != nil {
			for _, d := range cfg.Plugins.Disabled {
				disabledMap[d] = true
			}
		}

		for _, name := range installed {
			status := "enabled"
			if disabledMap[name] {
				status = "disabled"
			}
			fmt.Printf("  - %s (%s)\n", name, status)
		}

	case "enable":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: enough plugins enable <plugin-name>")
			os.Exit(1)
		}
		name := os.Args[3]

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if cfg.Plugins == nil {
			cfg.Plugins = &config.PluginsSettings{Disabled: []string{}}
		}

		found := false
		var newDisabled []string
		for _, d := range cfg.Plugins.Disabled {
			if d == name {
				found = true
			} else {
				newDisabled = append(newDisabled, d)
			}
		}

		if !found {
			fmt.Printf("Plugin %q is already enabled.\n", name)
			return
		}

		cfg.Plugins.Disabled = newDisabled
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Plugin %q enabled successfully.\n", name)

	case "disable":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: enough plugins disable <plugin-name>")
			os.Exit(1)
		}
		name := os.Args[3]

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if cfg.Plugins == nil {
			cfg.Plugins = &config.PluginsSettings{Disabled: []string{}}
		}

		alreadyDisabled := false
		for _, d := range cfg.Plugins.Disabled {
			if d == name {
				alreadyDisabled = true
				break
			}
		}

		if alreadyDisabled {
			fmt.Printf("Plugin %q is already disabled.\n", name)
			return
		}

		cfg.Plugins.Disabled = append(cfg.Plugins.Disabled, name)
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Plugin %q disabled successfully.\n", name)

	default:
		printPluginsUsage()
		os.Exit(1)
	}
}

func printPluginsUsage() {
	fmt.Println("Usage: enough plugins <command> [args]")
	fmt.Println("\nCommands:")
	fmt.Println("  list                           List all installed plugins and their status")
	fmt.Println("  enable <plugin-name>           Enable an installed plugin")
	fmt.Println("  disable <plugin-name>          Disable an installed plugin")
}
