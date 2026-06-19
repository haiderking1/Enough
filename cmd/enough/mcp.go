package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/mcp"
)

func runMcpCLI() {
	if len(os.Args) < 3 {
		printMcpUsage()
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

		if len(cfg.MCPServers) == 0 {
			fmt.Println("No MCP servers configured.")
			return
		}

		fmt.Println("Configured MCP servers:")
		for name, server := range cfg.MCPServers {
			enabled := true
			if server.Enabled != nil {
				enabled = *server.Enabled
			}
			status := "enabled"
			if !enabled {
				status = "disabled"
			}

			if server.Command != "" {
				fmt.Printf("  - %s (%s, stdio): %s %v\n", name, status, server.Command, server.Args)
			} else if server.URL != "" {
				fmt.Printf("  - %s (%s, remote): %s\n", name, status, server.URL)
			} else {
				fmt.Printf("  - %s (%s, unconfigured)\n", name, status)
			}
		}

	case "test":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: enough mcp test <server-name>")
			os.Exit(1)
		}
		serverName := os.Args[3]

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		srvCfg, ok := cfg.MCPServers[serverName]
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: MCP server %q not found in config\n", serverName)
			os.Exit(1)
		}

		fmt.Printf("Testing connection to MCP server %q...\n", serverName)
		manager := mcp.NewManager()
		defer manager.Close()

		ctx := context.Background()
		session, err := manager.Connect(ctx, serverName, srvCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully connected! Discovered %d tools:\n", len(session.Tools()))
		for _, tool := range session.Tools() {
			fmt.Printf("  - %s: %s\n", tool.Function.Name, tool.Function.Description)
		}

	case "call":
		if len(os.Args) < 5 {
			fmt.Fprintln(os.Stderr, "Usage: enough mcp call <server-name.tool-name> '<args-json>'")
			os.Exit(1)
		}
		target := os.Args[3]
		argsJSON := os.Args[4]

		var serverName, toolName string
		var modelToolName string
		if idx := strings.Index(target, "."); idx != -1 {
			serverName = target[:idx]
			toolName = target[idx+1:]
			modelToolName = fmt.Sprintf("mcp_%s_%s", serverName, sanitizeCLIName(toolName))
		} else if strings.HasPrefix(target, "mcp_") {
			modelToolName = target
		} else {
			fmt.Fprintf(os.Stderr, "Error: Target must be in 'server.tool' or 'mcp_server_tool' format\n")
			os.Exit(1)
		}

		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		manager := mcp.NewManager()
		defer manager.Close()

		// Reload the configured servers
		_ = manager.Reload(context.Background(), cfg.MCPServers)

		fmt.Printf("Calling tool %s...\n", modelToolName)
		outputBlock, contentBlocks, isErr, err := manager.CallTool(context.Background(), modelToolName, argsJSON)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calling tool: %v\n", err)
			os.Exit(1)
		}

		if isErr {
			fmt.Fprintln(os.Stderr, "Tool execution returned error:")
		}
		fmt.Println(outputBlock.Text)
		for _, cb := range contentBlocks {
			if cb.Type == "image" {
				fmt.Printf("[Image content: %s]\n", cb.MIMEType)
			}
		}
		if isErr {
			os.Exit(1)
		}

	default:
		printMcpUsage()
		os.Exit(1)
	}
}

func sanitizeCLIName(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('_')
		}
	}
	return sb.String()
}

func printMcpUsage() {
	fmt.Println("Usage: enough mcp <command> [args]")
	fmt.Println("\nCommands:")
	fmt.Println("  list                           List all configured MCP servers")
	fmt.Println("  test <server-name>             Connect to server and list exposed tools")
	fmt.Println("  call <server.tool> '<json>'    Invoke a specific MCP tool with JSON arguments")
	fmt.Println("\nAlso:")
	fmt.Println("  enough add mcp <name>          Interactively add a server to config")
	fmt.Println("  enough remove mcp <name>       Remove a server from config")
}
