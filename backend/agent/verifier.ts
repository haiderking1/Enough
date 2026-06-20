import { Agent, type toolResult } from "./agent";
import { verifierTools } from "./tools_registry";
import { Effect } from "effect";
import { kind_verifier_pass, kind_verifier_fail } from "./evidence/ledger";
import { type message, type chat_request, string_content, content_string } from "../opencode/types";
import { type obligation } from "./obligations/registry";

// maxVerifierToolRounds caps how many tool-calling rounds the verifier gets
// before it is forced to report.
export const maxVerifierToolRounds = 4;

// verifierAllowedTools is the verifier's hard tool allowlist.
export const verifierAllowedTools: Record<string, boolean> = {
  "read_file": true,
  "list_dir": true,
  "glob": true,
  "grep": true,
  "bash": true,
};

export const verifierSystemPrompt = `You are a verification agent. You cannot edit code. Your only job is to check, with commands, whether the worker's changes satisfy the task.

Your FIRST tool call must be bash running the verification command you were given. Optionally inspect changed files with read_file/grep after. You have very few tool calls — do not browse.

Then respond with ONLY a JSON object, no other text:
{"pass": true|false, "command_runs": [{"cmd": "...", "exit_code": 0}], "failures": ["specific, factual failure"]}

Rules:
- pass=true only if the verification command exited 0 AND the changes address the task.
- failures must be factual: raw error lines, failing test names, missing behavior. No advice, no coaching.`;

export const verifierForceReport = `Output the JSON report now. No more tool calls are available. Respond with ONLY the JSON object.`;

export interface verifierReport {
  pass: boolean;
  failures: string[];
}

// runVerifier executes the verifier role once and returns factual failure strings.
export async function runVerifier(this: Agent, ctx: AbortSignal): Promise<string[]> {
  const reg = this.obligations;
  const ledger = this.ledger;
  let task = this.lockedGoal;
  if (task === "") {
    task = this.lastUserPrompt;
  }
  if (reg === null || ledger === null) {
    return [];
  }

  const verifier = new Agent();
  verifier.cfg = this.cfg;
  verifier.client = this.client;
  verifier.workDir = this.workDir;
  verifier.ledger = ledger;
  verifier.obligations = reg;
  verifier.allowedTools = verifierAllowedTools;
  verifier.emit = null;
  verifier.toolStart = () => {};
  verifier.toolDelta = () => {};
  verifier.toolResult = () => {};
  verifier.executeTool = this.executeTool;

  let report: verifierReport;
  let err: any = null;
  try {
    report = await verifierLoop.call(verifier, ctx, task);
  } catch (e) {
    report = { pass: false, failures: [] };
    err = e;
  }

  const failures = [...report.failures];
  if (err !== null) {
    failures.push(`verifier error: ${err.message || String(err)}`);
  }

  const pass = err === null && report.pass && reg.verify_closed();
  if (pass) {
    try {
      const entry = Effect.runSync(
        ledger.append(kind_verifier_pass, { turn_id: ledger.turn_id() })
      );
      if (reg.note_verifier_pass(entry.id)) {
        this.emitObligations();
      }
    } catch {}
    return [];
  }

  if (err === null && report.pass && !reg.verify_closed()) {
    failures.push("verifier claimed pass but no passing verification run is recorded in the evidence ledger");
  }

  try {
    Effect.runSync(
      ledger.append(kind_verifier_fail, { turn_id: ledger.turn_id(), failures })
    );
  } catch {}

  this.emitObligations();
  return failures;
}

Agent.prototype.runVerifier = runVerifier;

// verifierLoop runs the restricted agent until it emits its JSON report.
export async function verifierLoop(this: Agent, ctx: AbortSignal, task: string): Promise<verifierReport> {
  const reg = this.obligations;
  if (!reg || !this.ledger) {
    throw new Error("uninitialized verifier state");
  }

  let promptBuilder = `User task:\n${task}\n\n`;
  const cmd = reg.verify_command();
  if (cmd !== "") {
    promptBuilder += `Verification command: ${cmd}\n`;
  } else {
    promptBuilder += `Verification command: none detected — run the most appropriate explicit check via bash.\n`;
  }
  const extras = reg.extra_verify_commands();
  if (extras.length > 0) {
    promptBuilder += `Task-specific checks from the user:\n`;
    for (const c of extras) {
      promptBuilder += `- ${c}\n`;
    }
  }
  const paths = this.ledger.mutated_paths();
  if (paths.length > 0) {
    promptBuilder += `Files changed this turn:\n${paths.join("\n")}\n`;
  }
  promptBuilder += `\nOpen obligations:\n`;
  for (const ob of reg.open()) {
    promptBuilder += `- ${ob.kind}: ${ob.description}\n`;
  }

  const messages: message[] = [
    { role: "system", content: string_content(verifierSystemPrompt) },
    { role: "user", content: string_content(promptBuilder) },
  ];

  for (let round = 0; round < maxVerifierToolRounds; round++) {
    if (ctx.aborted) {
      throw new Error("aborted");
    }

    const req: chat_request = {
      model: this.cfg.model,
      messages: messages,
      tools: verifierTools(),
    };

    const msg = (await Effect.runPromise(
      this.client.chat_stream_retry(ctx, req, {})
    )) as message;
    messages.push(msg);

    if (!msg.tool_calls || msg.tool_calls.length === 0) {
      return parseVerifierReport(content_string(msg));
    }

    for (let idx = 0; idx < msg.tool_calls.length; idx++) {
      const call = msg.tool_calls[idx];
      let id = call.id;
      if (id === "") {
        id = `verifier_call_${idx}`;
      }
      if (this.toolStart) {
        this.toolStart(id, call.function.name, call.function.arguments);
      }
      let result: toolResult;
      try {
        result = await Effect.runPromise(
          this.executeTool(ctx, id, call.function.name, call.function.arguments)
        );
      } catch (err: any) {
        result = { output: err.message || String(err), isErr: true };
      }
      if (this.toolResult) {
        this.toolResult(id, result.output, result.isErr, result.details);
      }

      let toolMsg: message;
      if (result.content && result.content.length > 0) {
        toolMsg = {
          role: "tool",
          tool_call_id: id,
          name: call.function.name,
          content: new TextEncoder().encode(JSON.stringify(result.content)),
        };
      } else {
        toolMsg = {
          role: "tool",
          tool_call_id: id,
          name: call.function.name,
          content: string_content(result.output),
        };
      }
      messages.push(toolMsg);
    }
  }

  // Tool budget exhausted: force the report.
  if (ctx.aborted) {
    throw new Error("aborted");
  }
  messages.push({
    role: "user",
    content: string_content(verifierForceReport),
  });

  const finalMsg = (await Effect.runPromise(
    this.client.chat_stream_retry(ctx, {
      model: this.cfg.model,
      messages: messages,
    }, {})
  )) as message;

  return parseVerifierReport(content_string(finalMsg));
}

// parseVerifierReport extracts the JSON object from the verifier's final message.
export function parseVerifierReport(text: string): verifierReport {
  const start = text.indexOf("{");
  const end = text.lastIndexOf("}");
  if (start < 0 || end <= start) {
    throw new Error(`verifier returned no JSON report: "${truncateForError(text)}"`);
  }
  try {
    return JSON.parse(text.slice(start, end + 1)) as verifierReport;
  } catch (err: any) {
    throw new Error(`verifier report unparsable: ${err.message || String(err)}`);
  }
}

export function truncateForError(s: string): string {
  const max = 200;
  if (s.length > max) {
    return s.slice(0, max) + "...";
  }
  return s;
}

/*
PORT STATUS
source path: backend/agent/verifier.go
source lines: 220
draft lines: 251
confidence: high
status: phase_b_compile
*/
