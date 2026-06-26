import { type SpawnOptions } from "node:child_process";

export function detachProcess(opts: SpawnOptions): void {
  opts.detached = true;
}

