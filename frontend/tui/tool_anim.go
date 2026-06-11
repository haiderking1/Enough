package tui

import "github.com/charmbracelet/lipgloss"

func (a *App) hasAnimatingTools() bool {
	for _, msg := range a.messages {
		if msg.role == "tool" && msg.toolPending && msg.toolName == "agent_swarm" {
			return true
		}
	}
	return false
}

func toolGroupAnimates(tools []chatMsg) bool {
	for _, msg := range tools {
		if msg.toolPending && msg.toolName == "agent_swarm" {
			return true
		}
	}
	return false
}

var spawnSpinnerStyles = func(styles Styles) []lipgloss.Style {
	return []lipgloss.Style{
		styles.CompactionSpinner,
		styles.LogAccent,
		styles.AssistBullet,
		styles.CompactionSpinner.Copy().Foreground(lipgloss.Color("#4ec9e0")),
	}
}

func spawnBullet(styles Styles, animating bool, frame int) string {
	if animating {
		palette := spawnSpinnerStyles(styles)
		return palette[frame%len(palette)].Render("*")
	}
	return styles.AssistBullet.Render("*")
}
