package opencode

import "testing"

func TestApplyThinkingToRequestEnabled(t *testing.T) {
	req := &ChatRequest{Model: "deepseek-v4-flash"}
	ApplyThinkingToRequest(req, ThinkingHigh, "deepseek-v4-flash")

	if req.Thinking == nil || req.Thinking.Type != "enabled" {
		t.Fatalf("thinking should be enabled, got %+v", req.Thinking)
	}
	if req.ReasoningEffort != "high" {
		t.Fatalf("reasoning effort: %q", req.ReasoningEffort)
	}
}

func TestApplyThinkingToRequestDisabled(t *testing.T) {
	req := &ChatRequest{Model: "deepseek-v4-flash"}
	ApplyThinkingToRequest(req, ThinkingOff, "deepseek-v4-flash")

	if req.Thinking == nil || req.Thinking.Type != "disabled" {
		t.Fatalf("thinking should be disabled, got %+v", req.Thinking)
	}
}
