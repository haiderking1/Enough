package tui

import (
	"strings"
	"testing"
)

// sampleMessages exercises every block type plus tool grouping.
func sampleMessages() []chatMsg {
	return []chatMsg{
		{role: "user", text: "hello there, can you help me?"},
		{role: "assistant", text: "Sure!", thinking: "considering the request"},
		{role: "tool", toolName: "read_file", toolArgs: `{"path":"a.go"}`, toolResult: "package a"},
		{role: "tool", toolName: "bash", toolArgs: `{"cmd":"ls"}`, toolResult: "a.go b.go"},
		{role: "assistant", text: "Here is the result."},
		{role: "system", text: "session restored"},
		{role: "error", text: "something failed"},
	}
}

// renderChatIncremental must produce byte-identical output to the monolithic
// renderChat path it replaced.
func TestIncrementalMatchesMonolithic(t *testing.T) {
	widths := []int{40, 80, 120}
	flags := []struct{ hide, expand bool }{{false, false}, {true, false}, {false, true}, {true, true}}

	for _, w := range widths {
		for _, f := range flags {
			a := &App{styles: NewStyles(), messages: sampleMessages(), hideThinking: f.hide, toolsExpanded: f.expand}

			got := strings.Join(a.renderChatIncremental(w), "\n")

			chat := renderChat(a.styles, a.messages, w, f.hide, f.expand)
			var want string
			if chat != "" {
				want = strings.Join(clampSplitLines(strings.Split(chat, "\n"), w), "\n")
			}

			if got != want {
				t.Fatalf("w=%d hide=%v expand=%v mismatch:\n--- got ---\n%s\n--- want ---\n%s", w, f.hide, f.expand, got, want)
			}
		}
	}
}

// During streaming only the last block changes; every earlier block keeps its
// fingerprint and is served from cache rather than re-rendered.
func TestIncrementalReusesUnchangedBlocks(t *testing.T) {
	a := &App{styles: NewStyles(), messages: sampleMessages()}
	a.renderChatIncremental(80)

	firstFps := append([]uint64(nil), a.chatBlocks.fps...)

	// A streaming delta grows only the last block.
	a.messages = append(a.messages, chatMsg{role: "assistant", text: "streaming"})
	a.renderChatIncremental(80)

	for i := range firstFps {
		if a.chatBlocks.fps[i] != firstFps[i] {
			t.Fatalf("block %d fingerprint changed unexpectedly (should have been reused)", i)
		}
	}
	if len(a.chatBlocks.fps) != len(firstFps)+1 {
		t.Fatalf("expected one new block, got %d total", len(a.chatBlocks.fps))
	}
}

func TestChatLinesCache(t *testing.T) {
	app := &App{
		styles: NewStyles(),
		width:  80,
	}
	for i := 0; i < 20; i++ {
		app.messages = append(app.messages, chatMsg{role: "user", text: "hello world"})
	}
	app.bumpChat()

	first := app.chatLines(80)
	if len(first) == 0 {
		t.Fatal("expected chat lines")
	}

	// Typing in the composer should not rebuild chat.
	before := app.chatRevision
	second := app.chatLines(80)
	if app.chatRevision != before {
		t.Fatal("chatLines should not bump revision")
	}
	if len(second) != len(first) {
		t.Fatalf("cache miss changed line count: %d vs %d", len(second), len(first))
	}

	app.appendMessage("user", "another")
	third := app.chatLines(80)
	if len(third) <= len(first) {
		t.Fatal("expected chat to grow after new message")
	}
}
