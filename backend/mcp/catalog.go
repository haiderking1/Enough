package mcp

import (
	"fmt"
	"strings"

	"github.com/enough/enough/backend/config"
)

// CatalogKind groups installable extensions; more kinds (skills, etc.) later.
type CatalogKind string

const (
	CatalogKindMCP CatalogKind = "mcp"
)

// CatalogSecret describes optional user input when installing a catalog entry.
type CatalogSecret struct {
	Key      string // header or env key
	Label    string
	Optional bool
}

// CatalogEntry is an installable integration shown in the TUI /plugins picker.
type CatalogEntry struct {
	ID          string
	Kind        CatalogKind
	Name        string
	Description string
	ServerName  string
	Secrets     []CatalogSecret
	build       func(secrets map[string]string) config.MCPServerConfig
}

func mcpCatalogEntries() []CatalogEntry {
	return []CatalogEntry{
		{
			ID:          "mcp-exa",
			Kind:        CatalogKindMCP,
			Name:        "Exa",
			Description: "Web search MCP (remote, no API key required)",
			ServerName:  "exa",
			build: func(map[string]string) config.MCPServerConfig {
				enabled := true
				return config.MCPServerConfig{
					URL:            "https://mcp.exa.ai/mcp",
					Enabled:        &enabled,
					Timeout:        30,
					ConnectTimeout: 45,
				}
			},
		},
		{
			ID:          "mcp-context7",
			Kind:        CatalogKindMCP,
			Name:        "Context7",
			Description: "Up-to-date library docs for LLM prompts",
			ServerName:  "context7",
			Secrets: []CatalogSecret{{
				Key:      "CONTEXT7_API_KEY",
				Label:    "Context7 API key",
				Optional: true,
			}},
			build: func(secrets map[string]string) config.MCPServerConfig {
				enabled := true
				cfg := config.MCPServerConfig{
					URL:            "https://mcp.context7.com/mcp",
					Enabled:        &enabled,
					Timeout:        30,
					ConnectTimeout: 45,
				}
				if key := strings.TrimSpace(secrets["CONTEXT7_API_KEY"]); key != "" {
					cfg.Headers = map[string]string{"CONTEXT7_API_KEY": key}
				}
				return cfg
			},
		},
	}
}

// Catalog returns installable MCP integrations for the /plugins picker.
func Catalog() []CatalogEntry {
	return mcpCatalogEntries()
}

// CatalogEntryByID finds a catalog entry.
func CatalogEntryByID(id string) (CatalogEntry, bool) {
	for _, e := range Catalog() {
		if e.ID == id {
			return e, true
		}
	}
	return CatalogEntry{}, false
}

// IsCatalogInstalled reports whether the entry's MCP server is in config.
func IsCatalogInstalled(cfg config.Config, entry CatalogEntry) bool {
	if entry.ServerName == "" || cfg.MCPServers == nil {
		return false
	}
	srv, ok := cfg.MCPServers[entry.ServerName]
	if !ok {
		return false
	}
	if srv.Enabled != nil && !*srv.Enabled {
		return false
	}
	return true
}

// InstallCatalogEntry adds the MCP server for entry to cfg (not persisted).
func InstallCatalogEntry(cfg *config.Config, entry CatalogEntry, secrets map[string]string) error {
	if entry.Kind != CatalogKindMCP {
		return fmt.Errorf("unsupported catalog kind %q", entry.Kind)
	}
	if entry.build == nil {
		return fmt.Errorf("catalog entry %q has no installer", entry.ID)
	}
	server := entry.build(secrets)
	return config.AddMCPServer(cfg, entry.ServerName, server)
}

// RemoveCatalogEntry removes the MCP server for entry from cfg (not persisted).
func RemoveCatalogEntry(cfg *config.Config, entry CatalogEntry) error {
	if entry.ServerName == "" {
		return fmt.Errorf("catalog entry %q has no server name", entry.ID)
	}
	return config.RemoveMCPServer(cfg, entry.ServerName)
}
