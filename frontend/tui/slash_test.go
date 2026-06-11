package tui

import "testing"

func TestFilteredSlashCommandsWithoutAgent(t *testing.T) {
	app := &App{}
	cmds := app.filteredSlashCommands()
	if len(cmds) < len(slashCommands) {
		t.Fatalf("expected at least %d static slash commands, got %d", len(slashCommands), len(cmds))
	}
}
