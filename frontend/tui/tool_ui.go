package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enough/enough/frontend/tui/term"
)

type toolKind string

const (
	toolKindWrite toolKind = "write"
	toolKindEdit  toolKind = "edit"
	toolKindRead  toolKind = "read"
	toolKindBash  toolKind = "bash"
	toolKindOther toolKind = "other"
)

type toolRow struct {
	Kind    toolKind
	Action  string
	Target  string
	Added   int
	Removed int
	Lines   int
	Pending bool
	Error   bool
	Output  string
}

func parseToolRow(msg chatMsg) toolRow {
	name := msg.toolName
	if name == "" {
		name = "tool"
	}

	var args map[string]json.RawMessage
	_ = json.Unmarshal([]byte(msg.toolArgs), &args)

	row := toolRow{
		Pending: msg.toolPending,
		Error:   msg.toolError,
		Output:  strings.TrimSpace(msg.toolResult),
		Added:   msg.toolAdded,
		Removed: msg.toolRemoved,
	}

	switch name {
	case "write_file":
		row.Kind = toolKindWrite
		row.Action = "Write"
		row.Target = displayPath(jsonString(args["path"]))
		if row.Added == 0 && row.Removed == 0 {
			if content := jsonString(args["content"]); content != "" {
				row.Added = countLines(content)
			}
		}
	case "edit_file":
		row.Kind = toolKindEdit
		row.Action = "Edited"
		row.Target = displayPath(jsonString(args["path"]))
		if row.Added == 0 && row.Removed == 0 {
			old := jsonString(args["old_string"])
			newS := jsonString(args["new_string"])
			row.Removed = countLines(old)
			row.Added = countLines(newS)
		}
	case "read_file":
		row.Kind = toolKindRead
		row.Action = "Read"
		row.Target = displayPathFull(jsonString(args["path"]))
		if row.Output != "" {
			row.Lines = countLines(row.Output)
		}
	case "bash":
		row.Kind = toolKindBash
		row.Action = "Bash"
		row.Target = oneLine(jsonString(args["command"]))
	case "web_search":
		row.Kind = toolKindOther
		row.Action = "Web"
		row.Target = truncateMiddle(oneLine(jsonString(args["query"])), 56)
	case "list_dir":
		row.Kind = toolKindOther
		row.Action = "List"
		row.Target = displayPath(jsonString(args["path"]))
		if row.Target == "" {
			row.Target = "."
		}
	default:
		row.Kind = toolKindOther
		row.Action = toolActionLabel(name)
		row.Target = truncateMiddle(oneLine(msg.toolArgs), 56)
	}

	if row.Target == "" {
		row.Target = name
	}
	return row
}

func displayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	path = filepath.ToSlash(path)
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, filepath.ToSlash(home)) {
		return "~" + strings.TrimPrefix(path, filepath.ToSlash(home))
	}
	return path
}

func displayPathFull(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			path = filepath.Join(cwd, path)
		}
	}
	path = filepath.ToSlash(path)
	home, err := os.UserHomeDir()
	if err == nil {
		home = filepath.ToSlash(home)
		if strings.HasPrefix(path, home) {
			return "~" + strings.TrimPrefix(path, home)
		}
	}
	return path
}

func toolActionLabel(name string) string {
	parts := strings.Fields(strings.ReplaceAll(name, "_", " "))
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
	}
	return strings.Join(parts, " ")
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimRight(s, "\n"), "\n"))
}

func renderToolGroup(styles Styles, tools []chatMsg, width int, expanded bool) string {
	if len(tools) == 0 {
		return ""
	}

	rows := make([]toolRow, len(tools))
	for i, msg := range tools {
		rows[i] = parseToolRow(msg)
	}

	var lines []string

	if len(tools) > 1 {
		header := fmt.Sprintf("Updated  %d items", len(tools))
		lines = append(lines, styles.ToolMuted.Render(header))
	}

	for i, msg := range tools {
		row := rows[i]
		lines = append(lines, renderToolBlock(styles, row, width, expanded)...)

		if expanded && row.Output != "" && !row.Pending && row.Kind != toolKindBash {
			detail := limitToolOutput(row.Output, true)
			outStyle := styles.ToolOutput
			if row.Error {
				outStyle = styles.AssistError
			}
			for j, line := range strings.Split(detail, "\n") {
				prefix := "  "
				if j == 0 {
					prefix = "└ "
				}
				lines = append(lines, outStyle.Render(prefix+line))
			}
		}
		_ = msg
	}

	return strings.Join(lines, "\n")
}

func renderToolBlock(styles Styles, row toolRow, width int, expanded bool) []string {
	switch row.Kind {
	case toolKindWrite:
		return []string{renderWriteLine(styles, row)}
	case toolKindEdit:
		return []string{renderEditLine(styles, row)}
	case toolKindRead:
		return renderReadBlock(styles, row)
	case toolKindBash:
		return renderBashBlock(styles, row, width, expanded)
	default:
		return []string{renderGenericLine(styles, row)}
	}
}

func renderWriteLine(styles Styles, row toolRow) string {
	head := styles.ToolMuted.Render("Write " + row.Target)
	if row.Pending {
		return head + " " + styles.ToolPending.Render("…")
	}
	delta := styles.ToolDelta.Render(fmt.Sprintf("+%d", row.Added)) +
		styles.ToolDeltaRemoved.Render(fmt.Sprintf("-%d", row.Removed))
	return head + " " + delta + " " + styles.ToolMuted.Render(">")
}

func renderEditLine(styles Styles, row toolRow) string {
	head := styles.ToolMuted.Render("Edited " + row.Target)
	if row.Pending {
		return head + " " + styles.ToolPending.Render("…")
	}
	delta := styles.ToolDelta.Render(fmt.Sprintf("+%d", row.Added)) +
		styles.ToolDeltaRemoved.Render(fmt.Sprintf("-%d", row.Removed))
	return head + " " + delta + " " + styles.ToolMuted.Render(">")
}

func renderReadBlock(styles Styles, row toolRow) []string {
	header := styles.ToolBullet.Render("●") + " " +
		styles.ToolAction.Render("Read") + " " +
		styles.ToolTarget.Render(row.Target)

	lines := []string{header}
	switch {
	case row.Pending:
		lines = append(lines, styles.ToolSub.Render("└ …"))
	case row.Lines > 0:
		lines = append(lines, styles.ToolSub.Render(fmt.Sprintf("└ Read %d lines", row.Lines)))
	}
	return lines
}

func renderBashBlock(styles Styles, row toolRow, width int, expanded bool) []string {
	cmd := row.Target
	if cmd == "" {
		cmd = "command"
	}
	cmd = term.TruncateWidth(cmd, width-12)

	header := styles.ToolBullet.Render("●") + " " +
		styles.ToolAction.Render("Bash") + " " +
		styles.ToolTarget.Render(cmd)

	lines := []string{header}

	out := strings.TrimRight(row.Output, "\n")
	if out == "" {
		if row.Pending {
			lines = append(lines, styles.ToolPending.Render("└ running… (esc to cancel)"))
		}
		return lines
	}

	outStyle := styles.ToolOutput
	if row.Error {
		outStyle = styles.AssistError
	}

	// While streaming and collapsed, show the tail so the newest output is
	// visible; once finished, show the head with a "more lines" hint.
	var detail string
	if row.Pending && !expanded {
		detail = tailLines(out, 8)
	} else {
		detail = limitToolOutput(out, expanded)
	}

	dl := strings.Split(detail, "\n")
	for i, line := range dl {
		if line == "" && i == len(dl)-1 {
			continue
		}
		prefix := "  "
		if i == 0 {
			prefix = "└ "
		}
		lines = append(lines, outStyle.Render(prefix+line))
	}

	if row.Pending {
		lines = append(lines, styles.ToolPending.Render("  running… (esc to cancel)"))
	}
	return lines
}

// tailLines returns the last n non-trailing-empty lines of text, prefixed with
// an elision hint when earlier lines were dropped.
func tailLines(text string, n int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= n {
		return text
	}
	tail := lines[len(lines)-n:]
	return fmt.Sprintf("… (%d earlier lines)\n", len(lines)-n) + strings.Join(tail, "\n")
}

func renderGenericLine(styles Styles, row toolRow) string {
	header := styles.ToolBullet.Render("●") + " " +
		styles.ToolAction.Render(row.Action) + " " +
		styles.ToolTarget.Render(row.Target)
	if row.Pending {
		return header + " " + styles.ToolPending.Render("…")
	}
	return header
}

func formatToolCall(name, argsJSON string) string {
	row := parseToolRow(chatMsg{toolName: name, toolArgs: argsJSON})
	if row.Target != "" && row.Action != "" {
		return row.Action + " " + row.Target
	}
	return name
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	return ""
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func truncateMiddle(s string, max int) string {
	if len(s) <= max {
		return s
	}
	head := max/2 - 1
	tail := max - head - 1
	return s[:head] + "…" + s[len(s)-tail:]
}

func limitToolOutput(text string, expanded bool) string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return ""
	}
	if expanded {
		return text
	}
	const maxLines = 8
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n… (%d more lines)", len(lines)-maxLines)
}
