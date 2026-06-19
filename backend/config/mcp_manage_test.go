package config

import "testing"

func TestValidateMCPServerName(t *testing.T) {
	for _, name := range []string{"qmd", "fs", "my_server", "server-1"} {
		if err := ValidateMCPServerName(name); err != nil {
			t.Fatalf("expected %q valid, got %v", name, err)
		}
	}
	for _, name := range []string{"", "9bad", "bad:name", "bad name"} {
		if err := ValidateMCPServerName(name); err == nil {
			t.Fatalf("expected %q invalid", name)
		}
	}
}

func TestAddRemoveMCPServer(t *testing.T) {
	cfg := Default()
	if err := AddMCPServer(&cfg, "fs", MCPServerConfig{
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.MCPServers["fs"]; !ok {
		t.Fatal("expected fs entry")
	}
	if err := RemoveMCPServer(&cfg, "fs"); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.MCPServers["fs"]; ok {
		t.Fatal("expected fs removed")
	}
	if err := RemoveMCPServer(&cfg, "fs"); err == nil {
		t.Fatal("expected error removing missing server")
	}
}

func TestValidateMCPServerConfig(t *testing.T) {
	if err := ValidateMCPServerConfig(MCPServerConfig{Command: "npx"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMCPServerConfig(MCPServerConfig{URL: "http://localhost/mcp"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateMCPServerConfig(MCPServerConfig{}); err == nil {
		t.Fatal("expected error for empty transport")
	}
	if err := ValidateMCPServerConfig(MCPServerConfig{Command: "npx", URL: "http://x"}); err == nil {
		t.Fatal("expected error for dual transport")
	}
}
