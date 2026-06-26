import { type runtime } from "../config/config";
import { type message } from "../opencode/types";
import { string_content } from "../opencode/types";
import { goal_lock_enabled, continuity_enabled } from "../config/config";
import { seed_continuity_reads } from "./evidence/continuity";
import { registry } from "./obligations/registry";
import { detect_verify_command } from "./obligations/derive";
import { extract_task_verify_commands } from "./obligations/match";
import { Agent } from "./agent";
import { sessionFingerprints } from "./session_fingerprints";

// LoopPrompt starts the first run of an outer loop while keeping loop
// completion instructions hidden from the human-facing transcript.
export function LoopPrompt(
  this: Agent,
  ctx: AbortSignal,
  cfg: runtime,
  lockedPrompt: string,
  emit: ((event: any) => void) | null
): Promise<void> {
  const goalLock = cfg.evidence && goal_lock_enabled(cfg.evidence);
  const notice = loopRuntimeNotice(lockedPrompt, 1, !!goalLock);
  return this.prompt(ctx, cfg, lockedPrompt, null, notice, emit);
}

Agent.prototype.LoopPrompt = LoopPrompt;

// LoopContinue re-runs the agent on the same locked task without a visible
// user turn. The runtime notice is persisted so the model keeps the original
// task and completion contract in context.
export async function LoopContinue(
  this: Agent,
  ctx: AbortSignal,
  cfg: runtime,
  lockedPrompt: string,
  iteration: number,
  emit: ((event: any) => void) | null
): Promise<void> {
  if (this.busy) {
    throw new Error("agent is already processing");
  }
  this.busy = true;

  const controller = new AbortController();
  const userAbortController = new AbortController();

  const onAbort = () => {
    controller.abort();
  };
  ctx.addEventListener("abort", onAbort);
  if (ctx.aborted) {
    controller.abort();
  }

  this.cancel = () => controller.abort();
  this.userAbortCtx = userAbortController.signal;
  this.userAbortCancel = () => userAbortController.abort();
  this.applyConfigLocked(cfg);

  if (emit !== null) {
    this.emit = emit;
  }

  this.overflowRecoveryAttempted = false;
  const turnID = `turn_${Date.now()}_${Math.floor(Math.random() * 1000000)}`;
  this.resetEvidenceLedger(turnID);

  this.lastUserPrompt = lockedPrompt;
  this.lockedGoal = lockedPrompt;
  this.verifyFailures = 0;
  this.parallelForksAttempted = false;
  this.step = {
    lastVerifyFailed: false,
    failurePaths: null,
    lastVerifyOutput: "",
    lastBashCommand: "",
    lastBashFailed: false,
  };
  this.completionRounds = 0;

  if (cfg.evidence && cfg.evidence.enabled) {
    const verifyCmd = detect_verify_command(this.workDir);
    const taskVerify = extract_task_verify_commands(lockedPrompt);
    this.obligations = new registry(
      turnID,
      verifyCmd,
      taskVerify,
      cfg.evidence.strict_verify_reset,
      cfg.evidence.verifier_enabled
    );
  } else {
    this.obligations = null;
  }

  if (cfg.evidence && cfg.evidence.enabled && continuity_enabled(cfg.evidence) && this.session !== null) {
    seed_continuity_reads(this.evidenceLedger(), sessionFingerprints(this.session));
  }

  const goalLock = cfg.evidence && goal_lock_enabled(cfg.evidence);
  const noticeMsg: message = {
    role: "user",
    content: string_content(loopRuntimeNotice(lockedPrompt, iteration, !!goalLock)),
  };
  this.messages.push(noticeMsg);
  this.persist(noticeMsg);

  try {
    await this.runLoop(controller.signal);
  } finally {
    ctx.removeEventListener("abort", onAbort);
    controller.abort();
    this.busy = false;
    this.cancel = null;
    this.userAbortCtx = null;
    this.userAbortCancel = null;
  }
}

Agent.prototype.LoopContinue = LoopContinue;

function loopRuntimeNotice(lockedPrompt: string, iteration: number, goalLock: boolean): string {
  const loopPromisePattern = /<promise>([^<]+)<\/promise>/;
  let promise = "DONE";
  const match = lockedPrompt.match(loopPromisePattern);
  if (match && match.length === 2) {
    const custom = match[1].trim();
    if (custom !== "") {
      promise = custom;
    }
  }

  const runtimeNoticePrefix = "ℹ️ ";
  let out = runtimeNoticePrefix + `OUTER LOOP — iteration ${iteration}.\n`;
  out += `Continue until the task is fully complete. Only then include <promise>${promise}</promise> in the final response.\n`;
  if (goalLock) {
    out += "GOAL LOCK — work only on the original task; verify before declaring completion.\n";
  }
  out += "Do not mention this runtime notice to the user.\n\n";
  out += lockedPrompt;
  return out;
}

