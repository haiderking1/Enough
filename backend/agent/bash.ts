// PORT: mirrors backend/agent/bash.go

import { Effect } from "effect";
import { type tool } from "../opencode/types";
import { Agent, type toolResult } from "./agent";
import { command_context } from "../shell/command";
import { resolve_safe_cwd } from "../shell/cwd";
import { bashCommandBlocked, SanitizeBashOutput } from "./bash_sanitize";
import { configureProcGroup as linuxConfigure, killProcessGroup as linuxKill } from "./bash_linux";
import { configureProcGroup as unixConfigure, killProcessGroup as unixKill } from "./bash_unix";
import { configureProcGroup as windowsConfigure, killProcessGroup as windowsKill } from "./bash_windows";
import { configureProcGroup as otherConfigure, killProcessGroup as otherKill } from "./bash_other";
import process from "node:process";

const maxBashOutput = 32000;
const exitStdioGrace = 100; // ms
const bashUpdateThrottle = 100; // ms
const truncMarker = "\n... truncated ...";

function configureProcGroup(cmd: any): void {
  if (process.platform === "linux") {
    linuxConfigure(cmd);
  } else if (process.platform === "win32") {
    windowsConfigure(cmd);
  } else if (
    process.platform === "darwin" ||
    process.platform === "freebsd" ||
    process.platform === "openbsd" ||
    process.platform === "netbsd"
  ) {
    unixConfigure(cmd);
  } else {
    otherConfigure(cmd);
  }
}

function killProcessGroup(cmd: any): Error | null {
  if (process.platform === "linux") {
    return linuxKill(cmd);
  } else if (process.platform === "win32") {
    return windowsKill(cmd);
  } else if (
    process.platform === "darwin" ||
    process.platform === "freebsd" ||
    process.platform === "openbsd" ||
    process.platform === "netbsd"
  ) {
    return unixKill(cmd);
  } else {
    return otherKill(cmd);
  }
}

export function bashTool(): tool {
  const schema = {
    type: "object",
    properties: {
      command: { type: "string" },
    },
    required: ["command"],
  };
  return {
    type: "function",
    function: {
      name: "bash",
      description: "Run a shell command in the project workspace. Do NOT run mpv, sixel, blessed, or full-screen TUI apps — they break the Enough terminal. Use curl, tests, and plain-text commands only.",
      parameters: new TextEncoder().encode(JSON.stringify(schema)),
    },
  };
}

Agent.prototype.toolBash = function (
  this: Agent,
  ctx: AbortSignal,
  id: string,
  argsJSON: string
): Effect.Effect<toolResult, Error> {
  let args: { command: string };
  try {
    args = JSON.parse(argsJSON);
  } catch (err) {
    return Effect.succeed({ output: err instanceof Error ? err.message : String(err), isErr: true });
  }

  const blocked = bashCommandBlocked(args.command);
  if (blocked !== "") {
    return Effect.succeed({ output: blocked, isErr: true });
  }

  const safeDir = resolve_safe_cwd(this.workDir);
  const commandEff = command_context(ctx, args.command, true, safeDir);

  return commandEff.pipe(
    Effect.flatMap((child) => {
      configureProcGroup(child);

      const delta = new BashDeltaEmitter((chunk) => {
        const [clean] = SanitizeBashOutput(chunk);
        if (clean !== "") {
          try {
            if (this.emit) {
              this.emit({ kind: "tool_delta", data: { id, chunk: clean } });
            }
          } catch {}
        }
      });

      const sw = new BashStreamWriter(maxBashOutput, (chunk) => delta.add(chunk));

      child.stdout?.on("data", (chunk: Buffer) => {
        sw.write(chunk.toString("utf8"));
      });
      child.stderr?.on("data", (chunk: Buffer) => {
        sw.write(chunk.toString("utf8"));
      });

      const started = Date.now();
      this.registerBashCmd(child);

      return Effect.async<toolResult, Error>((resume) => {
        let completed = false;
        let exitCode: number | null = null;
        let waitErr: Error | null = null;

        const onExit = (code: number | null, signal: string | null) => {
          if (completed) return;
          exitCode = code ?? -1;
          if (code !== 0) {
            waitErr = new Error(signal ? `signal: ${signal}` : `exit status ${code}`);
          }
          setTimeout(() => {
            finish();
          }, exitStdioGrace);
        };

        const onClose = () => {
          finish();
        };

        const finish = () => {
          if (completed) return;
          completed = true;

          child.removeListener("exit", onExit);
          child.removeListener("close", onClose);
          ctx.removeEventListener("abort", onAbort);
          this.unregisterBashCmd(child);

          sw.finalize();
          delta.flush();

          const duration = Date.now() - started;
          const [text] = SanitizeBashOutput(sw.toString());

          if (ctx.aborted) {
            let outText = text;
            if (outText !== "" && !outText.endsWith("\n")) {
              outText += "\n";
            }
            resume(Effect.succeed({ output: outText + "[interrupted]", isErr: true }));
            return;
          }

          this.recordCommandRun(args.command, exitCode ?? -1, text, duration);

          if (waitErr) {
            resume(Effect.succeed({ output: `${waitErr.message}\n${text}`, isErr: true }));
          } else {
            resume(Effect.succeed({ output: text }));
          }
        };

        const onAbort = () => {
          killProcessGroup(child);
          finish();
        };

        child.on("exit", onExit);
        child.on("close", onClose);
        ctx.addEventListener("abort", onAbort);
      });
    }),
    Effect.catchAll((err: any) =>
      Effect.succeed({ output: err instanceof Error ? err.message : String(err), isErr: true })
    )
  );
};

Agent.prototype.registerBashCmd = function (this: Agent, cmd: any) {
  this.activeBashCmd = cmd;
};

Agent.prototype.unregisterBashCmd = function (this: Agent, cmd: any) {
  if (this.activeBashCmd === cmd) {
    this.activeBashCmd = null;
  }
};

Agent.prototype.killActiveBash = function (this: Agent) {
  const cmd = this.activeBashCmd;
  if (cmd !== null) {
    killProcessGroup(cmd);
  }
};

export class BashStreamWriter {
  private buf: string[] = [];
  private max: number;
  private total = 0;
  private truncated = false;
  private onChunk?: (chunk: string) => void;
  private currentLen = 0;

  constructor(max: number, onChunk?: (chunk: string) => void) {
    this.max = max;
    this.onChunk = onChunk;
  }

  write(chunk: string): void {
    this.total += chunk.length;
    let emit = "";
    if (!this.truncated) {
      const room = this.max - this.currentLen;
      if (room <= 0) {
        this.truncated = true;
        this.buf.push(truncMarker);
        this.currentLen += truncMarker.length;
        emit = truncMarker;
      } else if (chunk.length <= room) {
        const [clean] = SanitizeBashOutput(chunk);
        this.buf.push(clean);
        this.currentLen += clean.length;
        emit = clean;
      } else {
        const [clean] = SanitizeBashOutput(chunk.slice(0, room));
        this.buf.push(clean);
        this.buf.push(truncMarker);
        this.currentLen += clean.length + truncMarker.length;
        this.truncated = true;
        emit = clean + truncMarker;
      }
    }
    if (emit !== "" && this.onChunk) {
      this.onChunk(emit);
    }
  }

  finalize(): void {
    let emit = "";
    if (this.total > this.max && !this.truncated) {
      this.truncated = true;
      this.buf.push(truncMarker);
      this.currentLen += truncMarker.length;
      emit = truncMarker;
    }
    if (emit !== "" && this.onChunk) {
      this.onChunk(emit);
    }
  }

  toString(): string {
    return this.buf.join("");
  }
}

export class BashDeltaEmitter {
  private emit: (chunk: string) => void;
  private pending: string[] = [];
  private lastAt = 0;
  private timer: NodeJS.Timeout | null = null;

  constructor(emit: (chunk: string) => void) {
    this.emit = emit;
  }

  add(chunk: string): void {
    if (chunk === "") {
      return;
    }
    this.pending.push(chunk);
    const now = Date.now();
    const delay = bashUpdateThrottle - (now - this.lastAt);
    if (delay <= 0) {
      this.flushLocked();
      return;
    }
    if (this.timer === null) {
      this.timer = setTimeout(() => {
        this.timer = null;
        this.flushLocked();
      }, delay);
    }
  }

  flush(): void {
    if (this.timer !== null) {
      clearTimeout(this.timer);
      this.timer = null;
    }
    this.flushLocked();
  }

  private flushLocked(): void {
    if (this.pending.length === 0) {
      return;
    }
    this.emit(this.pending.join(""));
    this.pending = [];
    this.lastAt = Date.now();
  }
}

/*
PORT STATUS
source path: backend/agent/bash.go
source lines: 304
draft lines: 279
confidence: high
status: phase_b_compile
*/
