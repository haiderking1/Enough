package mcp

import (
	"testing"

	"github.com/enough/enough/backend/config"
)

func TestCatalogEntries(t *testing.T) {
	entries := Catalog()
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 catalog entries, got %d", len(entries))
	}
}

func TestCatalogInstallRemoveExa(t *testing.T) {
	entry, ok := CatalogEntryByID("mcp-exa")
	if !ok {
		t.Fatal("missing exa catalog entry")
	}
	cfg := config.Default()
	if IsCatalogInstalled(cfg, entry) {
		t.Fatal("expected exa not installed")
	}
	if err := InstallCatalogEntry(&cfg, entry, nil); err != nil {
		t.Fatal(err)
	}
	if !IsCatalogInstalled(cfg, entry) {
		t.Fatal("expected exa installed")
	}
	if cfg.MCPServers["exa"].URL != "https://mcp.exa.ai/mcp" {
		t.Fatalf("unexpected exa url: %q", cfg.MCPServers["exa"].URL)
	}
	if err := RemoveCatalogEntry(&cfg, entry); err != nil {
		t.Fatal(err)
	}
	if IsCatalogInstalled(cfg, entry) {
		t.Fatal("expected exa removed")
	}
}

func TestCatalogContext7OptionalKey(t *testing.T) {
	entry, ok := CatalogEntryByID("mcp-context7")
	if !ok {
		t.Fatal("missing context7 catalog entry")
	}
	cfg := config.Default()
	if err := InstallCatalogEntry(&cfg, entry, nil); err != nil {
		t.Fatal(err)
	}
	if cfg.MCPServers["context7"].Headers != nil {
		t.Fatal("expected no headers without api key")
	}

	cfg = config.Default()
	if err := InstallCatalogEntry(&cfg, entry, map[string]string{"CONTEXT7_API_KEY": "test-key"}); err != nil {
		t.Fatal(err)
	}
	if cfg.MCPServers["context7"].Headers["CONTEXT7_API_KEY"] != "test-key" {
		t.Fatalf("unexpected headers: %+v", cfg.MCPServers["context7"].Headers)
	}
}
