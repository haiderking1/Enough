package tui

import (
	"fmt"

	"github.com/enough/enough/backend/core"
	"github.com/enough/enough/frontend/tui/term"
)

// renderObligations renders the compact obligation panel: a summary line,
// plus one line per open obligation so the user can see exactly what blocks
// completion. Nothing is rendered when the turn has no obligations.
func (a *App) renderObligations(width int) []string {
	ev := a.obligationState
	if ev == nil || len(ev.Items) == 0 || width <= 0 {
		return nil
	}

	summary := fmt.Sprintf("obligations: %d open · %d closed", ev.Open, ev.Closed)
	var lines []string
	if ev.Open > 0 {
		lines = append(lines, a.styles.FooterWarn.Render(term.TruncateWidth(summary, width)))
		for _, item := range ev.Items {
			if item.Closed {
				continue
			}
			row := fmt.Sprintf("  ○ %s — %s", item.Kind, item.Description)
			lines = append(lines, a.styles.FooterWarn.Render(term.TruncateWidth(row, width)))
		}
	} else {
		lines = append(lines, a.styles.LogDim.Render(term.TruncateWidth(summary+" ✓", width)))
	}
	return lines
}

func (a *App) setObligationState(ev core.ObligationEvent) {
	a.obligationState = &ev
}
