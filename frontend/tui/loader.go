package tui

var brailleSpinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (a *App) setCompacting(active bool, label string) {
	a.compacting = active
	if active {
		a.compactionLabel = label
		a.compactionFrame = 0
		return
	}
	a.compactionLabel = ""
	a.compactionFrame = 0
}

func (a *App) renderCompactionLoader() string {
	if !a.compacting {
		return ""
	}

	label := a.compactionLabel
	if label == "" {
		label = "Compacting context..."
	}

	frame := a.compactionFrame % len(brailleSpinner)
	spinner := a.styles.CompactionSpinner.Render(brailleSpinner[frame])
	text := a.styles.CompactionText.Render(label)
	hint := a.styles.LogDim.Render("(escape to cancel)")

	return spinner + "  " + text + "  " + hint
}
