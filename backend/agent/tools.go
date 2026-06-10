package agent

import "fmt"

type toolResult struct {
	output string
	isErr  bool
}

func (a *Agent) executeTool(name, argsJSON string) toolResult {
	switch name {
	case "read_file":
		return a.toolReadFile(argsJSON)
	case "write_file":
		return a.toolWriteFile(argsJSON)
	case "edit_file":
		return a.toolEditFile(argsJSON)
	case "list_dir":
		return a.toolListDir(argsJSON)
	case "bash":
		return a.toolBash(argsJSON)
	case "web_search":
		return a.toolWebSearch(argsJSON)
	default:
		return toolResult{output: fmt.Sprintf("unknown tool: %s", name), isErr: true}
	}
}
