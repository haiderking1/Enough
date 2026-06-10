package agent

import (
	"context"
	"encoding/json"

	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/web"
)

func webSearchTool() opencode.Tool {
	return opencode.Tool{
		Type: "function",
		Function: opencode.ToolFunction{
			Name: "web_search",
			Description: "Search the web via bundled SearXNG and return full readable page content for each result (not snippets). " +
				"Pass a search query, or a full http(s) URL to fetch a single page directly.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "string",
						"description": "Search query or full URL to fetch"
					},
					"max_pages": {
						"type": "integer",
						"description": "Top results to fetch fully (default 3, max 5). Ignored for direct URLs."
					}
				},
				"required": ["query"]
			}`),
		},
	}
}

func (a *Agent) toolWebSearch(argsJSON string) toolResult {
	var args struct {
		Query    string `json:"query"`
		MaxPages int    `json:"max_pages"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	hits, err := web.Search(context.Background(), args.Query, web.Options{MaxPages: args.MaxPages})
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}
	return toolResult{output: web.Format(hits)}
}
