package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	truncated := false
	if len(out) > max {
		out = out[:max]
		truncated = true
	}

	// Header carries the line count so callers never need an external `wc -l`.
	// Counting on the full data, not the truncated view, keeps the total accurate.
	lines := strings.Count(string(data), "\n")
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		lines++ // final line without a trailing newline still counts
	}
	header := fmt.Sprintf("Read %d lines from %s\n", lines, path)
	if truncated {
		out += "\n... truncated ..."
	}
	return toolResult{output: header + out}
}
