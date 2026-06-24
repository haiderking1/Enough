import type { AgentModel } from "../agent/rpc"

/** Default thinking level when selecting a model (first enabled non-off level). */
export function defaultThinkingLevel(model: AgentModel, maxMode = false): string {
  const levels = model.thinkingLevels ?? []
  if (!levels.length) return ""
  if (maxMode && levels.includes("max")) return "max"
  if (levels.includes("adaptive")) return "adaptive"
  return levels.find((l) => l !== "off") ?? levels[0]
}

/** Human label for a thinking level (uses catalog labels from the backend when present). */
export function thinkingLevelLabel(model: AgentModel, level: string): string {
  const levels = model.thinkingLevels ?? []
  const labels = model.thinkingLevelLabels ?? []
  const idx = levels.indexOf(level)
  if (idx >= 0 && labels[idx]) return labels[idx]
  if (!level || level === "off") return "off"
  return level
}

function capitalizeLabel(label: string): string {
  if (!label) return label
  return label.charAt(0).toUpperCase() + label.slice(1)
}

/** Label shown in the model picker and thinking-mode menu. */
export function formatThinkingLevelDisplay(model: AgentModel, level: string): string {
  return capitalizeLabel(thinkingLevelLabel(model, level))
}

/** Badge shown in the model list (mirrors backend FormatThinkingBadge). */
export function formatThinkingBadge(model: AgentModel, level: string): string {
  const levels = model.thinkingLevels ?? []
  if (levels.length <= 1) {
    if (model.reasoning) return "reasoning"
    return ""
  }
  if (!level || level === "off") return "off"
  return thinkingLevelLabel(model, level)
}
