import { useState } from "react"
import { cn } from "../../lib/utils"
import { bumpZoom, resetZoom, ZOOM_STEP } from "../../lib/zoom"
import { Card, FieldLabel, SectionTitle } from "./ui"
import { THEMES, getSavedTheme, setTheme, type ThemeId } from "./themes"

const ZOOM_MIN = 0.5
const ZOOM_MAX = 2.5

export function Appearance() {
  const [theme, setThemeState] = useState<ThemeId>(getSavedTheme())
  const [zoom, setZoom] = useState<number>(() => {
    try {
      const v = parseFloat(localStorage.getItem("enough-zoom") ?? "1")
      return Number.isFinite(v) ? Math.min(Math.max(v, ZOOM_MIN), ZOOM_MAX) : 1
    } catch {
      return 1
    }
  })

  const pickTheme = (id: ThemeId) => {
    setTheme(id)
    setThemeState(id)
  }

  const changeZoom = (delta: number) => setZoom(bumpZoom(delta))
  const reset = () => setZoom(resetZoom())

  return (
    <div className="space-y-5">
      <SectionTitle>Appearance</SectionTitle>

      <div>
        <FieldLabel>Theme</FieldLabel>
        <div className="grid grid-cols-2 gap-3">
          {THEMES.map((t) => (
            <button
              key={t.id}
              onClick={() => pickTheme(t.id)}
              className={cn(
                "group flex items-center gap-3 rounded-xl border p-3 text-left transition-colors",
                theme === t.id
                  ? "border-accent bg-surface-hover"
                  : "border-border bg-surface hover:bg-surface-hover",
              )}
            >
              <span
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border"
                style={{ background: t.swatch.bg, borderColor: t.swatch.border }}
              >
                <span className="text-[13px] font-bold" style={{ color: t.swatch.fg }}>A</span>
              </span>
              <div className="min-w-0">
                <div className="text-xs font-semibold text-foreground">{t.name}</div>
                <div className="mt-0.5 flex gap-1">
                  <span className="h-2 w-2 rounded-full" style={{ background: t.swatch.accent }} />
                  <span className="h-2 w-2 rounded-full" style={{ background: t.swatch.fg }} />
                  <span className="h-2 w-2 rounded-full" style={{ background: t.swatch.border }} />
                </div>
              </div>
              {theme === t.id && (
                <svg className="ml-auto h-4 w-4 shrink-0 text-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M20 6L9 17l-5-5" />
                </svg>
              )}
            </button>
          ))}
        </div>
      </div>

      <Card>
        <FieldLabel>Interface zoom</FieldLabel>
        <div className="flex items-center gap-3">
          <button
            onClick={() => changeZoom(-ZOOM_STEP)}
            disabled={zoom <= ZOOM_MIN + 1e-3}
            className="flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-surface text-foreground transition-colors hover:bg-surface-hover disabled:opacity-40"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
              <path d="M5 12h14" />
            </svg>
          </button>
          <div className="w-16 text-center font-mono text-xs text-foreground">{Math.round(zoom * 100)}%</div>
          <button
            onClick={() => changeZoom(ZOOM_STEP)}
            disabled={zoom >= ZOOM_MAX - 1e-3}
            className="flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-surface text-foreground transition-colors hover:bg-surface-hover disabled:opacity-40"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round">
              <path d="M12 5v14M5 12h14" />
            </svg>
          </button>
          <button
            onClick={reset}
            className="ml-auto rounded-lg border border-border bg-surface px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
          >
            Reset
          </button>
        </div>
        <p className="mt-2 text-[10px] text-muted-foreground">
          Shortcut: Ctrl/Cmd + = / − / 0
        </p>
      </Card>
    </div>
  )
}