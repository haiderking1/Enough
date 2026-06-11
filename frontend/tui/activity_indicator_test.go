package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func plainActivityLine(app *App) string {
	return ansi.Strip(app.renderAgentActivityLoader())
}

func TestAgentActivityShowsWorkingBeforeStream(t *testing.T) {
	app := &App{styles: NewStyles(), lastActivityWordIndex: -1}
	app.beginAgentActivity()

	line := plainActivityLine(app)
	if !strings.Contains(line, "Working") {
		t.Fatalf("expected connecting activity to show Working, got %q", line)
	}
	if strings.Contains(line, "Cocking") || strings.Contains(line, "Simmering") {
		t.Fatalf("connecting activity should not show streaming words, got %q", line)
	}
}

func TestAgentActivityShowsCockingWordWhenStreaming(t *testing.T) {
	app := &App{styles: NewStyles(), lastActivityWordIndex: -1}
	app.beginAgentActivity()
	app.onAssistantStreamStart()

	line := plainActivityLine(app)
	if !strings.Contains(line, "Cocking") {
		t.Fatalf("expected first streaming word, got %q", line)
	}
}

func TestAgentActivityNextTurnSkipsLastVisibleWord(t *testing.T) {
	app := &App{styles: NewStyles(), lastActivityWordIndex: -1}

	app.beginAgentActivity()
	app.onAssistantStreamStart()
	if got := app.activityLabel(); got != "Cocking" {
		t.Fatalf("expected first turn to show Cocking, got %q", got)
	}

	app.stopAgentActivity()
	app.beginAgentActivity()
	app.onAssistantStreamStart()

	if got := app.activityLabel(); got != "Simmering" {
		t.Fatalf("expected next turn to skip repeated Cocking and show Simmering, got %q", got)
	}
}

func TestAgentActivityToolRoundAdvancesWord(t *testing.T) {
	app := &App{styles: NewStyles(), lastActivityWordIndex: -1}
	app.beginAgentActivity()
	app.onAssistantStreamStart()
	if got := app.activityLabel(); got != "Cocking" {
		t.Fatalf("expected first segment Cocking, got %q", got)
	}

	app.onAssistantStreamStart()
	if got := app.activityLabel(); got != "Simmering" {
		t.Fatalf("expected second segment to advance to Simmering, got %q", got)
	}
}

func TestAgentActivityFullBreatheLoopAdvancesWord(t *testing.T) {
	app := &App{styles: NewStyles(), lastActivityWordIndex: -1}
	app.beginAgentActivity()
	app.onAssistantStreamStart()

	for i := 0; i < activityWordAdvanceTick; i++ {
		app.tickAgentActivity()
	}

	if got := app.activityLabel(); got != "Simmering" {
		t.Fatalf("expected full breathe loop to advance word, got %q", got)
	}
}
