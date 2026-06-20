import { memo } from "react"
import type { Message } from "../types"
import type { ModelCatalog } from "../agent/rpc"
import { ChatView } from "./chat-view"
import { EmptyState } from "./empty-state"
import { ModelPicker } from "./model-picker"
import { PromptInput } from "./prompt-input"

interface ChatWorkspaceProps {
  loadingThread: boolean
  messages: Message[]
  sessionId: string | null
  currentCwd: string
  modelCatalog: ModelCatalog | null
  isStreaming: boolean
  syncingThread: boolean
  onSend: (text: string) => void
  onAbort: () => void
  onSelectModel: (provider: string, modelId: string, thinkingLevel: string) => void
}

export const ChatWorkspace = memo(function ChatWorkspace({
  loadingThread,
  messages,
  sessionId,
  currentCwd,
  modelCatalog,
  isStreaming,
  syncingThread,
  onSend,
  onAbort,
  onSelectModel,
}: ChatWorkspaceProps) {
  const showEmpty = messages.length === 0
  const streaming = isStreaming || syncingThread

  const composer = (
    <div className="w-full space-y-1.5">
      <PromptInput onSend={onSend} isStreaming={isStreaming} onAbort={onAbort} />
      <ModelPicker
        catalog={modelCatalog}
        disabled={isStreaming}
        isStreaming={streaming}
        onSelect={onSelectModel}
      />
    </div>
  )

  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      {loadingThread ? (
        <div className="flex min-h-0 flex-1 items-center justify-center">
          <span className="block h-5 w-5 rounded-full border-2 border-muted-foreground/30 border-t-foreground animate-spin [animation-duration:0.9s]" />
        </div>
      ) : showEmpty ? (
        <EmptyState cwd={currentCwd} composer={composer} />
      ) : (
        <>
          <ChatView messages={messages} sessionId={sessionId} isStreaming={streaming} />
          <div className="absolute bottom-0 left-0 right-0 pointer-events-none bg-gradient-to-t from-background via-background/95 to-transparent pt-10 pb-4">
            <div className="w-full px-6 pointer-events-auto">{composer}</div>
          </div>
        </>
      )}
    </div>
  )
})
