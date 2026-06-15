import type {
  AgentEvent,
  AgentModel,
  AgentSessionInfo,
  AgentSessionState,
  ContentPart,
  RawMessage,
} from "./rpc"
import type { ToolVerb } from "../types"

type Listener = (event: AgentEvent) => void

interface BackendSession {
  id: string
  path?: string
  cwd?: string
  title: string
  createdAt: string
  created?: string
  modified?: string
  messageCount?: number
}

interface BackendHistoryTool {
  id: string
  name: string
  arguments: string
  status: "running" | "completed" | "failed"
  result?: string
}

interface BackendHistoryMessage {
  id: string
  role: "user" | "assistant" | "system"
  content: string
  timestamp: string
  tools?: BackendHistoryTool[]
}

type BackendMessage =
  | { type: "ready" }
  | { type: "session.list"; sessions: BackendSession[] | null }
  | { type: "session.history"; sessionId: string; messages: BackendHistoryMessage[] | null }
  | { type: "token"; text?: string }
  | {
      type: "tool"
      id: string
      name: string
      arguments: string
      status: "running" | "completed" | "failed"
      result?: string
    }
  | { type: "done" }
  | { type: "error"; message: string }

const WS_URL = import.meta.env.VITE_ENOUGH_WS || "ws://127.0.0.1:8754"
const DEFAULT_CWD = "Enough"

const model: AgentModel = {
  id: "enough",
  name: "Enough",
  provider: "local",
}

const nowIso = () => new Date().toISOString()

function commandText(command: Record<string, unknown>, key: string) {
  const value = command[key]
  return typeof value === "string" ? value : ""
}

function toolVerb(name: string): ToolVerb {
  const lower = name.toLowerCase()
  if (lower.includes("write")) return "Write"
  if (lower.includes("edit") || lower.includes("patch")) return "Edit"
  if (lower.includes("read") || lower.includes("file")) return "Read"
  if (lower.includes("grep")) return "Grep"
  if (lower.includes("search") || lower.includes("find")) return "Search"
  if (lower.includes("task") || lower.includes("agent")) return "Task"
  if (lower.includes("fetch")) return "Fetch"
  return "Bash"
}

function toolTitle(tool: BackendHistoryTool | (BackendMessage & { type: "tool" })) {
  try {
    const args = JSON.parse(tool.arguments || "{}")
    return (
      args.CommandLine ||
      args.command ||
      args.AbsolutePath ||
      args.TargetFile ||
      args.path ||
      args.filename ||
      tool.name
    )
  } catch {
    return tool.arguments || tool.name
  }
}

function mapTool(tool: BackendHistoryTool | (BackendMessage & { type: "tool" })): ContentPart {
  return {
    type: "tool",
    tool: toolVerb(tool.name),
    title: toolTitle(tool),
    status: tool.status === "failed" ? "error" : tool.status === "running" ? "running" : "done",
    output: tool.result,
  }
}

function mapHistory(messages: BackendHistoryMessage[] | null | undefined): RawMessage[] {
  return (messages ?? []).flatMap((message): RawMessage[] => {
    if (message.role === "system") {
      return [{ role: "assistant", content: [{ type: "text", text: message.content }] }]
    }

    const content: ContentPart[] = []
    if (message.content) content.push({ type: "text", text: message.content })
    for (const tool of message.tools ?? []) content.push(mapTool(tool))
    return [{ role: message.role, content }]
  })
}

function mapSession(session: BackendSession): AgentSessionInfo {
  const modified = session.modified || nowIso()
  const created = session.created || modified
  return {
    path: session.path || session.id,
    id: session.id,
    cwd: session.cwd || DEFAULT_CWD,
    name: session.title,
    created,
    modified,
    messageCount: session.messageCount ?? 0,
    firstMessage: session.title,
  }
}

class EnoughClient {
  private ws: WebSocket | null = null
  private listeners = new Set<Listener>()
  private connectStarted = false
  private reconnectTimer: number | null = null
  private sessions: AgentSessionInfo[] = []
  private histories = new Map<string, RawMessage[]>()
  private currentSessionId: string | null = null
  private streaming = false
  private streamText = ""
  private streamTools = new Map<string, ContentPart>()

  onEvent(listener: Listener) {
    this.listeners.add(listener)
    this.connect()
    return () => {
      this.listeners.delete(listener)
    }
  }

  send(command: Record<string, unknown>) {
    this.connect()

    switch (command.type) {
      case "get_available_models":
        this.emit({ type: "response", command: "get_available_models", success: true, data: { models: [model] } })
        break
      case "get_state":
        this.emit({ type: "response", command: "get_state", success: true, data: this.state() })
        break
      case "get_messages":
        this.emit({
          type: "response",
          command: "get_messages",
          success: true,
          data: { messages: this.currentSessionId ? this.histories.get(this.currentSessionId) ?? [] : [] },
        })
        break
      case "list_sessions":
        this.sendWs({ type: "listSessions" })
        break
      case "switch_session": {
        const id = commandText(command, "sessionPath")
        if (id) {
          this.currentSessionId = id
          this.sendWs({ type: "openSession", id })
        }
        this.emit({ type: "response", command: "switch_session", success: true, data: { cancelled: false } })
        break
      }
      case "new_session": {
        this.sendWs({ type: "newSession" })
        this.emit({ type: "response", command: "new_session", success: true, data: { cancelled: false } })
        break
      }
      case "prompt":
        this.streaming = true
        this.streamText = ""
        this.streamTools.clear()
        this.sendWs({ type: "prompt", text: commandText(command, "message") })
        this.emit({ type: "response", command: "get_state", success: true, data: this.state() })
        break
      case "abort":
        this.sendWs({ type: "interrupt" })
        this.streaming = false
        this.emit({ type: "agent_end" })
        break
      case "set_model":
        this.emit({ type: "response", command: "set_model", success: true, data: model })
        break
    }
  }

  private connect() {
    if (this.connectStarted) return
    this.connectStarted = true
    this.openSocket()
  }

  private openSocket() {
    this.ws = new WebSocket(WS_URL)

    this.ws.onopen = () => {
      this.emit({ type: "bridge_ready" })
      this.sendWs({ type: "listSessions" })
    }

    this.ws.onmessage = (event) => {
      try {
        this.handleBackendMessage(JSON.parse(event.data) as BackendMessage)
      } catch (error) {
        this.emit({ type: "bridge_error", error: String(error) })
      }
    }

    this.ws.onclose = () => {
      this.ws = null
      this.connectStarted = false
      this.emit({ type: "bridge_exit", code: null })
      if (this.reconnectTimer === null) {
        this.reconnectTimer = window.setTimeout(() => {
          this.reconnectTimer = null
          this.connect()
        }, 1000)
      }
    }

    this.ws.onerror = () => {
      this.emit({ type: "bridge_error", error: `Cannot connect to ${WS_URL}` })
    }
  }

  private handleBackendMessage(message: BackendMessage) {
    switch (message.type) {
      case "ready":
        break
      case "session.list": {
        this.sessions = (message.sessions ?? []).map(mapSession)
        const first = this.sessions[0]
        if (!this.currentSessionId && first) {
          this.currentSessionId = first.id
          this.sendWs({ type: "openSession", id: first.id })
        }
        this.emit({ type: "response", command: "list_sessions", success: true, data: { sessions: this.sessions } })
        this.emit({ type: "response", command: "get_state", success: true, data: this.state() })
        break
      }
      case "session.history": {
        this.currentSessionId = message.sessionId
        const history = mapHistory(message.messages)
        this.histories.set(message.sessionId, history)
        this.emit({ type: "response", command: "get_state", success: true, data: this.state() })
        this.emit({ type: "response", command: "get_messages", success: true, data: { messages: history } })
        break
      }
      case "token":
        this.streamText += message.text ?? ""
        this.emitAssistantUpdate()
        break
      case "tool":
        this.streamTools.set(message.id, mapTool(message))
        this.emitAssistantUpdate()
        break
      case "done":
        this.streaming = false
        this.emit({ type: "agent_end" })
        this.sendWs({ type: "listSessions" })
        break
      case "error":
        this.streaming = false
        this.emit({ type: "bridge_error", error: message.message })
        this.emit({ type: "agent_end" })
        break
    }
  }

  private emitAssistantUpdate() {
    const content: ContentPart[] = []
    if (this.streamText) content.push({ type: "text", text: this.streamText })
    content.push(...this.streamTools.values())
    this.emit({
      type: "message_update",
      assistantMessageEvent: {
        partial: {
          role: "assistant",
          content,
        },
      },
    })
  }

  private state(): AgentSessionState {
    return {
      model,
      sessionId: this.currentSessionId ?? "",
      isStreaming: this.streaming,
      messageCount: this.currentSessionId ? this.histories.get(this.currentSessionId)?.length ?? 0 : 0,
    }
  }

  private sendWs(message: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    }
  }

  private emit(event: AgentEvent) {
    for (const listener of this.listeners) listener(event)
  }
}

export const enoughAgent = new EnoughClient()
