package tui

import (
	"strings"
	"testing"
)

func TestRenderCompactionLoader(t *testing.T) {
	app := &App{styles: NewStyles()}
	app.setCompacting(true, "Compacting context...")

	line := app.renderCompactionLoader()
	if !strings.Contains(line, "Compacting context...") {
		t.Fatalf("expected label in loader, got %q", line)
	}
	if !strings.Contains(line, "escape to cancel") {
		t.Fatalf("expected cancel hint in loader, got %q", line)
	}
	if strings.Contains(line, "⠋") && strings.Contains(line, "⠏") {
		t.Fatal("expected single spinner frame")
	}

	app.setCompacting(false, "")
	if app.renderCompactionLoader() != "" {
		t.Fatal("expected empty loader when not compacting")
	}
}
