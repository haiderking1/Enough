package tui

import (
	"testing"

	"github.com/enough/enough/backend/core"
)

func TestHandleToolDeltaAppendsToPending(t *testing.T) {
	a := &App{styles: NewStyles()}
	a.handleToolStart(core.ToolCallEvent{ID: "t1", Name: "bash", Args: `{"command":"ls"}`})

	a.handleToolDelta(core.ToolCallEvent{ID: "t1", Result: "line1\n"})
	a.handleToolDelta(core.ToolCallEvent{ID: "t1", Result: "line2\n"})

	last := a.messages[len(a.messages)-1]
	if last.toolResult != "line1\nline2\n" {
		t.Fatalf("toolResult = %q", last.toolResult)
	}
	if !last.toolPending {
		t.Fatal("tool should still be pending while streaming")
	}

	a.handleToolResult(core.ToolCallEvent{ID: "t1", Result: "line1\nline2\n"})
	last = a.messages[len(a.messages)-1]
	if last.toolPending {
		t.Fatal("tool should not be pending after result")
	}
}
