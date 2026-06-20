// Theme registry + persistence. Themes are applied by setting
// `document.documentElement.dataset.theme`, which selects a token block in
// index.css (default / dark = `:root`, light = `:root[data-theme="light"]`).
// Components use semantic tokens, so this is the only place theme switching lives.

export type ThemeId = "dark" | "light"

export interface ThemeDef {
  id: ThemeId
  name: string
  /** Preview swatch colors (hex) for the picker. */
  swatch: { bg: string; fg: string; accent: string; border: string }
}

export const THEMES: ThemeDef[] = [
  {
    id: "dark",
    name: "Dark",
    swatch: { bg: "#0a0a0a", fg: "#ededec", accent: "#d97757", border: "#262624" },
  },
  {
    id: "light",
    name: "Light",
    swatch: { bg: "#fbf1c7", fg: "#3c3836", accent: "#c44a26", border: "#d5c39a" },
  },
]

const STORAGE_KEY = "hollow-theme"

export function getSavedTheme(): ThemeId {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v === "light" || v === "dark") return v
  } catch {
    /* ignore */
  }
  return "dark"
}

/** Apply a theme to the document. Called on load and when the user picks one. */
export function applyTheme(id: ThemeId): void {
  try {
    document.documentElement.dataset.theme = id
  } catch {
    /* ignore */
  }
}

/** Load the saved theme and apply it. Call once on app startup. */
export function initTheme(): void {
  applyTheme(getSavedTheme())
}

export function setTheme(id: ThemeId): void {
  applyTheme(id)
  try {
    localStorage.setItem(STORAGE_KEY, id)
  } catch {
    /* ignore */
  }
}