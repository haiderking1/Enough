package tui

import (
	"context"

	"github.com/enough/enough/backend/agent"
	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/core"
)

func startAgent(task string) <-chan core.Event {
	ch := make(chan core.Event, 64)

	go func() {
		defer close(ch)

		emit := func(e core.Event) {
			ch <- e
		}

		cfg, err := config.LoadRuntime()
		if err != nil {
			emit(core.Event{Kind: core.EventError, Data: err.Error()})
			return
		}

		a := agent.New(cfg, "", emit)
		_ = a.Run(context.Background(), task)
	}()

	return ch
}

func eventToChatMsg(e core.Event) (chatMsg, bool) {
	switch e.Kind {
	case core.EventAssistantMessage:
		text, ok := e.Data.(string)
		return chatMsg{role: "assistant", text: text}, ok

	case core.EventToolActivity:
		text, ok := e.Data.(string)
		return chatMsg{role: "tool", text: text}, ok

	case core.EventError:
		text, ok := e.Data.(string)
		return chatMsg{role: "error", text: text}, ok

	case core.EventSystem:
		text, ok := e.Data.(string)
		return chatMsg{role: "system", text: text}, ok

	case core.EventLog:
		entry, ok := e.Data.(core.LogEntry)
		if !ok {
			return chatMsg{}, false
		}
		switch entry.Level {
		case "err":
			return chatMsg{role: "error", text: entry.Message}, true
		default:
			return chatMsg{role: "system", text: entry.Message}, true
		}

	default:
		return chatMsg{}, false
	}
}
