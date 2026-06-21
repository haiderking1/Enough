import type { AgentModel } from "../agent/rpc"

/** Badge shown in the model list (mirrors backend FormatThinkingBadge). */
export function formatThinkingBadge(model: AgentModel, level: string): string {
  const levels = model.thinkingLevels ?? []
  if (levels.length <= 1) {
    if (model.reasoning) return "reasoning"
    return ""
  }
  if (!level || level === "off") return "off"
  return level
}
