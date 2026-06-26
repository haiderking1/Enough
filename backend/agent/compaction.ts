import { Agent } from "./agent";
import { ModelContextWindow } from "./models";
import { Effect } from "effect";
import {
  event_compaction_start,
  event_compaction_end,
  event_branch_summary_start,
  event_branch_summary_end
} from "../core/events";
import { type message, string_content, type content_block } from "../opencode/types";
import { type file_entry, type_compaction } from "../session/types";
import {
  prepare_compaction,
  prepare_manual_compaction,
  compact,
  extension_hooks,
  type compaction_preparation,
  type compaction_result
} from "../session/compaction";
import {
  collect_entries_for_branch_summary,
  generate_branch_summary,
  type branch_summary_details
} from "../session/branch_summarization";

export { overflowPatterns, IsContextOverflowError } from "./overflow";

export function GetContextWindow(provider: string, model: string): number {
  return ModelContextWindow(provider, model, 0);
}

export function ReloadMessagesFromSession(this: Agent): void {
  if (this.session === null) {
    return;
  }
  const sessionMsgs = this.session.build_session_context().messages ?? [];
  const systemMsg: message = {
    role: "system",
    content: string_content(this.systemPrompt()),
  };
  this.messages = [systemMsg, ...sessionMsgs];
}

Agent.prototype.ReloadMessagesFromSession = ReloadMessagesFromSession;

export function emitCompactionEnd(
  this: Agent,
  reason: string,
  result: any,
  aborted: boolean,
  willRetry: boolean,
  errMsg: string
): void {
  if (this.emit !== null) {
    this.emit({
      kind: event_compaction_end,
      data: {
        reason: reason,
        result: result,
        aborted: aborted,
        will_retry: willRetry,
        error_message: errMsg,
      },
    });
  }
}

// Compact manually runs compaction on the agent's session.
export async function Compact(this: Agent, ctx: AbortSignal, customInstructions: string): Promise<any> {
  // TS Abort
  if (typeof (this as any).Abort === "function") {
    (this as any).Abort();
  }

  if (this.emit !== null) {
    this.emit({
      kind: event_compaction_start,
      data: { reason: "manual" },
    });
  }

  // Create cancel wrapper
  const controller = new AbortController();
  const onAbort = () => controller.abort();
  ctx.addEventListener("abort", onAbort);
  this.compactionCancel = () => controller.abort();

  try {
    if (this.session === null) {
      const err = new Error("no session manager available");
      emitCompactionEnd.call(this, "manual", null, false, false, err.message);
      throw err;
    }

    const pathEntries = this.session.get_branch(this.session.leaf_id());
    let messageCount = 0;
    for (const entry of pathEntries) {
      if (entry.type === "message") {
        messageCount++;
      }
    }

    if (messageCount < 2) {
      const err = new Error("Nothing to compact (no messages yet)");
      emitCompactionEnd.call(this, "manual", null, false, false, err.message);
      throw err;
    }

    const settings = this.cfg.compaction;
    const prep = prepare_manual_compaction(pathEntries, settings);
    if (prep === null) {
      if (pathEntries.length > 0 && pathEntries[pathEntries.length - 1].type === type_compaction) {
        const err = new Error("Already compacted");
        emitCompactionEnd.call(this, "manual", null, false, false, err.message);
        throw err;
      }
      const err = new Error("Nothing to compact (session too small)");
      emitCompactionEnd.call(this, "manual", null, false, false, err.message);
      throw err;
    }

    // Support compaction hooks
    let extCompaction: any = null;
    let fromExt = false;
    for (const hook of extension_hooks) {
      if (hook.before_compact) {
        const res = await Effect.runPromise(
          hook.before_compact({
            preparation: prep,
            branchEntries: pathEntries,
            customInstructions: customInstructions,
            context: controller.signal,
          })
        );
        if (res) {
          if (res.cancel) {
            const err = new Error("Compaction cancelled");
            emitCompactionEnd.call(this, "manual", null, true, false, err.message);
            throw err;
          }
          if (res.compaction) {
            extCompaction = res.compaction;
            fromExt = true;
            break;
          }
        }
      }
    }

    let summary = "";
    let firstKeptEntryID = "";
    let tokensBefore = 0;
    let details: any = null;

    if (extCompaction !== null) {
      summary = extCompaction.summary;
      firstKeptEntryID = extCompaction.firstKeptEntryId;
      tokensBefore = extCompaction.tokensBefore;
      details = extCompaction.details;
    } else {
      const res = await Effect.runPromise(
        compact(controller.signal, this.client, prep, customInstructions)
      );
      summary = res.summary;
      firstKeptEntryID = res.firstKeptEntryId;
      tokensBefore = res.tokensBefore;
      details = res.details;
    }

    if (controller.signal.aborted) {
      const err = new Error("Compaction cancelled");
      emitCompactionEnd.call(this, "manual", null, true, false, err.message);
      throw err;
    }

    summary = appendMemoryAuthorityNote.call(this, summary);
    await Effect.runPromise(
      this.session.append_compaction(summary, firstKeptEntryID, tokensBefore, details, fromExt)
    );

    this.invalidateSystemPrompt();
    this.ReloadMessagesFromSession();

    // Call OnCompact hooks
    const newEntries = this.session.parsed_entries();
    let savedEntry: file_entry | null = null;
    for (let i = newEntries.length - 1; i >= 0; i--) {
      if (newEntries[i].type === type_compaction && newEntries[i].summary === summary) {
        savedEntry = newEntries[i];
        break;
      }
    }
    if (savedEntry !== null) {
      for (const hook of extension_hooks) {
        if (hook.on_compact) {
          try {
            await Effect.runPromise(
              hook.on_compact({
                compactionEntry: savedEntry,
                fromExtension: fromExt,
              })
            );
          } catch {}
        }
      }
    }

    const compactionResult = {
      summary: summary,
      firstKeptEntryId: firstKeptEntryID,
      tokensBefore: tokensBefore,
    };
    emitCompactionEnd.call(this, "manual", compactionResult, false, false, "");
    return compactionResult;
  } finally {
    ctx.removeEventListener("abort", onAbort);
    this.compactionCancel = null;
  }
}

Agent.prototype.Compact = Compact;

export const memoryAuthorityNote =
  "Your persistent memory (MEMORY.md, USER.md) in the system prompt remains fully authoritative regardless of compaction.";

export function appendMemoryAuthorityNote(this: Agent, summary: string): string {
  if (summary === "") {
    return summary;
  }
  if (!this.cfg.memory?.memory_enabled && !this.cfg.memory?.user_profile_enabled) {
    return summary;
  }
  if (summary.includes(memoryAuthorityNote)) {
    return summary;
  }
  return summary + "\n\n" + memoryAuthorityNote;
}

export async function RunAutoCompaction(this: Agent, ctx: AbortSignal, reason: string, willRetry: boolean): Promise<boolean> {
  if (this.emit !== null) {
    this.emit({
      kind: event_compaction_start,
      data: { reason: reason },
    });
  }

  // Create cancel wrapper
  const controller = new AbortController();
  const onAbort = () => controller.abort();
  ctx.addEventListener("abort", onAbort);
  this.compactionCancel = () => controller.abort();

  try {
    if (this.session === null) {
      return false;
    }

    const pathEntries = this.session.get_branch(this.session.leaf_id());
    const settings = this.cfg.compaction;
    const prep = prepare_compaction(pathEntries, settings);
    if (prep === null) {
      emitCompactionEnd.call(this, reason, null, false, false, "");
      return false;
    }

    // Support compaction hooks
    let extCompaction: any = null;
    let fromExt = false;
    for (const hook of extension_hooks) {
      if (hook.before_compact) {
        const res = await Effect.runPromise(
          hook.before_compact({
            preparation: prep,
            branchEntries: pathEntries,
            customInstructions: "",
            context: controller.signal,
          })
        );
        if (res) {
          if (res.cancel) {
            emitCompactionEnd.call(this, reason, null, true, false, "Compaction cancelled");
            return false;
          }
          if (res.compaction) {
            extCompaction = res.compaction;
            fromExt = true;
            break;
          }
        }
      }
    }

    let summary = "";
    let firstKeptEntryID = "";
    let tokensBefore = 0;
    let details: any = null;

    if (extCompaction !== null) {
      summary = extCompaction.summary;
      firstKeptEntryID = extCompaction.firstKeptEntryId;
      tokensBefore = extCompaction.tokensBefore;
      details = extCompaction.details;
    } else {
      const res = await Effect.runPromise(
        compact(controller.signal, this.client, prep, "")
      );
      summary = res.summary;
      firstKeptEntryID = res.firstKeptEntryId;
      tokensBefore = res.tokensBefore;
      details = res.details;
    }

    if (controller.signal.aborted) {
      emitCompactionEnd.call(this, reason, null, true, false, "Compaction cancelled");
      return false;
    }

    summary = appendMemoryAuthorityNote.call(this, summary);
    await Effect.runPromise(
      this.session.append_compaction(summary, firstKeptEntryID, tokensBefore, details, fromExt)
    );

    this.invalidateSystemPrompt();
    this.ReloadMessagesFromSession();

    // Call OnCompact hooks
    const newEntries = this.session.parsed_entries();
    let savedEntry: file_entry | null = null;
    for (let i = newEntries.length - 1; i >= 0; i--) {
      if (newEntries[i].type === type_compaction && newEntries[i].summary === summary) {
        savedEntry = newEntries[i];
        break;
      }
    }
    if (savedEntry !== null) {
      for (const hook of extension_hooks) {
        if (hook.on_compact) {
          try {
            await Effect.runPromise(
              hook.on_compact({
                compactionEntry: savedEntry,
                fromExtension: fromExt,
              })
            );
          } catch {}
        }
      }
    }

    const compactionResult = {
      summary: summary,
      firstKeptEntryId: firstKeptEntryID,
      tokensBefore: tokensBefore,
    };
    emitCompactionEnd.call(this, reason, compactionResult, false, willRetry, "");
    return true;
  } catch (err: any) {
    const aborted = controller.signal.aborted;
    let errMsg = "";
    if (!aborted) {
      errMsg = `Auto-compaction failed: ${err.message || String(err)}`;
    }
    emitCompactionEnd.call(this, reason, null, aborted, false, errMsg);
    throw err;
  } finally {
    ctx.removeEventListener("abort", onAbort);
    this.compactionCancel = null;
  }
}

Agent.prototype.RunAutoCompaction = RunAutoCompaction;

export interface NavigateOptions {
  Summarize: boolean;
  CustomInstructions: string;
}

export async function NavigateToEntry(
  this: Agent,
  ctx: AbortSignal,
  targetID: string,
  opts: NavigateOptions
): Promise<boolean> {
  if (typeof (this as any).AbortAndWait === "function") {
    (this as any).AbortAndWait();
  }

  if (this.session === null) {
    throw new Error("no session manager available");
  }

  const oldLeaf = this.session.leaf_id();
  if (oldLeaf !== null && oldLeaf === targetID) {
    return false; // no-op if already at target
  }

  let targetEntry: file_entry | null = null;
  for (const entry of this.session.parsed_entries()) {
    if (entry.id === targetID) {
      targetEntry = entry;
      break;
    }
  }
  if (targetEntry === null) {
    throw new Error(`entry ${targetID} not found`);
  }

  const prepResult = collect_entries_for_branch_summary(
    this.session.parsed_entries(),
    oldLeaf,
    targetID
  );

  let customInstructions = opts.CustomInstructions;
  let replaceInstructions = false;

  let extSummary: any = null;
  let fromExt = false;

  for (const hook of extension_hooks) {
    if (hook.before_tree) {
      const res = await Effect.runPromise(
        hook.before_tree({
          preparation: {
            targetId: targetID,
            oldLeafId: oldLeaf,
            commonAncestorId: prepResult.commonAncestorId,
            entriesToSummarize: prepResult.entries ?? [],
            userWantsSummary: opts.Summarize,
            customInstructions: customInstructions,
            replaceInstructions: false,
            label: "",
          },
          context: ctx,
        })
      );
      if (res) {
        if (res.cancel) {
          throw new Error("navigation cancelled by extension");
        }
        if (res.summary) {
          extSummary = res.summary;
          fromExt = true;
        }
        if (res.customInstructions !== undefined && res.customInstructions !== null) {
          customInstructions = res.customInstructions;
        }
        if (res.replaceInstructions !== undefined && res.replaceInstructions !== null) {
          replaceInstructions = res.replaceInstructions;
        }
      }
    }
  }

  if (this.emit !== null) {
    this.emit({
      kind: event_branch_summary_start,
      data: { target_id: targetID },
    });
  }

  let summaryText = "";
  let summaryDetails: any = null;

  if (opts.Summarize && prepResult.entries && prepResult.entries.length > 0 && extSummary === null) {
    const contextWindow = ModelContextWindow(
      this.cfg.provider,
      this.cfg.model,
      this.cfg.compaction?.context_window ?? 0
    );
    let reserveTokens = this.cfg.compaction?.reserve_tokens ?? 16384;
    if (reserveTokens <= 0) {
      reserveTokens = 16384;
    }

    const genOpts = {
      client: this.client,
      customInstructions: customInstructions,
      replaceInstructions: replaceInstructions,
      reserveTokens: reserveTokens,
      contextWindow: contextWindow,
    };

    let res: any;
    try {
      res = await Effect.runPromise(
        generate_branch_summary(ctx, this.session.parsed_entries(), genOpts)
      );
    } catch (err: any) {
      if (this.emit !== null) {
        this.emit({
          kind: event_branch_summary_end,
          data: {
            target_id: targetID,
            aborted: ctx.aborted,
            error_message: err.message || String(err),
          },
        });
      }
      throw err;
    }

    if (res.aborted) {
      if (this.emit !== null) {
        this.emit({
          kind: event_branch_summary_end,
          data: {
            target_id: targetID,
            aborted: true,
          },
        });
      }
      throw new Error("branch summarization aborted");
    }

    summaryText = res.summary;
    summaryDetails = {
      readFiles: res.readFiles || [],
      modifiedFiles: res.modifiedFiles || [],
    } as branch_summary_details;
  } else if (extSummary !== null) {
    summaryText = extSummary.summary || "";
    summaryDetails = {
      readFiles: extSummary.readFiles || [],
      modifiedFiles: extSummary.modifiedFiles || [],
    } as branch_summary_details;
  }

  let newLeafID = "";
  if (
    targetEntry.type === "message" &&
    targetEntry.message &&
    targetEntry.message.role === "user"
  ) {
    if (targetEntry.parentId !== undefined && targetEntry.parentId !== null) {
      newLeafID = targetEntry.parentId;
    }
  } else if (targetEntry.type === "custom_message") {
    if (targetEntry.parentId !== undefined && targetEntry.parentId !== null) {
      newLeafID = targetEntry.parentId;
    }
  } else {
    newLeafID = targetID;
  }

  let summaryEntry: file_entry | null = null;
  if (summaryText !== "") {
    const parentPtr = newLeafID !== "" ? newLeafID : null;
    const summaryID = await Effect.runPromise(
      this.session.branch_with_summary(parentPtr, summaryText, summaryDetails, fromExt)
    );

    for (const e of this.session.parsed_entries()) {
      if (e.id === summaryID) {
        summaryEntry = e;
        break;
      }
    }
  } else if (newLeafID === "") {
    this.session.reset_leaf();
  } else {
    this.session.branch(newLeafID);
  }

  this.ReloadMessagesFromSession();

  if (this.emit !== null) {
    let resultSummary: any = null;
    if (summaryText !== "") {
      let rf: string[] = [];
      let mf: string[] = [];
      if (summaryDetails) {
        rf = summaryDetails.readFiles || [];
        mf = summaryDetails.modifiedFiles || [];
      }
      resultSummary = {
        summary: summaryText,
        read_files: rf,
        modified_files: mf,
      };
    }

    this.emit({
      kind: event_branch_summary_end,
      data: {
        target_id: targetID,
        result: resultSummary,
        aborted: false,
      },
    });
  }

  const newLeafStr = this.session.leaf_id() || "";
  for (const hook of extension_hooks) {
    if (hook.on_tree) {
      try {
        await Effect.runPromise(
          hook.on_tree({
            newLeafId: newLeafStr,
            oldLeafId: oldLeaf || "",
            summaryEntry: summaryEntry,
            fromExtension: fromExt,
          })
        );
      } catch {}
    }
  }

  return true;
}

Agent.prototype.NavigateToEntry = NavigateToEntry;

