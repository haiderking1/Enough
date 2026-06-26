import path from "node:path";
import fs from "node:fs";
import { Effect } from "effect";
import { searxng_error, type searxng_error as searxng_error_type } from "./error";

export type run_state = {
  port: number;
  pid: number;
};

const state_path = (data_dir: string): string => path.join(data_dir, "run.json");

export const write_state = (
  data_dir: string,
  port: number,
  pid: number,
): Effect.Effect<void, searxng_error_type> =>
  Effect.gen(function* () {
    const data = new TextEncoder().encode(JSON.stringify({ port, pid } as run_state));
    const p = state_path(data_dir);
    yield* Effect.try({
      try: () => fs.writeFileSync(p, data, { mode: 0o600 }),
      catch: (cause) => searxng_error("write run state", cause),
    });
  });

export const read_state = (data_dir: string): [number, number, boolean] => {
  try {
    const p = state_path(data_dir);
    const data = fs.readFileSync(p);
    const st = JSON.parse(data.toString("utf8")) as run_state;
    return [st.port, st.pid, st.port > 0 && st.pid > 0];
  } catch {
    return [0, 0, false];
  }
};

