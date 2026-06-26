import { Effect } from "effect";
import fs from "node:fs/promises";
import path from "node:path";
import { delete_session } from "./delete";
import { list_for_cwd, list_all } from "./list";
import { encode_cwd, home_agent_dir, same_project_cwd, sessions_subdir } from "./paths";
import type { info } from "./types";

const resolve_session_path = (raw: string, known: info[]): string | null => {
  const trimmed = raw.trim();
  if (trimmed === "") return null;
  if (trimmed.endsWith(".jsonl")) {
    return path.resolve(trimmed);
  }
  for (const info of known) {
    if (info.id === trimmed || info.path === trimmed) {
      return path.resolve(info.path);
    }
  }
  return null;
};

const jsonl_in_encoded_dir = (cwd: string): Effect.Effect<string[], Error> => {
  return Effect.gen(function* () {
    const agent_dir = yield* home_agent_dir();
    const dir = path.join(agent_dir, sessions_subdir, encode_cwd(path.resolve(cwd)));
    return yield* Effect.tryPromise({
      try: async () => {
        const entries = await fs.readdir(dir);
        return entries
          .filter((name) => name.endsWith(".jsonl"))
          .map((name) => path.resolve(path.join(dir, name)));
      },
      catch: (cause) => {
        if (cause && typeof cause === "object" && "code" in cause && cause.code === "ENOENT") {
          return null;
        }
        return cause instanceof Error ? cause : new Error(String(cause));
      },
    }).pipe(
      Effect.catchAll((err) => (err === null ? Effect.succeed([]) : Effect.fail(err))),
    );
  });
};

const remove_empty_session_dir = (cwd: string): Effect.Effect<void, Error> => {
  return Effect.gen(function* () {
    const agent_dir = yield* home_agent_dir();
    const dir = path.join(agent_dir, sessions_subdir, encode_cwd(path.resolve(cwd)));
    yield* Effect.tryPromise({
      try: async () => {
        const entries = await fs.readdir(dir);
        if (entries.length === 0) {
          await fs.rmdir(dir);
        }
      },
      catch: (cause) => {
        if (cause && typeof cause === "object" && "code" in cause && cause.code === "ENOENT") {
          return null;
        }
        return cause instanceof Error ? cause : new Error(String(cause));
      },
    }).pipe(
      Effect.catchAll((err) => (err === null ? Effect.void : Effect.fail(err))),
    );
  });
};

const collect_project_sessions = (cwd: string): Effect.Effect<info[], Error> => {
  return Effect.gen(function* () {
    const by_path = new Map<string, info>();
    for (const info of yield* list_all()) {
      if (same_project_cwd(info.cwd, cwd)) {
        by_path.set(path.resolve(info.path), info);
      }
    }
    for (const info of yield* list_for_cwd(cwd)) {
      by_path.set(path.resolve(info.path), info);
    }
    return Array.from(by_path.values());
  });
};

const collect_delete_paths = (
  cwd: string,
  explicit_paths: string[],
  known: info[],
): Effect.Effect<string[], Error> => {
  return Effect.gen(function* () {
    const paths = new Set<string>();
    for (const info of yield* collect_project_sessions(cwd)) {
      paths.add(path.resolve(info.path));
    }
    for (const file_path of yield* jsonl_in_encoded_dir(cwd)) {
      paths.add(file_path);
    }
    for (const raw of explicit_paths) {
      const resolved = resolve_session_path(raw, known);
      if (resolved !== null) {
        paths.add(resolved);
      }
    }
    return Array.from(paths);
  });
};

// DeleteForCWD removes all session JSONL files for a project directory.
// skipPath, if non-empty, is left on disk (e.g. the active session).
export const delete_for_cwd = (
  cwd: string,
  skip_path: string,
  explicit_paths: string[] = [],
): Effect.Effect<number, Error> => {
  return Effect.gen(function* () {
    const known = yield* list_all();
    const targets = yield* collect_delete_paths(cwd, explicit_paths, known);
    const clean_skip = skip_path ? path.resolve(skip_path) : "";
    let deleted = 0;
    for (const file_path of targets) {
      if (clean_skip !== "" && file_path === clean_skip) {
        continue;
      }
      const res = yield* Effect.either(delete_session(file_path));
      if (res._tag === "Left") {
        return yield* Effect.fail(res.left);
      }
      deleted++;
    }
    yield* remove_empty_session_dir(cwd);
    return deleted;
  });
};

