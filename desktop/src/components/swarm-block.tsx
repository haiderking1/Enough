import { useEffect, useState } from "react"
import { ChevronRight } from "lucide-react"
import type { Block } from "../types"
import { cn } from "../lib/utils"

type ToolBlockType = Extract<Block, { type: "tool" }>

type SwarmAgent = {
  id: string
  status: "ok" | "error" | "aborted"
  turns: string
  worktree?: string
  body: string
}

type DoneState = {
  ok: number
  error: number
  aborted: number
  total: number
  goal: string
  agents: SwarmAgent[]
}

function parseProgress(output: string): { completed: number; total: number; planning: boolean } | null {
  let completed = 0
  let total = 0
  for (const l of output.split("\n")) {
    const m = l.match(/agent_swarm:\s*(\d+)\/(\d+)\s+agents finished/)
    if (m) {
      completed = parseInt(m[1], 10)
      total = parseInt(m[2], 10)
    }
  }
  const planning = /agent_swarm:\s*planning subtasks/.test(output)
  if (total > 0) return { completed, total, planning }
  if (planning) return { completed: 0, total: 0, planning: true }
  return null
}

function parseDone(output: string): DoneState {
  const res: DoneState = { ok: 0, error: 0, aborted: 0, total: 0, goal: "", agents: [] }
  const lines = output.split("\n")

  const goalLine = lines.find((l) => l.startsWith("Goal: "))
  if (goalLine) res.goal = goalLine.slice("Goal: ".length).trim()

  const header = lines.find((l) => l.startsWith("Ran ") && l.includes("agent(s)"))
  if (header) {
    const m = header.match(/Ran (\d+) agent\(s\).*—\s*(\d+)\s*ok(?:,\s*(\d+)\s*error)?(?:,\s*(\d+)\s*aborted)?/)
    if (m) {
      res.total = parseInt(m[1], 10)
      res.ok = parseInt(m[2], 10)
      res.error = parseInt(m[3] || "0", 10)
      res.aborted = parseInt(m[4] || "0", 10)
    }
  }

  const agents: SwarmAgent[] = []
  let cur: SwarmAgent | null = null
  const body: string[] = []
  for (const l of lines) {
    const m = l.match(/^## (.+?) \[(ok|error|aborted)\] \(([^)]+)\)(.*)$/)
    if (m) {
      if (cur) {
        cur.body = body.join("\n").trim()
        agents.push(cur)
      }
      const extra = m[4] || ""
      const wt = extra.match(/worktree:\s*(\S+)\s*·\s*branch:\s*([^\s)]+)/)
      cur = {
        id: m[1],
        status: m[2] as SwarmAgent["status"],
        turns: m[3],
        worktree: wt ? `${wt[1]} · ${wt[2]}` : undefined,
        body: "",
      }
      body.length = 0
    } else if (cur) {
      body.push(l)
    }
  }
  if (cur) {
    cur.body = body.join("\n").trim()
    agents.push(cur)
  }
  res.agents = agents
  if (res.total === 0 && agents.length > 0) res.total = agents.length
  return res
}

// Claude Code's braille spinner: ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
const BRAILLE = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"]

function useSpinner(active: boolean, offset = 0) {
  const [i, setI] = useState(offset % BRAILLE.length)
  useEffect(() => {
    if (!active) return
    const id = setInterval(() => setI((v) => (v + 1) % BRAILLE.length), 80)
    return () => clearInterval(id)
  }, [active])
  return active ? BRAILLE[i] : ""
}

type SlotStatus = "ok" | "error" | "aborted" | "running"

function statusLabel(s: SlotStatus): { text: string; cls: string } {
  if (s === "error") return { text: "Error", cls: "text-danger" }
  if (s === "aborted") return { text: "Aborted", cls: "text-muted-foreground" }
  if (s === "ok") return { text: "Done", cls: "text-muted-foreground" }
  return { text: "", cls: "text-info" }
}

// Leading circle per agent: hollow while running, filled green when done.
function AgentCircle({ status }: { status: SlotStatus }) {
  if (status === "ok") return <span className="size-2.5 shrink-0 rounded-full bg-success" />
  if (status === "error") return <span className="size-2.5 shrink-0 rounded-full bg-danger" />
  if (status === "aborted") return <span className="size-2.5 shrink-0 rounded-full bg-muted-foreground/50" />
  return <span className="size-2.5 shrink-0 rounded-full ring-1 ring-foreground/55" />
}

function AgentRow({
  id,
  status,
  turns,
  worktree,
  body,
  index,
}: {
  id: string
  status: SlotStatus
  turns?: string
  worktree?: string
  body?: string
  index: number
}) {
  const hasBody = Boolean(body && body.trim() !== "")
  const [open, setOpen] = useState(false)
  // Spinner removed per request. The hook call is kept inactive so hot-reload
  // doesn't trip on a hook-count change (HMR can't swap out a removed hook).
  // Safe to delete together with useSpinner/BRAILLE after a full window reload.
  void useSpinner(false, index)
  const label = statusLabel(status)
  return (
    <div>
      <button
        type="button"
        disabled={!hasBody}
        onClick={() => hasBody && setOpen((o) => !o)}
        className={cn(
          "flex w-full items-center gap-2 text-left font-mono text-[12.5px] leading-[1.55]",
          hasBody && "cursor-pointer transition-colors hover:opacity-90",
          !hasBody && "cursor-default",
        )}
      >
        <AgentCircle status={status} />
        <span className="shrink-0 text-foreground">{id}</span>
        {turns && <span className="shrink-0 text-muted-foreground">  ·  {turns}</span>}
        {worktree && (
          <span className="min-w-0 truncate text-muted-foreground/70" title={worktree}>
            {worktree}
          </span>
        )}
        {status !== "running" && <span className={cn("shrink-0", label.cls)}>{label.text}</span>}
        {hasBody && (
          <ChevronRight
            className={cn("h-3.5 w-3.5 shrink-0 text-muted-foreground/45 transition-transform", open && "rotate-90")}
            strokeWidth={2}
          />
        )}
      </button>
      {open && hasBody && (
        <pre
          className={cn(
            "ml-4 overflow-x-auto whitespace-pre-wrap border-l border-border/50 py-1 pl-4 font-mono text-[12px] leading-relaxed",
            status === "error" ? "text-danger/90" : "text-muted-foreground",
          )}
        >
          {body}
        </pre>
      )}
    </div>
  )
}

export function SwarmAgentBlock({ block }: { block: ToolBlockType }) {
  const running = block.status === "running"
  const output = block.output ?? ""

  const progress = running ? parseProgress(output) : null
  const done = !running ? parseDone(output) : null

  const total = progress?.total ?? done?.total ?? 0
  const completed = progress?.completed ?? done?.ok ?? 0
  const planning = progress?.planning ?? false

  const slots: { id: string; status: SlotStatus; turns?: string; worktree?: string; body?: string }[] = []
  if (done && done.agents.length > 0) {
    for (const a of done.agents) slots.push({ id: a.id, status: a.status, turns: a.turns, worktree: a.worktree, body: a.body })
  } else if (total > 0) {
    for (let i = 0; i < total; i++) slots.push({ id: `agent-${i + 1}`, status: i < completed ? "ok" : "running" })
  } else if (planning) {
    slots.push({ id: "planning", status: "running" })
  }

  return (
    <div className="font-mono text-[12.5px] leading-[1.55]">
      {/* header: blue ⏺ bullet + tool name */}
      <div className="flex items-center gap-2">
        <span className="inline-block size-2.5 shrink-0 rounded-full bg-info" />
        <span className="shrink-0 text-[13px] font-semibold text-foreground">Swarm Agent</span>
        {block.title && block.title !== "agent_swarm" && (
          <span className="min-w-0 truncate text-muted-foreground" title={block.title}>
            {block.title}
          </span>
        )}
        {total > 0 && <span className="shrink-0 tabular-nums text-muted-foreground">{completed}/{total}</span>}
      </div>

      {/* agents: circle + id + turns + status */}
      {slots.length > 0 && (
        <div className="mt-0.5">
          {done && done.goal !== "" && (
            <div className="flex items-center gap-2 py-0.5 pl-[18px] font-mono text-[12.5px]">
              <span className="text-muted-foreground/80">Goal:</span>
              <span className="min-w-0 truncate text-muted-foreground" title={done.goal}>
                {done.goal}
              </span>
            </div>
          )}
          {slots.map((s, i) => (
            <AgentRow key={i} index={i} {...s} />
          ))}
        </div>
      )}
    </div>
  )
}