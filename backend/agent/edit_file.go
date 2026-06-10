package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/enough/enough/backend/opencode"
)

func editFileTool() opencode.Tool {
	return opencode.Tool{
		Type: "function",
		Function: opencode.ToolFunction{
			Name:        "edit_file",
			Description: "Replace exact text in an existing file. old_string must match uniquely unless replace_all is true.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"path": {"type": "string", "description": "File path"},
					"old_string": {"type": "string", "description": "Exact text to find, including whitespace"},
					"new_string": {"type": "string", "description": "Replacement text"},
					"replace_all": {"type": "boolean", "description": "Replace every occurrence instead of requiring a unique match"}
				},
				"required": ["path", "old_string", "new_string"]
			}`),
		},
	}
}

func (a *Agent) toolEditFile(argsJSON string) toolResult {
	var args struct {
		Path        string `json:"path"`
		OldString   string `json:"old_string"`
		NewString   string `json:"new_string"`
		ReplaceAll  bool   `json:"replace_all"`
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

	newContent, count, err := applyEdit(string(data), args.OldString, args.NewString, args.ReplaceAll)
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	if err := os.WriteFile(path, []byte(newContent), 0o644); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	return toolResult{output: fmt.Sprintf("edited %s (%d replacement(s))", path, count)}
}

func applyEdit(content, old, new string, replaceAll bool) (string, int, error) {
	if old == "" {
		return "", 0, fmt.Errorf("old_string is required")
	}

	count := strings.Count(content, old)
	if count == 0 {
		return "", 0, fmt.Errorf("old_string not found in file")
	}

	if !replaceAll {
		if count > 1 {
			return "", 0, fmt.Errorf("old_string is not unique (%d matches); add context or set replace_all", count)
		}
		return strings.Replace(content, old, new, 1), 1, nil
	}

	return strings.Replace(content, old, new, -1), count, nil
}
