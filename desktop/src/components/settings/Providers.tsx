import { useEffect, useState } from "react"
import type { AgentModel, CodexLoginState, ConnectionInfo } from "../../agent/rpc"
import { Card, FieldLabel, KeyCard, SettingsSelect } from "./ui"

// Provider ids (mirror backend/config/config.ts constants).
const OPENCODE = "opencode-go" // opencode + zen share this key slot
const NEURALWATT = "neuralwatt"
const CODEX = "openai-codex"

export interface ProvidersProps {
  models: AgentModel[]
  currentModelId: string | null
  connections: ConnectionInfo[]
  codexLogin: CodexLoginState | null
  settingsError: string | null
  onClearError: () => void
  onSelectModel: (m: AgentModel) => void
  onConnectKey: (provider: string, key: string) => void
  onRemoveKey: (provider: string) => void
  onStartCodexLogin: () => void
  onCancelCodexLogin: () => void
}

export function Providers({
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
}: ProvidersProps) {
  const currentModel = models.find((m) => m.id === currentModelId)
  const providers = Array.from(new Set(models.map((m) => m.provider))).sort()
  const [provider, setProvider] = useState<string>(currentModel?.provider ?? "")
  const [keys, setKeys] = useState<Record<string, string>>({})
  const [pending, setPending] = useState<string | null>(null)

  // Follow the active model's provider when it changes from outside.
  useEffect(() => {
    if (currentModel) setProvider(currentModel.provider)
  }, [currentModel?.provider])

  // A connection update or an error means the pending action resolved.
  useEffect(() => {
    setPending(null)
  }, [connections, settingsError])

  const conn = (id: string) => connections.find((c) => c.provider === id)
  const opencodeConnected = conn(OPENCODE)?.connected ?? false
  const neuralwattConnected = conn(NEURALWATT)?.connected ?? false
  const codexConnected = conn(CODEX)?.connected ?? false
  const anyConnected = connections.some((c) => c.connected)
  const providerModels = models.filter((m) => m.provider === provider)

  const connect = (p: string) => {
    setPending(p)
    onConnectKey(p, keys[p] ?? "")
    setKeys((k) => ({ ...k, [p]: "" }))
  }
  const disconnect = (p: string) => {
    setPending(p)
    onRemoveKey(p)
  }

  return (
    <div className="space-y-5">
      {settingsError && (
        <div className="flex items-start gap-2 rounded-lg border border-red-500/40 bg-red-500/10 p-2.5 text-[11px] text-red-300">
          <span className="flex-1">{settingsError}</span>
          <button onClick={onClearError} className="text-red-300/70 hover:text-red-200">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round">
              <path d="M18 6L6 18M6 6l12 12" />
            </svg>
          </button>
        </div>
      )}

      <KeyCard
        title="OpenCode Go + Zen"
        hint="Shared API key"
        connected={opencodeConnected}
        pending={pending === OPENCODE}
        keyValue={keys[OPENCODE] ?? ""}
        onKeyChange={(v) => setKeys((k) => ({ ...k, [OPENCODE]: v }))}
        onConnect={() => connect(OPENCODE)}
        onDisconnect={() => disconnect(OPENCODE)}
      />
      <KeyCard
        title="NeuralWatt"
        hint="API key"
        connected={neuralwattConnected}
        pending={pending === NEURALWATT}
        keyValue={keys[NEURALWATT] ?? ""}
        onKeyChange={(v) => setKeys((k) => ({ ...k, [NEURALWATT]: v }))}
        onConnect={() => connect(NEURALWATT)}
        onDisconnect={() => disconnect(NEURALWATT)}
      />

      {/* Codex uses OAuth device-code login, not a pasteable key. */}
      <Card>
        <div className="flex items-center justify-between">
          <div>
            <div className="text-xs font-semibold text-foreground">OpenAI Codex</div>
            <div className="text-[10px] text-muted-foreground">Sign in with browser</div>
          </div>
          <span
            className={
              codexConnected
                ? "rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-medium text-emerald-400"
                : "rounded-full bg-surface-hover px-2 py-0.5 text-[10px] font-medium text-muted-foreground"
            }
          >
            {codexConnected ? "Connected" : "Not connected"}
          </span>
        </div>
        {codexLogin ? (
          <div className="mt-3 space-y-2">
            <div className="text-[11px] text-muted-foreground">Open this URL and enter the code:</div>
            <a
              href={codexLogin.verify_url}
              target="_blank"
              rel="noreferrer"
              className="block break-all text-[11px] text-accent underline"
            >
              {codexLogin.verify_url}
            </a>
            <div className="select-all font-mono text-base tracking-[0.3em] text-foreground">
              {codexLogin.user_code}
            </div>
            <div className="text-[10px] text-muted-foreground">Waiting for browser sign-in…</div>
            <button
              onClick={onCancelCodexLogin}
              className="w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-foreground transition-colors hover:bg-surface-hover"
            >
              Cancel
            </button>
          </div>
        ) : codexConnected ? (
          <button
            onClick={() => disconnect(CODEX)}
            disabled={pending === CODEX}
            className="mt-3 w-full rounded-lg border border-border bg-surface px-3 py-2 text-xs text-foreground transition-colors hover:bg-surface-hover disabled:opacity-50"
          >
            {pending === CODEX ? "Disconnecting…" : "Disconnect"}
          </button>
        ) : (
          <button
            onClick={() => {
              setPending(CODEX)
              onStartCodexLogin()
            }}
            disabled={pending === CODEX}
            className="mt-3 w-full rounded-lg bg-accent px-3 py-2 text-xs font-medium text-white transition-colors hover:bg-accent/90 disabled:opacity-50"
          >
            {pending === CODEX ? "Starting…" : "Connect Codex"}
          </button>
        )}
      </Card>

      <div className="space-y-3 border-t border-border/50 pt-4">
        <div>
          <FieldLabel>Provider</FieldLabel>
          <SettingsSelect value={provider} disabled={!anyConnected} onChange={(e) => setProvider(e.target.value)}>
            {providers.map((p) => (
              <option key={p} value={p} className="bg-surface text-foreground">{p}</option>
            ))}
          </SettingsSelect>
          {!anyConnected && <p className="mt-1.5 text-[10px] text-muted-foreground">Connect a provider above to switch models.</p>}
        </div>

        <div>
          <FieldLabel>Model</FieldLabel>
          <SettingsSelect
            value={currentModel?.provider === provider ? currentModelId ?? "" : ""}
            disabled={!anyConnected}
            onChange={(e) => {
              const m = models.find((mm) => mm.id === e.target.value)
              if (m) onSelectModel(m)
            }}
          >
            {currentModel?.provider !== provider && (
              <option value="" className="bg-surface text-muted-foreground">Select a model…</option>
            )}
            {providerModels.map((m) => (
              <option key={m.id} value={m.id} className="bg-surface text-foreground">{m.name}</option>
            ))}
          </SettingsSelect>
        </div>
      </div>
    </div>
  )
}