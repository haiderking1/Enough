package tui

import "github.com/charmbracelet/lipgloss"

type Styles struct {
	Text        lipgloss.Style
	LogDim      lipgloss.Style
	LogAccent   lipgloss.Style
	LogWarn     lipgloss.Style
	LogErr      lipgloss.Style
	LogOk       lipgloss.Style
	InputBox    lipgloss.Style
	InputPrompt lipgloss.Style
	InputHint   lipgloss.Style
	InputCaret  lipgloss.Style
	AssistBullet lipgloss.Style
	AssistText  lipgloss.Style
	AssistError lipgloss.Style
	ToolActivity lipgloss.Style
	SlashMenu   lipgloss.Style
	SlashSelected lipgloss.Style
	SlashName   lipgloss.Style
	SlashDesc   lipgloss.Style
	SlashDim    lipgloss.Style
}

func NewStyles() Styles {
	base := lipgloss.NewStyle()

	border := lipgloss.Color("#2a2a34")
	text := lipgloss.Color("#e8e8ed")
	textDim := lipgloss.Color("#6b6b78")
	accent := lipgloss.Color("#7c8cff")
	amber := lipgloss.Color("#f0b429")
	green := lipgloss.Color("#3dd68c")
	red := lipgloss.Color("#f25c5c")

	return Styles{
		Text: base.Copy().
			Foreground(text),

		LogDim: base.Copy().
			Foreground(textDim),

		LogAccent: base.Copy().
			Foreground(accent).
			Bold(true),

		LogWarn: base.Copy().
			Foreground(amber),

		LogErr: base.Copy().
			Foreground(red).
			Bold(true),

		LogOk: base.Copy().
			Foreground(green),

		InputBox: base.Copy().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1),

		InputPrompt: base.Copy().
			Foreground(textDim),

		InputHint: base.Copy().
			Foreground(textDim).
			Italic(true),

		InputCaret: base.Copy().
			Foreground(lipgloss.Color("#0d0d0f")).
			Background(lipgloss.Color("#e8e8ed")),

		AssistBullet: base.Copy().
			Foreground(text).
			Bold(true),

		AssistText: base.Copy().
			Foreground(text),

		AssistError: base.Copy().
			Foreground(red).
			Bold(true),

		ToolActivity: base.Copy().
			Foreground(textDim).
			Italic(true),

		SlashMenu: base.Copy().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Padding(0, 1),

		SlashSelected: base.Copy().
			Foreground(accent).
			Bold(true),

		SlashName: base.Copy().
			Foreground(text).
			Bold(true),

		SlashDesc: base.Copy().
			Foreground(textDim),

		SlashDim: base.Copy().
			Foreground(textDim).
			Italic(true),
	}
}
