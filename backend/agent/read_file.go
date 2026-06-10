package agent

import (
	"encoding/json"
	"os"

	"github.com/enough/enough/backend/opencode"
)

func readFileTool() opencode.Tool {
	return opencode.Tool{
		Type: "function",
		Function: opencode.ToolFunction{
			Name:        "read_file",
			Description: "Read a file from the project workspace",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "Relative or absolute path"}
				},
				"required": ["path"]
			}`),
		},
	}
}

func (a *Agent) toolReadFile(argsJSON string) toolResult {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	path, err := a.resolvePath(args.Path)
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	const max = 64_000
	out := string(data)
	if len(out) > max {
		out = out[:max] + "\n... truncated ..."
	}
	return toolResult{output: out}
}
