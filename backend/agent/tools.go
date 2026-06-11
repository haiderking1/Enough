package agent

import (
	"context"
	"fmt"
)

type toolResult struct {
	output string
	isErr  bool
}

func (a *Agent) executeTool(ctx context.Context, id, name, argsJSON string) toolResult {
	switch name {
	case "read_file":
		return a.toolReadFile(argsJSON)
	case "write_file":
		return a.toolWriteFile(argsJSON)
	case "edit_file":
		return a.toolEditFile(argsJSON)
	case "list_dir":
		return a.toolListDir(argsJSON)
	case "glob":
		return a.toolGlob(argsJSON)
	case "grep":
		return a.toolGrep(argsJSON)
	case "bash":
		return a.toolBash(ctx, id, argsJSON)
	case "web_search":
		return a.toolWebSearch(argsJSON)
	case "agent_swarm":
		return a.toolAgentSwarm(ctx, id, argsJSON, 0)
	default:
		return toolResult{output: fmt.Sprintf("unknown tool: %s", name), isErr: true}
	}
}

func (a *Agent) executeSwarmTool(ctx context.Context, id, name, argsJSON string) toolResult {
	if name == "agent_swarm" {
		return a.toolAgentSwarm(ctx, id, argsJSON, a.swarmDepth+1)
	}
	return a.executeTool(ctx, id, name, argsJSON)
}
