package config

import (
	"fmt"
	"regexp"
	"strings"
)

var mcpServerNameRE = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*$`)

// ValidateMCPServerName checks a config key for mcp_servers.
func ValidateMCPServerName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name is required")
	}
	if !mcpServerNameRE.MatchString(name) {
		return fmt.Errorf("server name %q is invalid (use letters, digits, _ and -; must start with a letter)", name)
	}
	return nil
}

// ValidateMCPServerConfig ensures transport fields are usable by the MCP client.
func ValidateMCPServerConfig(cfg MCPServerConfig) error {
	hasCmd := strings.TrimSpace(cfg.Command) != ""
	hasURL := strings.TrimSpace(cfg.URL) != ""
	switch {
	case hasCmd && hasURL:
		return fmt.Errorf("choose either stdio (command) or remote (url), not both")
	case !hasCmd && !hasURL:
		return fmt.Errorf("command or url is required")
	}
	return nil
}

// AddMCPServer inserts or replaces an MCP server entry on cfg (not persisted).
func AddMCPServer(cfg *Config, name string, server MCPServerConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := ValidateMCPServerName(name); err != nil {
		return err
	}
	if err := ValidateMCPServerConfig(server); err != nil {
		return err
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]MCPServerConfig)
	}
	cfg.MCPServers[name] = server
	return nil
}

// RemoveMCPServer deletes an MCP server entry from cfg (not persisted).
func RemoveMCPServer(cfg *Config, name string) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if err := ValidateMCPServerName(name); err != nil {
		return err
	}
	if cfg.MCPServers == nil {
		return fmt.Errorf("MCP server %q is not configured", name)
	}
	if _, ok := cfg.MCPServers[name]; !ok {
		return fmt.Errorf("MCP server %q is not configured", name)
	}
	delete(cfg.MCPServers, name)
	if len(cfg.MCPServers) == 0 {
		cfg.MCPServers = make(map[string]MCPServerConfig)
	}
	return nil
}
