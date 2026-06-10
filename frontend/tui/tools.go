package tui

import "github.com/enough/enough/backend/core"

func (a *App) handleToolStart(ev core.ToolCallEvent) {
	msg := chatMsg{
		role:        "tool",
		toolID:      ev.ID,
		toolName:    ev.Name,
		toolArgs:    ev.Args,
		toolPending: true,
	}
	switch ev.Name {
	case "write_file":
		msg.toolAdded, msg.toolRemoved = diffWriteFile(ev.Args)
	case "edit_file":
		msg.toolAdded, msg.toolRemoved = diffEditFile(ev.Args)
	}
	a.messages = append(a.messages, msg)
	a.bumpChat()
}

func (a *App) handleToolResult(ev core.ToolCallEvent) {
	for i := len(a.messages) - 1; i >= 0; i-- {
		msg := &a.messages[i]
		if msg.role != "tool" {
			continue
		}
		if ev.ID != "" && msg.toolID != ev.ID {
			continue
		}
		if !msg.toolPending && msg.toolResult != "" {
			continue
		}
		msg.toolPending = false
		msg.toolResult = ev.Result
		msg.toolError = ev.Error
		a.bumpChat()
		return
	}

	a.messages = append(a.messages, chatMsg{
		role:       "tool",
		toolID:     ev.ID,
		toolResult: ev.Result,
		toolError:  ev.Error,
	})
	a.bumpChat()
}

func (a *App) toggleToolsExpanded() {
	a.toolsExpanded = !a.toolsExpanded
	state := "collapsed"
	if a.toolsExpanded {
		state = "expanded"
	}
	a.appendMessage("system", "Tool output: "+state)
}
