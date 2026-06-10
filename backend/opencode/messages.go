package opencode

import "fmt"

const toolIncompleteMsg = "Error: tool call was not completed"

// RepairToolMessages ensures every assistant tool_call has a matching tool response.
// OpenAI-compatible APIs reject requests when tool_calls are missing replies.
func RepairToolMessages(msgs []Message) []Message {
	out := make([]Message, 0, len(msgs))

	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for j := range msg.ToolCalls {
				if msg.ToolCalls[j].ID == "" {
					msg.ToolCalls[j].ID = fmt.Sprintf("call_%d_%d", i, j)
				}
			}
		}

		out = append(out, msg)

		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			continue
		}

		required := make([]ToolCall, len(msg.ToolCalls))
		copy(required, msg.ToolCalls)

		answered := make(map[string]bool)
		i++
		for i < len(msgs) && msgs[i].Role == "tool" {
			tm := msgs[i]
			if tm.ToolCallID == "" {
				tm.ToolCallID = required[0].ID
			}
			if !answered[tm.ToolCallID] {
				out = append(out, tm)
				answered[tm.ToolCallID] = true
			}
			i++
		}
		i--

		for _, tc := range required {
			if answered[tc.ID] {
				continue
			}
			out = append(out, Message{
				Role:       "tool",
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Content:    StringContent(toolIncompleteMsg),
			})
		}
	}

	return out
}
