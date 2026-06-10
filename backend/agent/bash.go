package agent

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/enough/enough/backend/opencode"
)

func bashTool() opencode.Tool {
	return opencode.Tool{
		Type: "function",
		Function: opencode.ToolFunction{
			Name:        "bash",
			Description: "Run a shell command in the project workspace",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"command": {"type": "string"}
				},
				"required": ["command"]
			}`),
		},
	}
}

func (a *Agent) toolBash(argsJSON string) toolResult {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	cmd := exec.Command("bash", "-lc", args.Command)
	cmd.Dir = a.workDir
	out, err := cmd.CombinedOutput()

	const max = 32_000
	text := string(out)
	if len(text) > max {
		text = text[:max] + "\n... truncated ..."
	}

	if err != nil {
		return toolResult{output: fmt.Sprintf("%v\n%s", err, text), isErr: true}
	}
	return toolResult{output: text}
}
