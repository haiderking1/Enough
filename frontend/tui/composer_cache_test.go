package tui

import "testing"

func TestComposerLinesCache(t *testing.T) {
	app := &App{
		styles: NewStyles(),
		width:  80,
		editor: NewEditor(512),
	}
	app.editor.Insert('h')
	app.editor.Insert('i')

	first := app.composerLines(80)
	if len(first) == 0 {
		t.Fatal("expected composer lines")
	}

	second := app.composerLines(80)
	if len(second) != len(first) {
		t.Fatalf("cache miss changed line count: %d vs %d", len(second), len(first))
	}

	app.editor.Insert('!')
	third := app.composerLines(80)
	if app.composerCache.value != app.editor.Value() {
		t.Fatal("composer cache should track editor value")
	}
	if len(third) == 0 {
		t.Fatal("expected composer lines after edit")
	}
}
