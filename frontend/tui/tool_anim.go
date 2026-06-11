package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
)

// npmDotSpinner — block-dot cycle (cli-spinners "dots2"). Sits lower in the cell
// than braille npm frames so it lines up with latin text; still reads as npm install.
var npmDotSpinner = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

const spawnIdleGlyph = "●"

// npmSpinnerHoldTicks: 80ms per frame @ default tick → ~640ms per rotation.
const npmSpinnerHoldTicks = 1

func npmSpinnerAt(tick int) string {
	return npmDotSpinner[(tick/npmSpinnerHoldTicks)%len(npmDotSpinner)]
}

func npmSpinnerStyle(styles Styles) lipgloss.Style {
	return styles.CompactionSpinner.Copy().Bold(true).Inline(true)
}

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

func spawnBullet(styles Styles, animating bool, frame int) string {
	spin := npmSpinnerStyle(styles)
	if !animating {
		return spin.Foreground(lipgloss.Color("#4ec9e0")).Render(spawnIdleGlyph)
	}
	return spin.Render(npmSpinnerAt(frame))
}

func spawnBulletPlain(animating bool, frame int) string {
	return ansi.Strip(spawnBullet(NewStyles(), animating, frame))
}

func spawnBulletWidth(animating bool, frame int) int {
	return runewidth.StringWidth(spawnBulletPlain(animating, frame))
}
