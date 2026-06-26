import { Agent } from "./agent";
import { new_client_for_runtime } from "../opencode/runtime_client";
import { type runtime } from "../config/config";
import { type message, type chat_request, string_content, content_string } from "../opencode/types";
import { parse_thinking_level, apply_thinking_to_request } from "../opencode/thinking";
import { Effect } from "effect";

export interface WorkflowAgentOptions {
  Prompt: string;
  SystemPrompt: string;
  Tools: string[];
  Model: string;
  MaxTurns: number;
  Readonly: boolean;
}

export interface WorkflowAgentResult {
  Text: string;
  Error: string;
  TokensUsed: number;
  TurnCount: number;
}

export async function RunWorkflowAgent(
  ctx: AbortSignal,
  cfg: runtime,
  workDir: string,
  opts: WorkflowAgentOptions,
  emit: (event: any) => void
): Promise<WorkflowAgentResult> {
  const childCfg = { ...cfg };
  if (opts.Model && opts.Model.trim() !== "") {
    childCfg.model = opts.Model.trim();
  }

  const allowed = workflowAllowedTools(opts.Tools, opts.Readonly);

  const a = new Agent();
  a.cfg = childCfg;
  a.client = new_client_for_runtime(childCfg);
  a.workDir = workDir;
  a.emit = emit;
  a.swarmDepth = 1;
  a.allowedTools = allowed;
  a.readonlyRole = opts.Readonly;
  a.toolStart = a.toolStart || Agent.prototype.toolStart;
  a.toolDelta = a.toolDelta || Agent.prototype.toolDelta;
  a.toolResult = a.toolResult || Agent.prototype.toolResult;
  a.executeTool = a.executeTool || Agent.prototype.executeTool;

  const defaultSystemPrompt = "You are a helpful assistant."; // fallback description
  let system = (opts.SystemPrompt || "").trim();
  if (system === "") {
    system = defaultSystemPrompt;
  }

  const messages: message[] = [
    { role: "system", content: string_content(system) },
    { role: "user", content: string_content(opts.Prompt) },
  ];

  const tools = a.toolMenu();
  let totalTokens = 0;
  let turns = 0;

  while (true) {
    if (ctx.aborted) {
      return {
        Text: extractLastAssistantText(messages),
        Error: "aborted",
        TokensUsed: totalTokens,
        TurnCount: turns,
      };
    }

    const req: chat_request = {
      model: childCfg.model,
      messages: messages,
      tools: tools,
    };

    apply_thinking_to_request(req, parse_thinking_level(childCfg.thinking_level || ""), childCfg.model);

    let msg: message;
    try {
      msg = await Effect.runPromise(
        a.client.chat_stream_retry(ctx, req, {})
      );
    } catch (err: any) {
      return {
        Text: extractLastAssistantText(messages),
        Error: err.message || String(err),
        TokensUsed: totalTokens,
        TurnCount: turns,
      };
    }

    turns++;

    if (msg.usage) {
      const usageTotal = msg.usage.totalTokens ?? 0;
      if (usageTotal > 0) {
        totalTokens += usageTotal;
      } else {
        totalTokens += (msg.usage.input || 0) + (msg.usage.output || 0);
      }
    }

    messages.push(msg);

    if (!msg.tool_calls || msg.tool_calls.length === 0) {
      return {
        Text: content_string(msg),
        Error: "",
        TokensUsed: totalTokens,
        TurnCount: turns,
      };
    }

    for (let idx = 0; idx < msg.tool_calls.length; idx++) {
      if (ctx.aborted) {
        return {
          Text: extractLastAssistantText(messages),
          Error: "aborted",
          TokensUsed: totalTokens,
          TurnCount: turns,
        };
      }
      const call = msg.tool_calls[idx];
      let id = call.id;
      if (id === "") {
        id = `workflow_call_${idx}`;
      }
      if (a.toolStart) {
        a.toolStart(id, call.function.name, call.function.arguments);
      }

      let result: any;
      try {
        result = await Effect.runPromise(
          a.executeTool(ctx, id, call.function.name, call.function.arguments)
        );
      } catch (err: any) {
        result = { output: err.message || String(err), isErr: true };
      }

      if (a.toolResult) {
        a.toolResult(id, result.output, result.isErr, result.details);
      }

      messages.push({
        role: "tool",
        tool_call_id: id,
        name: call.function.name,
        content: string_content(result.output),
      });
    }
  }
}

export function workflowAllowedTools(requested: string[], readonly: boolean): Record<string, boolean> {
  const allowed: Record<string, boolean> = {};
  if (!requested || requested.length === 0) {
    const list = ["read_file", "list_dir", "glob", "grep", "bash", "web_search", "web_fetch", "browser"];
    for (const name of list) {
      allowed[name] = true;
    }
    if (!readonly) {
      allowed["write_file"] = true;
      allowed["edit_file"] = true;
    }
    return allowed;
  }
  for (let name of requested) {
    name = (name || "").trim();
    if (name === "" || name === "agent_swarm") {
      continue;
    }
    if (readonly && (name === "write_file" || name === "edit_file" || name === "skill_manage" || name === "memory")) {
      continue;
    }
    allowed[name] = true;
  }
  return allowed;
}

function extractLastAssistantText(messages: message[]): string {
  for (let i = messages.length - 1; i >= 0; i--) {
    const m = messages[i];
    if (m.role === "assistant" && (!m.tool_calls || m.tool_calls.length === 0)) {
      return content_string(m);
    }
  }
  return "";
}

