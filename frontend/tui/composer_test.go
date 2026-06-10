package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderTaskInput(t *testing.T) {
	app := &App{
		styles: NewStyles(),
		editor: NewEditor(512),
	}

	// Case 1: Idle, empty input
	app.running = false
	app.mode = modeTask
	res := app.renderTaskInput()
	plain := ansi.Strip(res)
	if !strings.Contains(plain, "❯") {
		t.Fatalf("expected idle prompt to contain '❯', got: %q", plain)
	}
	if !strings.Contains(plain, "▎") {
		t.Fatalf("expected idle prompt to contain caret, got: %q", plain)
	}

	// Case 2: Running, empty input
	app.running = true
	res = app.renderTaskInput()
	plain = ansi.Strip(res)
	if !strings.Contains(plain, "esc interrupt") {
		t.Fatalf("expected running prompt to contain hint, got: %q", plain)
	}
	if !strings.Contains(plain, "▎") {
		t.Fatalf("expected running prompt to contain caret when empty, got: %q", plain)
	}

	// Case 3: Running, typed input with cursor at the end
	app.editor.SetValue("hello")
	app.editor.End()
	res = app.renderTaskInput()
	plain = ansi.Strip(res)
	if !strings.Contains(plain, "hello") {
		t.Fatalf("expected running prompt to contain typed text, got: %q", plain)
	}
	if strings.Contains(plain, "esc interrupt") {
		t.Fatalf("expected running prompt NOT to contain hint when typing, got: %q", plain)
	}
	if !strings.Contains(plain, "▎") {
		t.Fatalf("expected running prompt to contain caret, got: %q", plain)
	}
}
