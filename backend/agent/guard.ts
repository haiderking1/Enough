import { Agent, type toolResult, WriteOriginBackgroundReview } from "./agent";
import { Effect } from "effect";
import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";
import {
  ledger,
  kind_read_file,
  kind_write_file,
  kind_edit_file,
  kind_command_run,
  kind_verifier_pass,
  kind_verifier_fail,
  read_source_author,
  hash_bytes,
  type read_file_payload,
  type mutation_payload,
} from "./evidence/ledger";
import {
  event_evidence_append,
  event_obligation_update,
} from "../core/events";
import { status_closed, type obligation } from "./obligations/registry";

// sessionFingerprints moved to ./session_fingerprints (pure, no Agent dependency).
export { sessionFingerprints } from "./session_fingerprints";

// evidenceLedger returns the agent's per-turn ledger, lazily creating one.
export function evidenceLedger(this: Agent): ledger {
  if (this.ledger === null) {
    this.ledger = new ledger("");
  }
  return this.ledger;
}

Agent.prototype.evidenceLedger = evidenceLedger;

export function resetEvidenceLedger(this: Agent, turnID: string): void {
  this.ledger = new ledger(turnID);
}

Agent.prototype.resetEvidenceLedger = resetEvidenceLedger;

export function evidenceEnabled(this: Agent): boolean {
  return !!this.cfg.evidence?.enabled;
}

Agent.prototype.evidenceEnabled = evidenceEnabled;

// guardTool enforces hard rules before a tool runs. A non-nil result is the
// rejection returned to the model; the tool is never executed.
export function guardTool(this: Agent, name: string, argsJSON: string): toolResult | null {
  if (this.allowedTools !== null && !this.allowedTools[name]) {
    return {
      output: `REJECTED: tool '${name}' is not permitted for this role`,
      isErr: true,
    };
  }
  if (this.readonlyRole && name === "bash") {
    let args: { command?: string } = {};
    try {
      args = JSON.parse(argsJSON);
    } catch {
      // ignore
    }
    const violation = readonlyBashViolation(args.command || "");
    if (violation !== "") {
      return { output: "REJECTED: read-only role cannot run " + violation, isErr: true };
    }
  }

  if (name !== "write_file" && name !== "edit_file") {
    return null;
  }
  if (!this.evidenceEnabled()) {
    return null;
  }

  const [filePath, ok] = toolPathArg(argsJSON);
  if (!ok) {
    return null; // malformed args; let the tool produce its own error
  }
  let abs: string;
  try {
    abs = Effect.runSync(this.resolvePath(filePath));
  } catch {
    return null; // path errors are the tool's to report
  }

  try {
    if (!fs.existsSync(abs)) {
      return null; // creating a new file requires no prior read
    }
  } catch {
    return null;
  }

  if (!this.evidenceLedger().has_read(abs)) {
    return {
      output: `REJECTED: edit/write blocked — read_file '${filePath}' not in evidence ledger this turn`,
      isErr: true,
    };
  }
  return null;
}

Agent.prototype.guardTool = guardTool;

export function readonlyBashViolation(command: string): string {
  const lower = command.toLowerCase();
  const blocked = [
    "git checkout", "git switch", "git reset", "git clean", "git commit",
    "git merge", "git rebase", "git pull", "git fetch", "gh pr checkout",
    "rm ", "rm\t", "mv ", "cp ", "mkdir ", "touch ", "truncate ",
    "sed -i", "perl -pi", "tee ", "chmod ", "chown ",
    ">>", " > ",
  ];
  for (const token of blocked) {
    if (lower.includes(token)) {
      return `mutating command "${token}"`;
    }
  }
  return "";
}

// recordEvidence appends a ledger entry for a successful tool execution.
export function recordEvidence(this: Agent, name: string, argsJSON: string, beforeHash: string): void {
  if (!this.evidenceEnabled()) {
    return;
  }

  const [filePath, ok] = toolPathArg(argsJSON);
  if (!ok) {
    return;
  }
  let abs: string;
  try {
    abs = Effect.runSync(this.resolvePath(filePath));
  } catch {
    return;
  }

  let appendErr: any = null;
  let entry: any = null;

  switch (name) {
    case "read_file":
      let data: Buffer;
      try {
        data = fs.readFileSync(abs);
      } catch {
        return;
      }
      const dataStr = data.toString("utf8");
      let lines = 0;
      for (const c of dataStr) {
        if (c === "\n") {
          lines++;
        }
      }
      if (data.length > 0 && !dataStr.endsWith("\n")) {
        lines++;
      }
      try {
        entry = Effect.runSync(
          this.evidenceLedger().append(kind_read_file, {
            path: abs,
            content_hash: hash_bytes(new Uint8Array(data)),
            line_count: lines,
          } as read_file_payload)
        );
      } catch (err) {
        appendErr = err;
      }
      break;

    case "write_file":
    case "edit_file":
      const kind = name === "edit_file" ? kind_edit_file : kind_write_file;
      let wdata: Buffer;
      try {
        wdata = fs.readFileSync(abs);
      } catch {
        return;
      }
      const afterHash = hash_bytes(new Uint8Array(wdata));
      try {
        entry = Effect.runSync(
          this.evidenceLedger().append(kind, {
            path: abs,
            before_hash: beforeHash,
            after_hash: afterHash,
          } as mutation_payload)
        );
      } catch (err) {
        appendErr = err;
      }

      if (appendErr === null && entry !== null) {
        // In-turn author credit
        try {
          Effect.runSync(this.evidenceLedger().note_author_credit(abs, afterHash));
        } catch {}

        // Cross-turn continuity
        const sm = this.session;
        if (sm !== null && sm.fingerprints) {
          try {
            sm.fingerprints().upsert({
              path: abs,
              after_hash: afterHash,
              kind: name,
              turn_id: this.evidenceLedger().turn_id(),
              timestamp: new Date(),
            });
          } catch {}
        }
      }
      break;

    default:
      return;
  }

  if (appendErr !== null || entry === null) {
    return;
  }
  this.emitEvidence(entry.kind, abs);

  if (name === "write_file" || name === "edit_file") {
    this.noteMutation();
  }
}

Agent.prototype.recordEvidence = recordEvidence;

// recordCommandRun appends command evidence and closes the verify obligation on a matching pass.
export function recordCommandRun(
  this: Agent,
  command: string,
  exitCode: number,
  output: string,
  durationMs: number
): void {
  if (!this.evidenceEnabled()) {
    return;
  }

  const outputHash = crypto.createHash("sha256").update(output).digest("hex");
  let entry: any = null;
  try {
    entry = Effect.runSync(
      this.evidenceLedger().append(kind_command_run, {
        command,
        cwd: this.workDir,
        exit_code: exitCode,
        output_hash: outputHash,
        duration_ms: durationMs,
      })
    );
  } catch {
    return;
  }

  this.emitEvidence(entry.kind, command);

  const reg = this.obligationRegistry();
  if (reg !== null) {
    const touches = commandTouchesMutation(command, this.evidenceLedger().mutated_paths());
    if (reg.note_command_run(command, exitCode, entry.id, touches)) {
      this.noteVerifySuccess();
      this.emitObligations();
    } else {
      // IsVerifyCommand
      const isVerify = (cmd: string, verifyCmd: string, extraVerifyCmds: string[]): boolean => {
        const cmdTrim = cmd.trim();
        if (verifyCmd !== "" && cmdTrim.includes(verifyCmd.trim())) {
          return true;
        }
        for (const extra of extraVerifyCmds) {
          if (cmdTrim.includes(extra.trim())) {
            return true;
          }
        }
        return false;
      };

      if (isVerify(command, reg.verify_command(), reg.extra_verify_commands())) {
        const pytestNoTests = exitCode === 5 && command.includes("pytest");
        if (exitCode !== 0 && !pytestNoTests) {
          this.noteVerifyFailure();
        }
      }
    }
  }
}

Agent.prototype.recordCommandRun = recordCommandRun;

export function commandTouchesMutation(command: string, mutated: string[]): boolean {
  for (const p of mutated) {
    const base = path.basename(p);
    if (base !== "" && command.includes(base)) {
      return true;
    }
  }
  return false;
}

export function noteMutation(this: Agent): void {
  const reg = this.obligationRegistry();
  if (reg !== null) {
    if (reg.note_mutation()) {
      this.emitObligations();
    }
  }
}

Agent.prototype.noteMutation = noteMutation;

export function obligationRegistry(this: Agent): any {
  return this.obligations;
}

Agent.prototype.obligationRegistry = obligationRegistry;

export function emitEvidence(this: Agent, kind: string, pathStr: string): void {
  if (this.emit === null) {
    return;
  }
  this.emit({
    kind: event_evidence_append,
    data: {
      kind,
      path: pathStr,
      count: this.evidenceLedger().count(),
    },
  });
}

Agent.prototype.emitEvidence = emitEvidence;

export function emitObligations(this: Agent): void {
  const reg = this.obligationRegistry();
  if (this.emit === null || reg === null) {
    return;
  }
  const snap = reg.snapshot();
  const items: any[] = [];
  let closed = 0;
  let open = 0;
  for (const ob of snap) {
    const item = {
      kind: ob.kind,
      description: ob.description,
      closed: ob.status === status_closed,
    };
    items.push(item);
    if (item.closed) {
      closed++;
    } else {
      open++;
    }
  }
  this.emit({
    kind: event_obligation_update,
    data: {
      open,
      closed,
      items,
    },
  });
}

Agent.prototype.emitObligations = emitObligations;

export function fileHashIfExists(this: Agent, argsJSON: string): string {
  const [filePath, ok] = toolPathArg(argsJSON);
  if (!ok) {
    return "";
  }
  let abs: string;
  try {
    abs = Effect.runSync(this.resolvePath(filePath));
  } catch {
    return "";
  }
  try {
    const data = fs.readFileSync(abs);
    return crypto.createHash("sha256").update(data).digest("hex");
  } catch {
    return "";
  }
}

Agent.prototype.fileHashIfExists = fileHashIfExists;

export function toolPathArg(argsJSON: string): [string, boolean] {
  let args: { path?: string } = {};
  try {
    args = JSON.parse(argsJSON);
  } catch {
    return ["", false];
  }
  if (!args.path || args.path === "") {
    return ["", false];
  }
  return [args.path, true];
}

