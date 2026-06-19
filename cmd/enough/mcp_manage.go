package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/mcp"
)

func runAddCLI() {
	if len(os.Args) < 3 {
		printAddUsage()
		os.Exit(1)
	}
	switch os.Args[2] {
	case "mcp":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: enough add mcp <server-name>")
			os.Exit(1)
		}
		runAddMCP(strings.TrimSpace(os.Args[3]))
	default:
		printAddUsage()
		os.Exit(1)
	}
}

func runRemoveCLI() {
	if len(os.Args) < 3 {
		printRemoveUsage()
		os.Exit(1)
	}
	switch os.Args[2] {
	case "mcp":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: enough remove mcp <server-name>")
			os.Exit(1)
		}
		runRemoveMCP(strings.TrimSpace(os.Args[3]))
	default:
		printRemoveUsage()
		os.Exit(1)
	}
}

func runAddMCP(name string) {
	if err := config.ValidateMCPServerName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if _, exists := cfg.MCPServers[name]; exists {
		if !promptYesNo(os.Stdin, fmt.Sprintf("MCP server %q already exists. Replace it?", name), false) {
			fmt.Println("Cancelled.")
			return
		}
	}

	server, err := promptMCPServerConfig(os.Stdin, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := config.AddMCPServer(&cfg, name, server); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if promptYesNo(os.Stdin, "Test connection before saving?", true) {
		if err := testMCPServer(name, server); err != nil {
			fmt.Fprintf(os.Stderr, "Connection test failed: %v\n", err)
			if !promptYesNo(os.Stdin, "Save anyway?", false) {
				fmt.Println("Cancelled.")
				return
			}
		}
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	path, _ := config.Path()
	fmt.Printf("Added MCP server %q", name)
	if path != "" {
		fmt.Printf(" to %s", path)
	}
	fmt.Println(".")
	fmt.Println("Reload in TUI with /mcp reload, or send a new message in the desktop app.")
}

func runRemoveMCP(name string) {
	if err := config.ValidateMCPServerName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := config.RemoveMCPServer(&cfg, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	path, _ := config.Path()
	fmt.Printf("Removed MCP server %q", name)
	if path != "" {
		fmt.Printf(" from %s", path)
	}
	fmt.Println(".")
}

func promptMCPServerConfig(in *os.File, name string) (config.MCPServerConfig, error) {
	reader := bufio.NewReader(in)
	fmt.Printf("Adding MCP server %q\n", name)
	fmt.Println("Templates: 1) stdio  2) http  3) filesystem (npx)")
	choice := strings.ToLower(strings.TrimSpace(promptDefault(reader, "Choose template [1]", "1")))

	var server config.MCPServerConfig
	enabled := true
	server.Enabled = &enabled
	server.Timeout = 30
	server.ConnectTimeout = 45

	switch choice {
	case "3", "filesystem", "fs":
		root := promptDefault(reader, "Filesystem root path", "/tmp")
		server.Command = "npx"
		server.Args = []string{"-y", "@modelcontextprotocol/server-filesystem", root}
	case "2", "http", "remote":
		server.URL = strings.TrimSpace(promptRequired(reader, "HTTP/SSE URL"))
		if auth := strings.TrimSpace(promptDefault(reader, "Authorization header (optional)", "")); auth != "" {
			server.Headers = map[string]string{"Authorization": auth}
		}
	default:
		server.Command = strings.TrimSpace(promptRequired(reader, "Command"))
		if argsLine := strings.TrimSpace(promptDefault(reader, "Args (space-separated, optional)", "")); argsLine != "" {
			server.Args = strings.Fields(argsLine)
		}
		if cwd := strings.TrimSpace(promptDefault(reader, "Working directory (optional)", "")); cwd != "" {
			server.Cwd = cwd
		}
		for {
			line := strings.TrimSpace(promptDefault(reader, "Env KEY=value (empty to finish)", ""))
			if line == "" {
				break
			}
			key, val, ok := strings.Cut(line, "=")
			if !ok || strings.TrimSpace(key) == "" {
				fmt.Println("  Skipped — use KEY=value")
				continue
			}
			if server.Env == nil {
				server.Env = make(map[string]string)
			}
			server.Env[strings.TrimSpace(key)] = strings.TrimSpace(val)
		}
	}

	if err := config.ValidateMCPServerConfig(server); err != nil {
		return config.MCPServerConfig{}, err
	}
	return server, nil
}

func testMCPServer(name string, server config.MCPServerConfig) error {
	manager := mcp.NewManager()
	defer manager.Close()

	session, err := manager.Connect(context.Background(), name, server)
	if err != nil {
		return err
	}

	tools := session.Tools()
	fmt.Printf("Connected. Discovered %d tool(s).\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  - %s\n", tool.Function.Name)
	}
	return nil
}

func promptDefault(r *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return defaultVal
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptRequired(r *bufio.Reader, label string) string {
	for {
		val := strings.TrimSpace(promptDefault(r, label, ""))
		if val != "" {
			return val
		}
		fmt.Println("  Required.")
	}
}

func promptYesNo(in *os.File, label string, defaultYes bool) bool {
	def := "y/N"
	if defaultYes {
		def = "Y/n"
	}
	r := bufio.NewReader(in)
	for {
		ans := strings.ToLower(strings.TrimSpace(promptDefault(r, fmt.Sprintf("%s (%s)", label, def), "")))
		if ans == "" {
			return defaultYes
		}
		switch ans {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("  Enter y or n.")
		}
	}
}

func printAddUsage() {
	fmt.Println("Usage: enough add mcp <server-name>")
	fmt.Println("\nInteractively add an MCP server to ~/.enough/config.json.")
}

func printRemoveUsage() {
	fmt.Println("Usage: enough remove mcp <server-name>")
	fmt.Println("\nRemove an MCP server from ~/.enough/config.json.")
}
