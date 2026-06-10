package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/core"
	"github.com/enough/enough/backend/opencode"
)

const maxRounds = 32

type Agent struct {
	cfg     config.Runtime
	client  *opencode.Client
	workDir string
	emit    func(core.Event)
}

func New(cfg config.Runtime, workDir string, emit func(core.Event)) *Agent {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	return &Agent{
		cfg:     cfg,
		client:  opencode.NewClient(cfg.Endpoint, cfg.APIKey, cfg.Model),
		workDir: workDir,
		emit:    emit,
	}
}

func (a *Agent) assistant(text string) {
	if a.emit != nil && text != "" {
		a.emit(core.Event{Kind: core.EventAssistantMessage, Data: text})
	}
}

func (a *Agent) toolActivity(name, args string) {
	if a.emit != nil {
		a.emit(core.Event{
			Kind: core.EventToolActivity,
			Data: fmt.Sprintf("%s(%s)", name, truncate(args, 80)),
		})
	}
}

func (a *Agent) err(text string) {
	if a.emit != nil {
		a.emit(core.Event{Kind: core.EventError, Data: text})
	}
}

// Run executes the agent loop for a user task using native tool_calls.
func (a *Agent) Run(ctx context.Context, task string) error {
	messages := []opencode.Message{
		{Role: "system", Content: opencode.StringContent(systemPrompt)},
		{Role: "user", Content: opencode.StringContent(task)},
	}

	tools := nativeTools()

	for round := 0; round < maxRounds; round++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := a.client.Chat(ctx, opencode.ChatRequest{
			Model:    a.cfg.Model,
			Messages: messages,
			Tools:    tools,
		})
		if err != nil {
			a.err(err.Error())
			return err
		}

		choice := resp.Choices[0]
		msg := choice.Message

		if len(msg.ToolCalls) == 0 {
			a.assistant(opencode.ContentString(msg))
			return nil
		}

		messages = append(messages, msg)

		for _, call := range msg.ToolCalls {
			a.toolActivity(call.Function.Name, call.Function.Arguments)

			result := a.executeTool(call.Function.Name, call.Function.Arguments)
			if result.isErr {
				a.toolActivity(call.Function.Name, truncate(result.output, 200))
			}

			messages = append(messages, opencode.Message{
				Role:       "tool",
				ToolCallID: call.ID,
				Name:       call.Function.Name,
				Content:    opencode.StringContent(result.output),
			})
		}
	}

	err := fmt.Errorf("agent stopped after %d tool rounds", maxRounds)
	a.err(err.Error())
	return err
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
