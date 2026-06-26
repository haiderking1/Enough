// Maps each tool name to its toolset category.
export const tool_to_toolset: Record<string, string> = {
  read_file: "file",
  write_file: "file",
  edit_file: "file",
  list_dir: "file",
  glob: "file",
  grep: "file",
  bash: "terminal",
  web_search: "web",
  web_fetch: "web",
  browser: "web",
  agent_swarm: "delegate",
  skills_list: "skills",
  skill_view: "skills",
  skill_manage: "skills",
  memory: "memory",
};

// GetAvailableToolsets takes a slice of active tool names and returns a sorted slice of toolset names.
export const get_available_toolsets = (active_tools: string[]): string[] => {
  const seen = new Set<string>();
  for (const tool of active_tools) {
    const ts = tool_to_toolset[tool];
    if (ts !== undefined) {
      seen.add(ts);
    }
  }
  return Array.from(seen).sort();
};

