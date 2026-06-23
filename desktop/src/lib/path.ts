/** Last segment of a file path (works with `/` and `\`, including Windows drive paths). */
export function pathBasename(cwd: string): string {
  if (!cwd) return "(unknown)"
  const trimmed = cwd.replace(/[/\\]+$/, "")
  const sep = Math.max(trimmed.lastIndexOf("/"), trimmed.lastIndexOf("\\"))
  if (sep === -1) return trimmed
  return trimmed.slice(sep + 1) || trimmed
}

/** Sidebar label: custom alias, or folder name; never a full path. */
export function projectDisplayName(cwd: string, alias?: string): string {
  const raw = (alias?.trim() || pathBasename(cwd)).trim()
  if (raw.includes("\\") || raw.includes("/")) return pathBasename(raw)
  return raw || pathBasename(cwd)
}
