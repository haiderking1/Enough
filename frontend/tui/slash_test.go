package tui

import (
	"strings"
	"testing"
)

func TestRenderSlashMenuLayout(t *testing.T) {
	app := &App{styles: NewStyles(), width: 100, mode: modeTask}
	app.editor = NewEditor(64)
	app.editor.SetValue("/")
	app.slashCursor = 0

	out := app.renderSlashMenu(100)
	if !strings.Contains(out, "→") {
		t.Fatalf("expected selection arrow in menu: %q", out)
	}
	if !strings.Contains(out, "(1/") {
		t.Fatalf("expected position counter in menu: %q", out)
	}
	if strings.Contains(out, "SlashMenu") {
		t.Fatal("unexpected raw style leak")
	}
	rows := strings.Split(out, "\n")
	visible := 0
	for _, row := range rows {
		if strings.Contains(row, "connect") || strings.Contains(row, "model") || strings.Contains(row, "plugins") {
			visible++
		}
	}
	if visible > slashMenuVisible {
		t.Fatalf("expected at most %d command rows visible, got %d", slashMenuVisible, visible)
	}
}

func TestSlashMenuViewport(t *testing.T) {
	start, end := slashMenuViewport(0, 22)
	if start != 0 || end != 5 {
		t.Fatalf("cursor 0: got %d..%d, want 0..5", start, end)
	}
	start, end = slashMenuViewport(4, 22)
	if start != 0 || end != 5 {
		t.Fatalf("cursor 4: got %d..%d, want 0..5", start, end)
	}
	start, end = slashMenuViewport(5, 22)
	if start != 1 || end != 6 {
		t.Fatalf("cursor 5: got %d..%d, want 1..6", start, end)
	}
	start, end = slashMenuViewport(21, 22)
	if start != 17 || end != 22 {
		t.Fatalf("cursor 21: got %d..%d, want 17..22", start, end)
	}
	start, end = slashMenuViewport(0, 3)
	if start != 0 || end != 3 {
		t.Fatalf("short list: got %d..%d, want 0..3", start, end)
	}
}

func TestFilteredSlashCommandsNoSkillsOnBareSlash(t *testing.T) {
	app := &App{}
	app.editor = NewEditor(64)
	app.editor.SetValue("/")
	cmds := app.filteredSlashCommands()
	for _, cmd := range cmds {
		if strings.HasPrefix(cmd.name, "skill:") {
			t.Fatalf("skill %q should not appear on bare /", cmd.name)
		}
	}
}

func TestFilteredSlashCommandsSkillsOnlyAfterSkillPrefix(t *testing.T) {
	app := &App{}
	app.editor = NewEditor(64)
	app.editor.SetValue("/skill:")
	cmds := app.filteredSlashCommands()
	for _, cmd := range cmds {
		if !strings.HasPrefix(cmd.name, "skill:") {
			t.Fatalf("expected only skill: entries after /skill:, got %q", cmd.name)
		}
	}
}
