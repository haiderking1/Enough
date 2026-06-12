package agent

import (
	"testing"

	"github.com/enough/enough/backend/opencode"
)

func TestModelContextWindowDeepSeekV4(t *testing.T) {
	if got := ModelContextWindow(opencode.ProviderOpenCode, "deepseek-v4-flash", 0); got != 1_000_000 {
		t.Fatalf("deepseek-v4-flash context = %d, want 1000000", got)
	}
	if got := ModelContextWindow(opencode.ProviderOpenCode, "deepseek-v4-pro", 0); got != 1_000_000 {
		t.Fatalf("deepseek-v4-pro context = %d, want 1000000", got)
	}
}

func TestModelContextWindowOverride(t *testing.T) {
	if got := ModelContextWindow(opencode.ProviderOpenCode, "deepseek-v4-flash", 512000); got != 512000 {
		t.Fatalf("override context = %d, want 512000", got)
	}
}

func TestModelContextWindowCodex(t *testing.T) {
	if got := ModelContextWindow(opencode.ProviderCodex, "gpt-5-codex", 0); got != 272_000 {
		t.Fatalf("gpt-5-codex context = %d, want 272000", got)
	}
	if got := ModelContextWindow(opencode.ProviderCodex, "gpt-5.3-codex-spark", 0); got != 128_000 {
		t.Fatalf("spark context = %d, want 128000", got)
	}
}
