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

func TestSupportsThinkingCodex(t *testing.T) {
	if !SupportsThinking("gpt-5.4") {
		t.Fatal("gpt-5.4 should support thinking variants")
	}
	levels := SupportedThinkingLevels("gpt-5.4")
	if len(levels) < 4 {
		t.Fatalf("expected multiple levels, got %v", levels)
	}
}

func TestApplyThinkingCodexResponses(t *testing.T) {
	req := &ChatRequest{Model: "gpt-5.4"}
	ApplyThinkingToRequest(req, ThinkingMedium, "gpt-5.4")
	r := reasoningFromChatRequest(*req)
	if r == nil || r.Effort != "medium" || r.Summary != "auto" {
		t.Fatalf("reasoning = %#v", r)
	}
}

func TestStepThinkingLevel(t *testing.T) {
	got := StepThinkingLevel(ThinkingOff, "gpt-5.4", 1)
	if got != ThinkingMinimal {
		t.Fatalf("step = %q", got)
	}
}
