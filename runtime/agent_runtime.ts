// PORT STATUS: active
// source path: runtime/agent_runtime.ts
// confidence: high
// status: phase_b_compile

import { Effect, Context, PubSub } from "effect";
// Single registration barrel — attaches all Agent prototype methods and
// re-exports { Agent, New } so this is the only agent import needed.
import { Agent, New } from "../backend/agent/register";
import { type runtime, load_runtime } from "../backend/config/config";
import { apply_provider_model } from "../backend/config/provider";
import { EnsureBootstrapped } from "../backend/skills/bootstrap";
import { open_session, list_for_cwd, list_all } from "../backend/session/list";
import { continue_recent, start_new } from "../backend/session/manager";
import { delete_session } from "../backend/session/delete";
import { type info } from "../backend/session/types";
import { new_client_for_runtime } from "../backend/opencode/runtime_client";

export class AgentRuntimeImpl {
  config!: runtime;
  workDir!: string;
  agent!: Agent;
  pubsub!: PubSub.PubSub<any>;

  boot(workDir?: string): Effect.Effect<void, Error> {
    const self = this;
    return Effect.gen(function* () {
      self.pubsub = yield* PubSub.unbounded<any>();
      const cfg = yield* load_runtime();
      self.config = cfg;

      yield* EnsureBootstrapped();

      let cleanWorkDir = workDir || "";
      if (cleanWorkDir === "") {
        cleanWorkDir = process.cwd();
      }
      self.workDir = cleanWorkDir;

      const sm = yield* continue_recent(self.workDir);
      self.agent = New(self.config, self.workDir, sm);

      self.agent.emit = (event: any) => {
        // Synchronous publish: prompt completion must mean all emitted events
        // are already delivered to subscribers before emit() returns.
        Effect.runSync(PubSub.publish(self.pubsub, event));
      };
    }).pipe(
      Effect.mapError((err) => {
        if (err instanceof Error) return err;
        const reason =
          (err && typeof err === "object" && "reason" in err && String(err.reason)) ||
          (err && typeof err === "object" && "message" in err && String(err.message)) ||
          String(err);
        return new Error(reason);
      })
    );
  }

  listSessions(cwd?: string): Effect.Effect<info[], Error> {
    return cwd ? list_for_cwd(cwd) : list_all();
  }

  openSession(id: string): Effect.Effect<string, Error> {
    const self = this;
    return Effect.gen(function* () {
      const infos = yield* list_all();
      let targetPath = "";
      for (const info of infos) {
        if (info.id === id || info.path === id) {
          targetPath = info.path;
          break;
        }
      }
      if (targetPath === "") {
        targetPath = id;
      }
      const sm = yield* open_session(targetPath);
      self.agent.LoadSession(sm);
      return sm.session_id();
    });
  }

  newSession(cwd?: string): Effect.Effect<string, Error> {
    const self = this;
    return Effect.gen(function* () {
      const targetCwd = cwd || self.workDir;
      const sm = yield* start_new(targetCwd);
      self.agent.LoadSession(sm);
      return sm.session_id();
    });
  }

  deleteSession(id: string): Effect.Effect<void, Error> {
    const self = this;
    return Effect.gen(function* () {
      const infos = yield* list_all();
      let targetPath = "";
      for (const info of infos) {
        if (info.id === id || info.path === id) {
          targetPath = info.path;
          break;
        }
      }
      if (targetPath === "") {
        targetPath = id;
      }
      if (self.agent.session && self.agent.session.session_file() === targetPath) {
        return yield* Effect.fail(new Error("cannot delete the active session"));
      }
      yield* delete_session(targetPath);
    });
  }

  prompt(text: string, attachments?: readonly any[]): Effect.Effect<void, Error> {
    const self = this;
    return Effect.async<void, Error>((resume) => {
      const controller = new AbortController();
      const mappedAttachments = attachments?.map((att) => ({
        MIMEType: att.mime,
        Data: Buffer.from(att.data, "base64"),
      })) || null;

      self.agent
        .Prompt(controller.signal, self.config, text, mappedAttachments, self.agent.emit)
        .then(() => resume(Effect.void))
        .catch((cause) =>
          resume(Effect.fail(cause instanceof Error ? cause : new Error(String(cause))))
        );
    });
  }

  interrupt(): Effect.Effect<void, Error> {
    const self = this;
    return Effect.sync(() => {
      self.agent.Abort();
    });
  }

  setModel(provider: string, model: string, thinkingLevel?: string): Effect.Effect<void, Error> {
    const self = this;
    return Effect.gen(function* () {
      yield* apply_provider_model(provider, model, thinkingLevel || "");
      const newCfg = yield* load_runtime();
      self.config = newCfg;
      self.agent.UpdateConfig(newCfg);
    }).pipe(
      Effect.mapError((err) => (err instanceof Error ? err : new Error(String(err))))
    );
  }
}

export const AgentRuntime = Context.GenericTag<AgentRuntimeImpl>("AgentRuntime");
