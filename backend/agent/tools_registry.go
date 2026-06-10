package agent

import "github.com/enough/enough/backend/opencode"

func nativeTools() []opencode.Tool {
	return []opencode.Tool{
		readFileTool(),
		writeFileTool(),
		editFileTool(),
		listDirTool(),
		bashTool(),
		webSearchTool(),
	}
}
