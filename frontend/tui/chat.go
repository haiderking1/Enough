package tui

import (
	"strings"
)

type chatMsg struct {
	role string // user, assistant, tool, error, system
	text string
}

func wrapText(text string, width int) string {
	if width < 10 {
		width = 10
	}

	parts := strings.Split(text, "\n")
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapWords(part, width))
	}
	return strings.Join(out, "\n")
}

func wrapWords(text string, width int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var line strings.Builder

	flush := func() {
		if line.Len() > 0 {
			lines = append(lines, line.String())
			line.Reset()
		}
	}

	for _, word := range words {
		if line.Len() == 0 {
			line.WriteString(word)
			continue
		}
		if line.Len()+1+len(word) > width {
			flush()
			line.WriteString(word)
			continue
		}
		line.WriteString(" ")
		line.WriteString(word)
	}
	flush()
	return strings.Join(lines, "\n")
}

func renderChat(styles Styles, messages []chatMsg, width int) string {
	if width <= 0 {
		width = 80
	}

	contentW := width - 2
	var blocks []string

	for _, msg := range messages {
		switch msg.role {
		case "user":
			blocks = append(blocks, renderUser(styles, msg.text, contentW))
		case "assistant":
			blocks = append(blocks, renderAssistant(styles, msg.text, contentW))
		case "tool":
			blocks = append(blocks, styles.ToolActivity.Render("  ⚙ "+msg.text))
		case "error":
			blocks = append(blocks, styles.AssistError.Render("● "+wrapText(msg.text, contentW-4)))
		case "system":
			blocks = append(blocks, styles.LogDim.Render(wrapText(msg.text, contentW-4)))
		}
	}

	return strings.Join(blocks, "\n\n")
}

func renderUser(styles Styles, text string, width int) string {
	wrapped := wrapText(text, width-4)
	lines := strings.Split(wrapped, "\n")
	if len(lines) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString(styles.InputPrompt.Render("❯ "))
	out.WriteString(styles.Text.Render(lines[0]))

	for _, line := range lines[1:] {
		out.WriteString("\n")
		out.WriteString(styles.Text.Render("  " + line))
	}
	return out.String()
}

func renderAssistant(styles Styles, text string, width int) string {
	wrapped := wrapText(text, width-4)
	lines := strings.Split(wrapped, "\n")
	if len(lines) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString(styles.AssistBullet.Render("● "))
	out.WriteString(styles.AssistText.Render(lines[0]))

	for _, line := range lines[1:] {
		out.WriteString("\n")
		out.WriteString(styles.AssistText.Render("  " + line))
	}
	return out.String()
}
