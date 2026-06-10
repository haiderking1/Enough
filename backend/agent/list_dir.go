package agent

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/enough/enough/backend/opencode"
)

func listDirTool() opencode.Tool {
	return opencode.Tool{
		Type: "function",
		Function: opencode.ToolFunction{
			Name:        "list_dir",
			Description: "List entries in a directory",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "Directory path, default ."}
				}
			}`),
		},
	}
}

func (a *Agent) toolListDir(argsJSON string) toolResult {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	path := args.Path
	if path == "" {
		path = "."
	}

	path, err := a.resolvePath(path)
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	var b strings.Builder
	for _, e := range entries {
		if e.IsDir() {
			b.WriteString(e.Name())
			b.WriteString("/\n")
			continue
		}
		b.WriteString(e.Name())
		b.WriteByte('\n')
	}
	return toolResult{output: b.String()}
}
