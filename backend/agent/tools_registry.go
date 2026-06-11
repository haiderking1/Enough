package agent

import "github.com/enough/enough/backend/opencode"

// nativeTools is the full tool set available to the main (coder) agent,
// including agent_swarm for parallel sub-agents.
func nativeTools() []opencode.Tool {
	return []opencode.Tool{
		readFileTool(),
		writeFileTool(),
		editFileTool(),
		listDirTool(),
		globTool(),
		grepTool(),
		bashTool(),
		webSearchTool(),
		agentSwarmTool(),
	}
}

// workerTools returns the coding toolset for a swarm worker. Workers at depth
// below maxSwarmDepth also get a nested agent_swarm tool.
func workerTools(depth int) []opencode.Tool {
	tools := []opencode.Tool{
		readFileTool(),
		writeFileTool(),
		editFileTool(),
		listDirTool(),
		globTool(),
		grepTool(),
		bashTool(),
		webSearchTool(),
	}
	if depth < maxSwarmDepth {
		tools = append(tools, agentSwarmTool())
	}
	return tools
}

// plannerTools is the read-only subset for the swarm goal planner.
func plannerTools() []opencode.Tool {
	return []opencode.Tool{
		readFileTool(),
		listDirTool(),
		globTool(),
		grepTool(),
	}
}
