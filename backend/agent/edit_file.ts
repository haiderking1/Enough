// PORT: mirrors backend/agent/edit_file.go

import { Effect } from "effect";
import fs from "node:fs/promises";
import { type tool } from "../opencode/types";
import { Agent, type toolResult } from "./agent";

export function editFileTool(): tool {
  const schema = {
    type: "object",
    properties: {
      path: { type: "string", description: "File path" },
      old_string: { type: "string", description: "Exact text to find, including whitespace" },
      new_string: { type: "string", description: "Replacement text" },
      replace_all: { type: "boolean", description: "Replace every occurrence instead of requiring a unique match" }
    },
    required: ["path", "old_string", "new_string"]
  };
  return {
    type: "function",
    function: {
      name: "edit_file",
      description: "Replace exact text in an existing file. old_string must match uniquely unless replace_all is true.",
      parameters: new TextEncoder().encode(JSON.stringify(schema)),
    },
  };
}

Agent.prototype.toolEditFile = function (
  this: Agent,
  argsJSON: string
): Effect.Effect<toolResult, Error> {
  return Effect.gen(this, function* () {
    let args: {
      path: string;
      old_string: string;
      new_string: string;
      replace_all?: boolean;
    };
    try {
      args = JSON.parse(argsJSON);
    } catch (err) {
      return { output: err instanceof Error ? err.message : String(err), isErr: true };
    }

    const resolvedPath = yield* this.resolvePath(args.path);

    const data = yield* Effect.tryPromise({
      try: async () => await fs.readFile(resolvedPath, "utf8"),
      catch: (cause) => (cause instanceof Error ? cause : new Error(String(cause))),
    });

    const [newContent, count, editErr] = applyEdit(data, args.old_string, args.new_string, !!args.replace_all);
    if (editErr !== null) {
      return { output: editErr.message, isErr: true };
    }

    yield* Effect.tryPromise({
      try: async () => await fs.writeFile(resolvedPath, newContent, "utf8"),
      catch: (cause) => (cause instanceof Error ? cause : new Error(String(cause))),
    });

    return {
      output: `edited ${resolvedPath} (${count} replacement(s))`,
    };
  }).pipe(
    Effect.catchAll((err) =>
      Effect.succeed({ output: err.message, isErr: true })
    )
  );
};

export function applyEdit(
  content: string,
  oldStr: string,
  newStr: string,
  replaceAll: boolean
): [string, number, Error | null] {
  if (oldStr === "") {
    return ["", 0, new Error("old_string is required")];
  }

  let count = 0;
  let pos = content.indexOf(oldStr);
  while (pos !== -1) {
    count++;
    pos = content.indexOf(oldStr, pos + oldStr.length);
  }

  if (count === 0) {
    return ["", 0, new Error("old_string not found in file")];
  }

  if (!replaceAll) {
    if (count > 1) {
      return ["", 0, new Error(`old_string is not unique (${count} matches); add context or set replace_all`)];
    }
    return [content.replace(oldStr, newStr), 1, null];
  }

  const newContent = content.split(oldStr).join(newStr);
  return [newContent, count, null];
}

/*
PORT STATUS
source path: backend/agent/edit_file.go
source lines: 84
draft lines: 99
confidence: high
status: phase_b_compile
*/
