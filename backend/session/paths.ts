import { Effect } from "effect";
import fs from "node:fs/promises";
import path from "node:path";
import { home_dir } from "../hollowhome/home";

export const agent_dir_name = ".hollow";
export const agent_subdir = "agent";
export const sessions_subdir = "sessions";
export const current_version = 1;

/** True when two paths refer to the same project folder (Windows-safe). */
export const same_project_cwd = (left: string, right: string): boolean => {
  const a = path.resolve(left.trim() === "" ? process.cwd() : left);
  const b = path.resolve(right.trim() === "" ? process.cwd() : right);
  if (process.platform === "win32") {
    return a.toLowerCase() === b.toLowerCase();
  }
  return a === b;
};

export const home_agent_dir = (): Effect.Effect<string, Error> => {
  return Effect.succeed(path.join(home_dir(), agent_subdir));
};

export const encode_cwd = (cwd: string): string => {
  let s = path.resolve(cwd.trim() === "" ? process.cwd() : cwd);
  // Strip leading path separator
  if (s.startsWith(path.sep)) {
    s = s.substring(path.sep.length);
  }
  // Handle Windows volume name (e.g. C:)
  const drive_match = s.match(/^[a-zA-Z]:/);
  if (drive_match) {
    const vol = drive_match[0];
    s = s.substring(vol.length);
    if (s.startsWith(path.sep)) {
      s = s.substring(path.sep.length);
    }
  }
  // Replace separators and colons with "-"
  s = s.split(path.sep).join("-").split(":").join("-");
  return "--" + s + "--";
};

export const session_dir = (cwd: string): Effect.Effect<string, Error> => {
  return home_agent_dir().pipe(
    Effect.flatMap((agent_dir) =>
      Effect.tryPromise({
        try: async () => {
          const dir = path.join(agent_dir, sessions_subdir, encode_cwd(cwd));
          await fs.mkdir(dir, { recursive: true, mode: 0o700 });
          return dir;
        },
        catch: (cause) => cause instanceof Error ? cause : new Error(String(cause)),
      }),
    ),
  );
};

