// PORT STATUS: active
// Thin keyring wrapper using the `secret-tool` CLI (libsecret/DBus) on Linux
// and `security` (Keychain) on macOS.
//
// Why CLI instead of keytar (native addon)?
// keytar requires node-gyp compilation and electron-rebuild for each ABI.
// secret-tool talks to the SAME libsecret/DBus backend as Go's
// zalando/go-keyring — zero native deps, works in both `bun` and `electron`
// without rebuild. Already installed on Arch Linux (libsecret-tools package).
//
// Linux attribute names must match go-keyring: service + username (NOT account).
// See zalando/go-keyring keyring_unix.go — Set/Get use attributes["username"].

import { spawnSync } from "node:child_process";
import { Effect } from "effect";
import type { secrets_error } from "./store";

let _available_cache: boolean | null = null;

export const keyring_available = (): boolean => {
  if (_available_cache !== null) return _available_cache;
  _available_cache = false;

  if (process.platform === "linux") {
    try {
      const r = spawnSync("secret-tool", ["--version"], { stdio: "ignore" });
      _available_cache = r.status === 0 || r.error === undefined;
    } catch {
      _available_cache = false;
    }
  } else if (process.platform === "darwin") {
    try {
      const r = spawnSync("security", ["-h"], { stdio: "ignore" });
      _available_cache = r.status !== undefined;
    } catch {
      _available_cache = false;
    }
  }
  return _available_cache;
};

const fail = (reason: string, cause: unknown): secrets_error => ({
  _tag: "SecretsError",
  reason,
  cause,
});

/**
 * Read a secret from the OS keyring.
 * Returns the key string on success, or fails with secrets_error (keyring
 * unavailable or key not found — caller falls through to file).
 *
 * Uses Effect.try (not Effect.sync) so throws go to the error channel,
 * not the defect channel — Effect.either catches them correctly.
 */
export const keyring_get = (
  service: string,
  account: string,
): Effect.Effect<string, secrets_error> =>
  Effect.try({
    try: () => {
      if (!keyring_available()) {
        throw fail("keyring unavailable", null);
      }

      if (process.platform === "linux") {
        const r = spawnSync(
          "secret-tool",
          ["lookup", "service", service, "username", account],
          { encoding: "utf8" },
        );
        if (r.status === 0 && r.stdout != null) {
          return r.stdout.trim();
        }
        // Key not found or secret-tool error — both fall through to file.
        throw fail("key not found", r.stderr);
      }

      if (process.platform === "darwin") {
        const r = spawnSync(
          "security",
          ["find-generic-password", "-s", service, "-a", account, "-w"],
          { encoding: "utf8" },
        );
        if (r.status === 0 && r.stdout != null) {
          return r.stdout.trim();
        }
        throw fail("key not found", r.stderr);
      }

      throw fail("keyring unsupported on this platform", null);
    },
    catch: (e) => e as secrets_error,
  });

/**
 * Store a secret in the OS keyring.
 * Succeeds or fails with secrets_error (caller falls through to file).
 */
export const keyring_set = (
  service: string,
  account: string,
  key: string,
): Effect.Effect<void, secrets_error> =>
  Effect.try({
    try: () => {
      if (!keyring_available()) {
        throw fail("keyring unavailable", null);
      }

      if (process.platform === "linux") {
        const r = spawnSync(
          "secret-tool",
          [
            "store",
            "--label",
            `Password for '${account}' on '${service}'`,
            "service",
            service,
            "username",
            account,
          ],
          { input: key, encoding: "utf8" },
        );
        if (r.status !== 0) {
          throw fail("keyring store failed", r.stderr);
        }
        return;
      }

      if (process.platform === "darwin") {
        // Delete existing entry first (security add-generic-password fails if key exists)
        spawnSync("security", ["delete-generic-password", "-s", service, "-a", account], {
          stdio: "ignore",
        });
        const r = spawnSync(
          "security",
          ["add-generic-password", "-s", service, "-a", account, "-w", key],
          { encoding: "utf8" },
        );
        if (r.status !== 0) {
          throw fail("keyring store failed", r.stderr);
        }
        return;
      }

      throw fail("keyring unsupported on this platform", null);
    },
    catch: (e) => e as secrets_error,
  });

/**
 * Delete a secret from the OS keyring. Never fails — errors are silently
 * ignored (matches Go's `_ = keyring.Delete(...)`).
 */
export const keyring_delete = (
  service: string,
  account: string,
): Effect.Effect<void, never> =>
  Effect.sync(() => {
    if (!keyring_available()) return;

    if (process.platform === "linux") {
      spawnSync("secret-tool", ["clear", "service", service, "username", account], {
        stdio: "ignore",
      });
    } else if (process.platform === "darwin") {
      spawnSync("security", ["delete-generic-password", "-s", service, "-a", account], {
        stdio: "ignore",
      });
    }
  });

// Allow tests to override the availability cache.
export const _reset_keyring_cache = (): void => {
  _available_cache = null;
};
