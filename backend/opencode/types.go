package opencode

import (
	"encoding/json"
	"strings"
)

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type ChatRequest struct {
	Model           string          `json:"model"`
	Messages        []Message       `json:"messages"`
	Tools           []Tool          `json:"tools,omitempty"`
	Stream          bool            `json:"stream,omitempty"`
	Thinking        *ThinkingParams `json:"thinking,omitempty"`
	ReasoningEffort string          `json:"reasoning_effort,omitempty"`
	StreamOptions   *StreamOptions  `json:"stream_options,omitempty"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	Input       int `json:"input"`
	Output      int `json:"output"`
	TotalTokens int `json:"totalTokens,omitempty"`
	CacheRead   int `json:"cacheRead,omitempty"`
	CacheWrite  int `json:"cacheWrite,omitempty"`
}

type Message struct {
	Role             string     `json:"role"`
	Content          *string    `json:"content"`
	ReasoningContent *string    `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	Name             string     `json:"name,omitempty"`
	Usage            *Usage     `json:"usage,omitempty"`
}

func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	var aux struct {
		Content json.RawMessage `json:"content"`
		*Alias
	}
	aux.Alias = (*Alias)(m)
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(aux.Content) == 0 {
		return nil
	}

	// Try to unmarshal as string
	var s string
	if err := json.Unmarshal(aux.Content, &s); err == nil {
		m.Content = &s
		return nil
	}

	// Try to unmarshal as array of blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(aux.Content, &blocks); err == nil {
		var contentParts []string
		for _, b := range blocks {
			if b.Type == "text" && b.Text != "" {
				contentParts = append(contentParts, b.Text)
			}
		}
		res := strings.Join(contentParts, "\n")
		m.Content = &res
		return nil
	}

	return nil
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

func StringContent(s string) *string {
	return &s
}

func ContentString(m Message) string {
	if m.Content == nil {
		return ""
	}
	return *m.Content
}
