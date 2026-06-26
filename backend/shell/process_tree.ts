// Cross-platform process-tree kill + detached-pid tracking.

// killProcessTree. Detached children are tracked so they can be reaped on agent
// abort / backend shutdown; killProcessTree kills the whole group on Unix
// (negative pid) and uses taskkill /T on Windows.

import { spawnSync } from "node:child_process";
import process from "node:process";

const trackedDetachedChildPids = new Set<number>();

export function trackDetachedChildPid(pid: number): void {
  if (pid) {
    trackedDetachedChildPids.add(pid);
  }
}

export function untrackDetachedChildPid(pid: number): void {
  trackedDetachedChildPids.delete(pid);
}

// killProcessTree kills a process and all its descendants (cross-platform).
export function killProcessTree(pid: number): void {
  if (!pid) return;
  if (process.platform === "win32") {
    // taskkill /T kills the whole tree; /F forces it.
    try {
      spawnSync("taskkill", ["/F", "/T", "/PID", String(pid)], {
        stdio: "ignore",
        windowsHide: true,
        timeout: 10000,
      });
    } catch {
      // ignore — best-effort
    }
    return;
  }
  // Unix/Linux/Mac: SIGKILL the process group (negative pid), fall back to the
  // single process if the group kill misses.
  try {
    process.kill(-pid, "SIGKILL");
    return;
  } catch {
    // fall through
  }
  try {
    process.kill(pid, "SIGKILL");
  } catch {
    // process already dead
  }
}

// killTrackedDetachedChildren reaps every tracked detached child. Called on
// agent abort and backend shutdown so daemonized descendants don't outlive us.
export function killTrackedDetachedChildren(): void {
  for (const pid of trackedDetachedChildPids) {
    killProcessTree(pid);
  }
  trackedDetachedChildPids.clear();
}

// Reap tracked children when the event loop empties (natural backend exit).
// Registered once, idempotent. We intentionally avoid hijacking SIGTERM/SIGHUP
// so Electron's own shutdown lifecycle stays intact.
let shutdownHookInstalled = false;
export function installDetachedShutdownHook(): void {
  if (shutdownHookInstalled) return;
  shutdownHookInstalled = true;
  try {
    process.once("beforeExit", () => killTrackedDetachedChildren());
  } catch {
    // ignore
  }
}