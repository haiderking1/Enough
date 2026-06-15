import { memo, useCallback, useEffect, useLayoutEffect, useRef } from "react"
import type { Message } from "../types"
import { MarkdownContent } from "./markdown-content"
import { ToolBlock } from "./tool-block"
import { ThinkingBlock } from "./thinking-block"
import { TodoBlock } from "./todo-block"

const BOTTOM_THRESHOLD_PX = 80

interface ChatViewProps {
  messages: Message[]
  sessionId: string | null
  isStreaming: boolean
}

export const ChatView = memo(function ChatView({ messages, sessionId, isStreaming }: ChatViewProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const pinnedToBottomRef = useRef(true)
  const ignoreScrollRef = useRef(false)

  const isNearBottom = useCallback(() => {
    const el = scrollRef.current
    if (!el) return true
    return el.scrollHeight - el.scrollTop - el.clientHeight <= BOTTOM_THRESHOLD_PX
  }, [])

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    ignoreScrollRef.current = true
    el.scrollTop = el.scrollHeight
    requestAnimationFrame(() => {
      ignoreScrollRef.current = false
      pinnedToBottomRef.current = isNearBottom()
    })
  }, [isNearBottom])

  const handleScroll = useCallback(() => {
    if (ignoreScrollRef.current) return
    pinnedToBottomRef.current = isNearBottom()
  }, [isNearBottom])

  // New session — land at latest messages.
  useLayoutEffect(() => {
    pinnedToBottomRef.current = true
    scrollToBottom()
  }, [sessionId, scrollToBottom])

  // New tokens / messages — follow only while pinned; sending always pins.
  useLayoutEffect(() => {
    const last = messages[messages.length - 1]
    if (last?.role === "user") {
      pinnedToBottomRef.current = true
    }
    if (pinnedToBottomRef.current) {
      scrollToBottom()
    }
  }, [messages, isStreaming, scrollToBottom])

  // Markdown / tools grow after paint — keep up while pinned.
  useEffect(() => {
    const content = contentRef.current
    if (!content) return

    const ro = new ResizeObserver(() => {
      if (pinnedToBottomRef.current) {
        scrollToBottom()
      }
    })
    ro.observe(content)
    return () => ro.disconnect()
  }, [sessionId, scrollToBottom])

  return (
    <div ref={scrollRef} onScroll={handleScroll} className="min-h-0 flex-1 overflow-y-auto">
      <div ref={contentRef} className="w-full px-6 pt-6 pb-36">
        <div className="space-y-6">
          {messages.map((m) => (
            <MessageRow key={m.id} message={m} />
          ))}
        </div>
      </div>
    </div>
  )
})

const MessageRow = memo(function MessageRow({ message }: { message: Message }) {
  if (message.role === "user") {
    return (
      <div className="flex justify-end">
        <div className="max-w-[85%] rounded-2xl rounded-br-md bg-elevated px-4 py-2.5">
          <MarkdownContent id={message.id} text={message.text} className="text-foreground" />
        </div>
      </div>
    )
  }

  return (
    <div className="flex gap-3">
      <div className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-white" />
      <div className="min-w-0 flex-1 space-y-2">
        {message.blocks.map((block, i) => {
          const isLast = i === message.blocks.length - 1
          switch (block.type) {
            case "text":
              return (
                <MarkdownContent
                  key={i}
                  id={`${message.id}-text-${i}`}
                  text={block.text}
                  streaming={message.streaming && isLast && block.type === "text"}
                />
              )
            case "thinking":
              return (
                <ThinkingBlock
                  key={i}
                  id={`${message.id}-thinking-${i}`}
                  text={block.text}
                  streaming={message.streaming && isLast}
                />
              )
            case "tool":
              return <ToolBlock key={i} block={block} />
            case "todo":
              return <TodoBlock key={i} items={block.items} />
            default:
              return null
          }
        })}
      </div>
    </div>
  )
})
