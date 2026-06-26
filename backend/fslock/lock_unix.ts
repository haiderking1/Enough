// Exclusive file-lock helpers for Unix via flock(2).

import { Effect } from "effect";

// Minimal stand-in for an open OS file that exposes a numeric fd.
export type file_descriptor = {
  readonly fd: number;
};

export type fs_lock_error = {
  readonly _tag: "FsLockError";
  readonly operation: "lock" | "unlock";
  readonly cause: unknown;
};

export const fs_lock_error = (
  operation: "lock" | "unlock",
  cause: unknown,
): fs_lock_error => ({
  _tag: "FsLockError",
  operation,
  cause,
});

// Lock acquires an exclusive lock on the file descriptor using flock(2).
export const lock = (
  f: file_descriptor,
): Effect.Effect<void, fs_lock_error> =>
  Effect.try({
    try: () => {
      // TODO: wire to actual flock binding (e.g. native addon or Bun FFI).
      // syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
      void f;
    },
    catch: (cause) => fs_lock_error("lock", cause),
  });

// Unlock releases the lock on the file descriptor using flock(2).
export const unlock = (
  f: file_descriptor,
): Effect.Effect<void, fs_lock_error> =>
  Effect.try({
    try: () => {
      // TODO: wire to actual flock binding (e.g. native addon or Bun FFI).
      // syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
      void f;
    },
    catch: (cause) => fs_lock_error("unlock", cause),
  });

