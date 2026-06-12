package opencode

import "testing"

func TestRepairToolMessagesAddsMissingToolReplies(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: StringContent("hi")},
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "call_a", Type: "function", Function: ToolCallFunction{Name: "read_file", Arguments: `{}`}},
			{ID: "call_b", Type: "function", Function: ToolCallFunction{Name: "bash", Arguments: `{}`}},
		}},
		{Role: "tool", ToolCallID: "call_a", Name: "read_file", Content: StringContent("ok")},
	}

	fixed := RepairToolMessages(msgs)
	if len(fixed) != 4 {
		t.Fatalf("got %d messages, want 4", len(fixed))
	}
	last := fixed[len(fixed)-1]
	if last.Role != "tool" || last.ToolCallID != "call_b" {
		t.Fatalf("expected stub for call_b, got %+v", last)
	}
	if ContentString(last) != toolIncompleteMsg {
		t.Fatalf("stub content = %q", ContentString(last))
	}
}

func TestRepairToolMessagesAssignsMissingToolCallIDs(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{
			{Type: "function", Function: ToolCallFunction{Name: "bash", Arguments: `{}`}},
		}},
	}

	fixed := RepairToolMessages(msgs)
	if len(fixed) != 2 {
		t.Fatalf("got %d messages, want 2", len(fixed))
	}
	if fixed[0].ToolCalls[0].ID == "" {
		t.Fatal("expected synthetic tool call id")
	}
	if fixed[1].ToolCallID != fixed[0].ToolCalls[0].ID {
		t.Fatalf("tool reply id mismatch: %q vs %q", fixed[1].ToolCallID, fixed[0].ToolCalls[0].ID)
	}
}

func TestStripResponseFieldsRemovesUsage(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: StringContent("hi")},
		{Role: "assistant", Content: StringContent("hello"), Usage: &Usage{Input: 10, Output: 5}},
	}
	out := StripResponseFields(msgs)
	if out[1].Usage != nil {
		t.Fatal("expected usage stripped from assistant message")
	}
	if msgs[1].Usage == nil {
		t.Fatal("StripResponseFields should not mutate input slice")
	}
}

func TestPrepareRequestMessagesStripsUsage(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: StringContent("hi")},
		{Role: "assistant", Content: StringContent("hello"), Usage: &Usage{Input: 10, Output: 5}},
	}
	out := PrepareRequestMessages(msgs, "deepseek-v4-flash")
	if out[1].Usage != nil {
		t.Fatal("expected usage stripped")
	}
}
