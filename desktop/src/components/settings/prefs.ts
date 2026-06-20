// App-level preferences persisted to localStorage. These drive the General
// settings rows. Theme is also applied to the document (see themes/index.ts);
// it's a free-form string so new theme files can be added without touching prefs.
export interface HollowPrefs {
  theme: string
  timeFormat: "system" | "12" | "24"
  diffWrap: boolean
  hideWhitespace: boolean
  assistantOutput: boolean
  newThreadLocation: "local"
}

// Defaults match the reference layout's on/off states.
export const DEFAULT_PREFS: HollowPrefs = {
  theme: "dark",
  timeFormat: "system",
  diffWrap: false,
  hideWhitespace: true,
  assistantOutput: true,
  newThreadLocation: "local",
}

const KEY = "hollow-prefs"

export function loadPrefs(): HollowPrefs {
  try {
    const v = localStorage.getItem(KEY)
    if (!v) return DEFAULT_PREFS
    const parsed = JSON.parse(v) as Partial<HollowPrefs>
    if (parsed.assistantOutput === false || parsed.assistantOutput === undefined) {
      parsed.assistantOutput = true
    }
    return { ...DEFAULT_PREFS, ...parsed }
  } catch {
    return DEFAULT_PREFS
  }
}

export function savePrefs(p: HollowPrefs): void {
  try {
    localStorage.setItem(KEY, JSON.stringify(p))
  } catch {
    /* ignore */
  }
}

export type PrefKey = keyof HollowPrefs