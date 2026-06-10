package agent

import "testing"

func TestModelContextWindowDeepSeekV4(t *testing.T) {
	if got := ModelContextWindow("deepseek-v4-flash", 0); got != 1_000_000 {
		t.Fatalf("deepseek-v4-flash context = %d, want 1000000", got)
	}
	if got := ModelContextWindow("deepseek-v4-pro", 0); got != 1_000_000 {
		t.Fatalf("deepseek-v4-pro context = %d, want 1000000", got)
	}
}

func TestModelContextWindowOverride(t *testing.T) {
	if got := ModelContextWindow("deepseek-v4-flash", 512000); got != 512000 {
		t.Fatalf("override context = %d, want 512000", got)
	}
}
