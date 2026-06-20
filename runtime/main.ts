// PORT STATUS: active
// source path: runtime/main.ts
// confidence: high
// status: phase_b_compile

import { Effect, Fiber, Stream, Deferred } from "effect";
import { AgentRuntimeImpl } from "./agent_runtime";
import { DesktopBridge } from "./desktop_bridge";

const main = () => {
  const args = process.argv.slice(2);
  let workdir: string | undefined;
  let prompt: string | undefined;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--workdir" && i + 1 < args.length) {
      workdir = args[i + 1];
      i++;
    } else if (args[i] === "--prompt" && i + 1 < args.length) {
      prompt = args[i + 1];
      i++;
    } else if (args[i] === "--help" || args[i] === "-h") {
      console.log("Usage: bun runtime/main.ts [--workdir <dir>] [--prompt <text>]");
      process.exit(0);
    }
  }

  const isProviderKeyError = (err: any): boolean => {
    const msg = String(err?.message || err || "").toLowerCase();
    return (
      msg.includes("not connected") ||
      msg.includes("api key") ||
      msg.includes("credentials") ||
      msg.includes("keyring") ||
      msg.includes("auth") ||
      msg.includes("token")
    );
  };

  const runtime = new AgentRuntimeImpl();
  const program = Effect.gen(function* () {
    yield* runtime.boot(workdir);

    if (prompt) {
      const bridge = new DesktopBridge(runtime);
      const eventStream = bridge.subscribeEvents();

      const fiber = yield* Effect.fork(
        Stream.runForEach(eventStream, (event) =>
          Effect.sync(() => {
            console.log(JSON.stringify(event));
          })
        )
      );

      yield* runtime.prompt(prompt);
      yield* Fiber.interrupt(fiber);
    } else {
      console.log("runtime ready");

      const shutdownDeferred = yield* Deferred.make<void, never>();

      process.on("SIGINT", () => {
        console.log("\nReceived SIGINT. Shutting down...");
        Effect.runFork(Deferred.succeed(shutdownDeferred, undefined));
      });
      process.on("SIGTERM", () => {
        console.log("\nReceived SIGTERM. Shutting down...");
        Effect.runFork(Deferred.succeed(shutdownDeferred, undefined));
      });

      yield* Deferred.await(shutdownDeferred);
    }
  });

  Effect.runPromise(program)
    .then(() => {
      process.exit(0);
    })
    .catch((err) => {
      if (isProviderKeyError(err)) {
        console.warn(`[Graceful Exit] Provider API keys/credentials missing: ${err.message || err}`);
        process.exit(0);
      } else {
        console.error("Runtime failed with error:", err?.message || err);
        process.exit(1);
      }
    });
};

main();
