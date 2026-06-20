// PORT STATUS: active
// Real Electron IPC layer (replaces electron_stub.ts).
// Renderer talks IPC only — NO WebSocket, NO Go binary spawn.

import { Effect, Stream } from "effect";
import { DesktopBridge, type DesktopResponse } from "../desktop_bridge";
import { decodeCommand } from "../schemas";
import { mapAgentEvent, mapDispatchResponse, type BackendMessage } from "./event_mapper";

/**
 * Minimal Electron main-process interface, structurally compatible with
 * electron's `ipcMain`. Hollow typechecks against this interface (not the real
 * electron types) so no hard electron import is required for `bun run typecheck`.
 */
export interface ElectronMainApi {
  handle(
    channel: string,
    handler: (event: unknown, payload: unknown) => Promise<unknown> | unknown,
  ): void;
  on?(channel: string, listener: (...args: unknown[]) => void): void;
}

/** Minimal BrowserWindow.webContents shape. */
export interface WebContentsLike {
  send(channel: string, payload: unknown): void;
  isDestroyed(): boolean;
}

export interface RegisterElectronIpcOptions {
  /** Return all live renderer webContents; called per event to broadcast. */
  getAllWebContents(): WebContentsLike[];
}

const DISPATCH_CHANNEL = "hollow:dispatch";
const EVENT_CHANNEL = "hollow:event";

const broadcast = (options: RegisterElectronIpcOptions, msg: BackendMessage): void => {
  for (const wc of options.getAllWebContents()) {
    if (!wc.isDestroyed()) {
      wc.send(EVENT_CHANNEL, msg);
    }
  }
};

const isPlainObject = (v: unknown): v is Record<string, unknown> =>
  v !== null && typeof v === "object" && !Array.isArray(v);

/**
 * Register Electron IPC handlers so the renderer's `window.hollowDesktop.dispatch`
 * (exposed by preload.cjs) reaches the agent in the main process.
 *
 * Subscribes to bridge events exactly once and fans every mapped event out to
 * all live renderer windows via webContents.send("hollow:event", msg).
 *
 * Static imports only — NO require(), NO WebSocket.
 */
export const registerElectronIpc = (
  api: ElectronMainApi,
  bridge: DesktopBridge,
  options: RegisterElectronIpcOptions,
): void => {
  // Dispatch: renderer → main → DesktopBridge → renderer.
  api.handle(DISPATCH_CHANNEL, async (_event, payload: unknown) => {
    const cmdOpt = decodeCommand(payload);
    if (cmdOpt._tag === "None") {
      return {
        ok: false,
        error: `Invalid desktop command structure: ${JSON.stringify(payload)}`,
      };
    }
    const command = cmdOpt.value;

    const result = await Effect.runPromise(
      Effect.either(bridge.dispatch(command)),
    );

    if (result._tag === "Left") {
      return { ok: false, error: result.left.message || String(result.left) };
    }

    const response: DesktopResponse = result.right;
    const mapped = mapDispatchResponse(response);
    return { ok: true, data: mapped };
  });

  // Subscribe to agent events exactly once and broadcast mapped messages.
  // Stream.fromPubSub is pulled lazily; runFork keeps it alive for the app lifetime.
  const eventStream: Stream.Stream<unknown, never> = bridge.subscribeEvents() as unknown as Stream.Stream<
    unknown,
    never
  >;

  Effect.runFork(
    Stream.runForEach(eventStream, (event) =>
      Effect.sync(() => {
        const msg = mapAgentEvent(event);
        if (msg !== null) {
          broadcast(options, msg as BackendMessage);
        }
      }),
    ),
  );
};
