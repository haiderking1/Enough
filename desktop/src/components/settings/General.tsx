import { Card, FieldLabel, SectionTitle } from "./ui"

const SHORTCUTS: { keys: string; label: string }[] = [
  { keys: "Ctrl + K", label: "Toggle search" },
  { keys: "Ctrl + =", label: "Zoom in" },
  { keys: "Ctrl + −", label: "Zoom out" },
  { keys: "Ctrl + 0", label: "Reset zoom" },
]

export function General() {
  return (
    <div className="space-y-5">
      <SectionTitle>General</SectionTitle>

      <Card>
        <div className="space-y-2">
          <div className="flex justify-between text-xs">
            <span className="text-muted-foreground">Version</span>
            <span className="font-medium text-foreground">0.1.0</span>
          </div>
          <div className="flex justify-between text-xs">
            <span className="text-muted-foreground">Core</span>
            <span className="font-medium text-foreground">Electron</span>
          </div>
          <div className="flex justify-between text-xs">
            <span className="text-muted-foreground">Framework</span>
            <span className="font-medium text-foreground">React 19</span>
          </div>
          <div className="flex justify-between text-xs">
            <span className="text-muted-foreground">Runtime</span>
            <span className="font-medium text-foreground">Bun · effect</span>
          </div>
        </div>
      </Card>

      <div>
        <FieldLabel>Keyboard shortcuts</FieldLabel>
        <Card className="space-y-1.5">
          {SHORTCUTS.map((s) => (
            <div key={s.label} className="flex items-center justify-between text-xs">
              <span className="text-muted-foreground">{s.label}</span>
              <kbd className="rounded border border-border-strong bg-surface px-1.5 py-0.5 font-mono text-[10px] text-foreground">
                {s.keys}
              </kbd>
            </div>
          ))}
        </Card>
      </div>
    </div>
  )
}