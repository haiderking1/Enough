import { useEffect, useMemo, useRef, useState } from "react"
import { Check, ChevronDown, Plus, Search } from "lucide-react"
import type { AgentModel, ModelCatalog } from "../agent/rpc"
import { formatThinkingBadge } from "../lib/thinking"

interface ModelPickerProps {
  catalog: ModelCatalog | null
  disabled?: boolean
  onSelect: (provider: string, modelId: string, thinkingLevel: string) => void
  onRefreshCatalog?: () => void
}

// Cursor-style model picker: compact popover with search, toggles, active model
// row with a checkmark, and an "Add Models" row.
const C = {
  muted: "#6e6e78",
  panelBg: "#1c1c1e",
  panelBorder: "rgba(255,255,255,0.08)",
  hoverBg: "rgba(255,255,255,0.05)",
  toggleBg: "#2a2a2d",
  toggleKnob: "#ffffff",
}

function speedLabel(model: AgentModel, level: string): string {
  const badge = formatThinkingBadge(model, level)
  const tiers: Record<string, string> = {
    off: "Fast",
    minimal: "Fast",
    low: "Fast",
    medium: "Balanced",
    high: "Slow",
    xhigh: "Slow",
    max: "Slow",
  }
  return tiers[badge.toLowerCase()] ?? "Fast"
}

function labelFor(model: AgentModel, level: string): string {
  return `${model.name} ${speedLabel(model, level)}`
}

export function ModelPicker({ catalog, disabled, onSelect, onRefreshCatalog }: ModelPickerProps) {
  const state = catalog?.state
  const providers = catalog?.providers ?? []
  const models = catalog?.models ?? []

  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState("")
  const [maxMode, setMaxMode] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)

  const activeModel = useMemo(
    () => models.find((m) => m.id === state?.modelId && m.provider === state?.provider),
    [models, state?.modelId, state?.provider],
  )

  const activeLabel = useMemo(() => {
    if (!activeModel) return state?.modelName ?? "Model"
    return labelFor(activeModel, state?.thinkingLevel ?? "")
  }, [activeModel, state?.modelName, state?.thinkingLevel])

  const filteredModels = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return models
    return models.filter(
      (m) =>
        m.name.toLowerCase().includes(q) ||
        m.id.toLowerCase().includes(q) ||
        m.provider.toLowerCase().includes(q),
    )
  }, [models, query])

  useEffect(() => {
    if (!open) return
    onRefreshCatalog?.()
    const t = window.setTimeout(() => searchRef.current?.focus(), 30)
    return () => window.clearTimeout(t)
  }, [open, onRefreshCatalog])

  useEffect(() => {
    if (!open) return
    const onDoc = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false)
    }
    document.addEventListener("mousedown", onDoc)
    document.addEventListener("keydown", onKey)
    return () => {
      document.removeEventListener("mousedown", onDoc)
      document.removeEventListener("keydown", onKey)
    }
  }, [open])

  const select = (model: AgentModel) => {
    const level = model.thinkingLevels?.length
      ? maxMode
        ? "max"
        : model.thinkingLevels.includes("medium")
          ? "medium"
          : model.thinkingLevels.find((l) => l !== "off") ?? model.thinkingLevels[0]
      : ""
    onSelect(model.provider, model.id, level)
    setOpen(false)
    setQuery("")
  }

  // Loading fallback: plain trigger text.
  if (!catalog || providers.length === 0) {
    return (
      <button
        type="button"
        disabled
        className="inline-flex items-center gap-1 text-[13px] font-medium leading-none"
        style={{ color: C.muted }}
      >
        <span>Model Fast</span>
        <ChevronDown size={12} strokeWidth={2} style={{ color: C.muted }} />
      </button>
    )
  }

  return (
    <div ref={rootRef} className="relative shrink-0">
      {/* Inline trigger. */}
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-1 text-[13px] font-medium leading-none transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
        style={{ color: C.muted }}
      >
        <span className="truncate">{activeLabel}</span>
        <ChevronDown
          size={12}
          strokeWidth={2}
          style={{ color: C.muted }}
          className={open ? "rotate-180" : ""}
        />
      </button>

      {open && (
        <div
          className="absolute bottom-full right-0 z-50 mb-2 w-[260px] overflow-hidden rounded-xl p-2 shadow-2xl"
          style={{ background: C.panelBg, border: `1px solid ${C.panelBorder}` }}
        >
          {/* Search models. */}
          <div className="flex items-center gap-2 px-2 py-2">
            <Search size={14} strokeWidth={2} style={{ color: C.muted }} />
            <input
              ref={searchRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search models"
              className="min-w-0 flex-1 bg-transparent text-[13px] text-foreground placeholder:text-muted-foreground/60 focus:outline-none"
            />
          </div>

          {/* MAX Mode toggle. */}
          <Row
            label="MAX Mode"
            right={
              <Toggle
                checked={maxMode}
                onChange={() => {
                  setMaxMode((v) => !v)
                }}
              />
            }
          />

          <div className="my-1 h-px" style={{ background: "rgba(255,255,255,0.06)" }} />

          {/* Active / selectable model rows. */}
          {filteredModels.slice(0, 6).map((model) => {
            const isActive = model.id === state?.modelId && model.provider === state?.provider
            return (
              <button
                key={`${model.provider}:${model.id}`}
                type="button"
                onClick={() => select(model)}
                className="flex w-full items-center justify-between rounded-lg px-2 py-2 text-left text-[13px] transition-colors"
                style={{ color: isActive ? C.muted : "#b8b8be" }}
                onMouseEnter={(e) => {
                  (e.currentTarget as HTMLElement).style.background = C.hoverBg
                }}
                onMouseLeave={(e) => {
                  (e.currentTarget as HTMLElement).style.background = "transparent"
                }}
              >
                <span className="font-medium">{labelFor(model, state?.thinkingLevel ?? "")}</span>
                {isActive ? (
                  <span className="flex items-center gap-1 text-[11px]" style={{ color: C.muted }}>
                    Edit
                    <Check size={12} strokeWidth={2.5} style={{ color: C.muted }} />
                  </span>
                ) : null}
              </button>
            )
          })}

          {filteredModels.length === 0 && (
            <div className="px-2 py-3 text-[12px]" style={{ color: C.muted }}>
              No models found
            </div>
          )}

          <div className="my-1 h-px" style={{ background: "rgba(255,255,255,0.06)" }} />

          {/* Add Models row. */}
          <button
            type="button"
            className="flex w-full items-center gap-2 rounded-lg px-2 py-2 text-left text-[13px] transition-colors"
            style={{ color: "#b8b8be" }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.background = C.hoverBg
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLElement).style.background = "transparent"
            }}
          >
            <Plus size={14} strokeWidth={2} style={{ color: C.muted }} />
            <span className="font-medium">Add Models</span>
          </button>
        </div>
      )}
    </div>
  )
}

function Row({ label, right }: { label: string; right: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between px-2 py-2">
      <span className="text-[13px] font-medium" style={{ color: "#b8b8be" }}>{label}</span>
      {right}
    </div>
  )
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: () => void }) {
  return (
    <button
      type="button"
      onClick={onChange}
      className="relative h-4 w-7 rounded-full transition-colors"
      style={{ background: checked ? "#4b4b50" : C.toggleBg }}
      aria-checked={checked}
      role="switch"
    >
      <span
        className="absolute top-0.5 left-0.5 h-3 w-3 rounded-full transition-transform"
        style={{
          background: C.toggleKnob,
          transform: checked ? "translateX(12px)" : "translateX(0)",
        }}
      />
    </button>
  )
}
