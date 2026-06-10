package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/enough/enough/backend/opencode"
)

func writeFileTool() opencode.Tool {
	return opencode.Tool{
		Type: "function",
		Function: opencode.ToolFunction{
			Name:        "write_file",
			Description: "Write content to a file in the workspace",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string"},
					"content": {"type": "string"}
				},
				"required": ["path", "content"]
			}`),
		},
	}
}

func (a *Agent) toolWriteFile(argsJSON string) toolResult {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	path, err := a.resolvePath(args.Path)
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	if err := os.WriteFile(path, []byte(args.Content), 0o644); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	return toolResult{output: fmt.Sprintf("wrote %d bytes to %s", len(args.Content), path)}
}
