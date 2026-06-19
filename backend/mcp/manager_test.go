package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/enough/enough/backend/config"
)

// TestHelperProcess is the helper process that acts as a mock MCP stdio server.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req struct {
			JSONRPC string `json:"jsonrpc"`
			ID      any    `json:"id"`
			Method  string `json:"method"`
		}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		switch req.Method {
		case "initialize":
			resp := map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]any{
						"tools": map[string]any{},
					},
					"serverInfo": map[string]any{
						"name":    "mock-server",
						"version": "1.0.0",
					},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Println(string(data))

		case "tools/list":
			resp := map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "echo",
							"description": "Echoes input back",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"msg": map[string]any{"type": "string"},
								},
							},
						},
						{
							"name":        "ignored_tool",
							"description": "This tool should be filtered out",
							"inputSchema": map[string]any{
								"type": "object",
							},
						},
					},
				},
			}
			data, _ := json.Marshal(resp)
			fmt.Println(string(data))

		case "tools/call":
			// Parse request params
			var callReq struct {
				Params struct {
					Name      string         `json:"name"`
					Arguments map[string]any `json:"arguments"`
				} `json:"params"`
			}
			_ = json.Unmarshal([]byte(line), &callReq)

			if callReq.Params.Name == "echo" {
				msg := callReq.Params.Arguments["msg"]
				if msg == "delay" {
					time.Sleep(2 * time.Second)
				}
				resp := map[string]any{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"result": map[string]any{
						"content": []map[string]any{
							{
								"type": "text",
								"text": fmt.Sprintf("echo: %v", msg),
							},
						},
						"isError": false,
					},
				}
				data, _ := json.Marshal(resp)
				fmt.Println(string(data))
			}
		}
	}
}

func helperCommand(t *testing.T) *exec.Cmd {
	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func TestConnectValidation(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	// Both command and url
	cfg1 := config.MCPServerConfig{
		Command: "node",
		URL:     "http://localhost:8080/mcp",
	}
	_, err := manager.Connect(context.Background(), "test1", cfg1)
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected mutually exclusive transport error, got: %v", err)
	}

	// Neither command nor url
	cfg2 := config.MCPServerConfig{}
	_, err = manager.Connect(context.Background(), "test2", cfg2)
	if err == nil || !strings.Contains(err.Error(), "no command or url") {
		t.Errorf("expected no command or url error, got: %v", err)
	}
}

func TestIncludeExcludeFilter(t *testing.T) {
	filter1 := &config.MCPServerToolsConfig{
		Include: []string{"echo"},
	}
	if !isToolAllowed("echo", filter1) {
		t.Error("expected echo to be allowed by include filter")
	}
	if isToolAllowed("ignored_tool", filter1) {
		t.Error("expected ignored_tool to be filtered out by include filter")
	}

	filter2 := &config.MCPServerToolsConfig{
		Exclude: []string{"ignored_tool"},
	}
	if isToolAllowed("ignored_tool", filter2) {
		t.Error("expected ignored_tool to be filtered out by exclude filter")
	}
	if !isToolAllowed("echo", filter2) {
		t.Error("expected echo to be allowed by exclude filter")
	}
}

func TestStdioMockMCPServerCall(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	cmd := helperCommand(t)
	cfg := config.MCPServerConfig{
		Command: cmd.Path,
		Args:    cmd.Args[1:],
		Env:     map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
		Tools: &config.MCPServerToolsConfig{
			Include: []string{"echo"},
		},
	}

	ctx := context.Background()
	session, err := manager.Connect(ctx, "mock", cfg)
	if err != nil {
		t.Fatalf("failed to connect to mock MCP server: %v", err)
	}

	if len(session.Tools()) != 1 {
		t.Errorf("expected 1 tool, got %d: %v", len(session.Tools()), session.Tools())
	}
	if session.Tools()[0].Function.Name != "mcp_mock_echo" {
		t.Errorf("expected tool name mcp_mock_echo, got: %s", session.Tools()[0].Function.Name)
	}

	manager.mu.Lock()
	manager.sessions["mock"] = session
	manager.toolMapping["mcp_mock_echo"] = ToolCallTarget{
		ServerName:       "mock",
		OriginalToolName: "echo",
	}
	manager.mu.Unlock()

	output, content, isErr, err := manager.CallTool(ctx, "mcp_mock_echo", `{"msg": "hello mock"}`)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if isErr {
		t.Error("expected no error in tool result")
	}
	if output.Text != "echo: hello mock" {
		t.Errorf("expected output 'echo: hello mock', got: %q", output.Text)
	}
	if len(content) != 1 || content[0].Text != "echo: hello mock" {
		t.Errorf("expected content to match output")
	}
}

func TestStdioMockMCPServerCancel(t *testing.T) {
	manager := NewManager()
	defer manager.Close()

	cmd := helperCommand(t)
	cfg := config.MCPServerConfig{
		Command: cmd.Path,
		Args:    cmd.Args[1:],
		Env:     map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
	}

	ctx := context.Background()
	session, err := manager.Connect(ctx, "mock", cfg)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	manager.mu.Lock()
	manager.sessions["mock"] = session
	manager.toolMapping["mcp_mock_echo"] = ToolCallTarget{
		ServerName:       "mock",
		OriginalToolName: "echo",
	}
	manager.mu.Unlock()

	cancelCtx, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	output, _, isErr, err := manager.CallTool(cancelCtx, "mcp_mock_echo", `{"msg": "delay"}`)
	if err != nil {
		t.Fatalf("expected no protocol error, got: %v", err)
	}
	if !isErr {
		t.Error("expected error flag to be true on interrupt")
	}
	if output.Text != "[interrupted]" {
		t.Errorf("expected output to be [interrupted], got: %q", output.Text)
	}
}
