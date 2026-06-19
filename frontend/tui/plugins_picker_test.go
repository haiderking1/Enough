package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/enough/enough/frontend/tui/term"
)

func TestPluginsSearchPlainLine(t *testing.T) {
	app := &App{}
	line := app.pluginsSearchPlainLine(30, "test", true)
	if term.VisibleWidth(line) != 30 {
		t.Errorf("expected width 30, got %d", term.VisibleWidth(line))
	}

	lineEmpty := app.pluginsSearchPlainLine(40, "", false)
	if term.VisibleWidth(lineEmpty) != 40 {
		t.Errorf("expected width 40, got %d", term.VisibleWidth(lineEmpty))
	}
}

func TestRenderFixedInputBoxClosed(t *testing.T) {
	lines := renderFixedInputBox(30, "⌕ Search...", lipgloss.Color("#2a2a34"))
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	top, mid, bot := lines[0], lines[1], lines[2]
	if !strings.HasPrefix(top, "╭") || !strings.HasSuffix(top, "╮") {
		t.Fatalf("bad top: %q", top)
	}
	if !strings.HasPrefix(bot, "╰") || !strings.HasSuffix(bot, "╯") {
		t.Fatalf("bad bottom: %q", bot)
	}
	if term.VisibleWidth(top) != term.VisibleWidth(mid) || term.VisibleWidth(top) != term.VisibleWidth(bot) {
		t.Fatalf("width mismatch top=%d mid=%d bot=%d", term.VisibleWidth(top), term.VisibleWidth(mid), term.VisibleWidth(bot))
	}
}

func TestPluginsPickerSearchSurvivesRenderPipeline(t *testing.T) {
	app := &App{styles: NewStyles(), width: 80}
	app.mode = modePluginsPicker
	app.pluginsPickerTab = int(pluginsTabMCP)

	picker := app.renderPluginsPicker(80)
	rows := strings.Split(picker, "\n")

	var top string
	for _, row := range rows {
		if strings.Contains(row, "╭") && strings.Contains(row, "╮") {
			top = row
			break
		}
	}
	if top == "" {
		t.Fatalf("search box top border not found in picker output:\n%s", picker)
	}
	if !strings.HasSuffix(strings.TrimSpace(top), "╮") {
		t.Fatalf("top border truncated: %q", top)
	}
	if term.VisibleWidth(top) > 80 {
		t.Fatalf("top border wider than terminal: %d", term.VisibleWidth(top))
	}
}
