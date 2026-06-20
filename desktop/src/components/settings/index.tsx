import { ArrowLeft, X } from "lucide-react"
import type { AgentModel, CodexLoginState, ConnectionInfo } from "../../agent/rpc"
import { Appearance } from "./Appearance"
import { General } from "./General"
import { Providers, type ProvidersProps } from "./Providers"
import { SectionTitle } from "./ui"

export interface SettingsPageProps extends ProvidersProps {
  open: boolean
  onClose: () => void
}

export default function SettingsPage({
  open,
  onClose,
  models,
  currentModelId,
  connections,
  codexLogin,
  settingsError,
  onClearError,
  onSelectModel,
  onConnectKey,
  onRemoveKey,
  onStartCodexLogin,
  onCancelCodexLogin,
}: SettingsPageProps) {
  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-background text-foreground">
      {/* Header */}
      <header className="app-drag flex h-12 shrink-0 items-center justify-between border-b border-border px-4 select-none">
        <span className="text-[14px] font-semibold">Settings</span>
        <button
          onClick={onClose}
          className="app-no-drag flex h-6 w-6 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
          aria-label="Close settings"
        >
          <X className="h-4 w-4" strokeWidth={2.25} />
        </button>
      </header>

      {/* Two-column body. Left = Providers, right = Appearance + General. */}
      <div className="flex min-h-0 flex-1">
        <div className="flex w-[440px] shrink-0 flex-col overflow-y-auto border-r border-border px-6 py-5">
          <SectionTitle>Providers</SectionTitle>
          <p className="mt-1 mb-4 text-[11px] text-muted-foreground">
            Connect a provider to go live. OpenCode Go and Zen share one key.
          </p>
          <Providers
            models={models}
            currentModelId={currentModelId}
            connections={connections}
            codexLogin={codexLogin}
            settingsError={settingsError}
            onClearError={onClearError}
            onSelectModel={onSelectModel}
            onConnectKey={onConnectKey}
            onRemoveKey={onRemoveKey}
            onStartCodexLogin={onStartCodexLogin}
            onCancelCodexLogin={onCancelCodexLogin}
          />
        </div>

        <div className="flex min-w-0 flex-1 flex-col overflow-y-auto px-6 py-5">
          <Appearance />
          <div className="my-6 h-px bg-border/50" />
          <General />
        </div>
      </div>

      {/* Back button — pinned bottom-left */}
      <footer className="flex h-12 shrink-0 items-center border-t border-border px-4">
        <button
          onClick={onClose}
          className="flex items-center gap-2 rounded-lg px-3 py-1.5 text-[13px] font-medium text-muted-foreground transition-colors hover:bg-surface-hover hover:text-foreground"
        >
          <ArrowLeft className="h-4 w-4" strokeWidth={2} />
          Back
        </button>
      </footer>
    </div>
  )
}