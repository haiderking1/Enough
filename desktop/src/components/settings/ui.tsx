import { cn } from "../../lib/utils"

/** Native select styled to match the app, with a custom chevron. */
export function SettingsSelect({ children, ...props }: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <div className="relative flex w-full items-center">
      <select
        {...props}
        className="w-full cursor-pointer appearance-none rounded-lg border border-border bg-surface px-3 py-2 text-xs text-foreground outline-none transition-colors hover:border-border-strong hover:bg-surface-hover focus-visible:border-accent disabled:opacity-50"
      >
        {children}
      </select>
      <div className="pointer-events-none absolute right-3 flex items-center text-muted-foreground">
        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
          <path d="M6 9l6 6 6-6" />
        </svg>
      </div>
    </div>
  )
}

export function FieldLabel({ children }: { children: React.ReactNode }) {
  return (
    <label className="mb-2 block text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </label>
  )
}

/** Bold column / section heading. */
export function SectionTitle({ children }: { children: React.ReactNode }) {
  return <h2 className="text-[13px] font-semibold tracking-wide text-foreground">{children}</h2>
}

export function StatusBadge({ connected }: { connected: boolean }) {
  return (
    <span
      className={cn(
        "rounded-full px-2 py-0.5 text-[10px] font-medium",
        connected ? "bg-emerald-500/15 text-emerald-400" : "bg-surface-hover text-muted-foreground",
      )}
    >
      {connected ? "Connected" : "Not connected"}
    </span>
  )
}

/** A bordered surface card used to group settings within a column. */
export function Card({ className, children }: { className?: string; children: React.ReactNode }) {
  return (
    <div className={cn("rounded-xl border border-border bg-surface p-4", className)}>{children}</div>
  )
}

export interface KeyCardProps {
  title: string
  hint: string
  connected: boolean
  pending: boolean
  keyValue: string
  onKeyChange: (v: string) => void
  onConnect: () => void
  onDisconnect: () => void
}

export function KeyCard({
  title,
  hint,
  connected,
  pending,
  keyValue,
  onKeyChange,
  onConnect,
  onDisconnect,
}: KeyCardProps) {
  return (
    <Card>
      <div className="flex items-center justify-between">
        <div>
          <div className="text-xs font-semibold text-foreground">{title}</div>
          <div className="text-[10px] text-muted-foreground">{hint}</div>
        </div>
        <StatusBadge connected={connected} />
      </div>
      {connected ? (
        <button
          onClick={onDisconnect}
          disabled={pending}
          className="mt-3 w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-foreground transition-colors hover:bg-surface-hover disabled:opacity-50"
        >
          {pending ? "Disconnecting…" : "Disconnect"}
        </button>
      ) : (
        <div className="mt-3 flex gap-2">
          <input
            type="password"
            value={keyValue}
            onChange={(e) => onKeyChange(e.target.value)}
            placeholder="Paste API key"
            autoComplete="off"
            spellCheck={false}
            className="min-w-0 flex-1 rounded-lg border border-border bg-surface px-3 py-2 text-xs text-foreground outline-none focus-visible:border-accent"
          />
          <button
            onClick={onConnect}
            disabled={pending || keyValue.trim() === ""}
            className="shrink-0 rounded-lg bg-accent px-3 py-2 text-xs font-medium text-white transition-colors hover:bg-accent/90 disabled:opacity-50"
          >
            {pending ? "Connecting…" : "Connect"}
          </button>
        </div>
      )}
    </Card>
  )
}