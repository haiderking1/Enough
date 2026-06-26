//go:build !unix placeholder

import fs from "node:fs";
import { Effect } from "effect";
import { secrets_error, type secrets_error as secrets_error_type } from "./store";

export const verify_owner = (path: string): Effect.Effect<void, secrets_error_type> =>
  Effect.gen(function* () {
    yield* Effect.try({
      try: () => fs.statSync(path),
      catch: (cause) => secrets_error("stat credentials file", cause),
    });
  });

