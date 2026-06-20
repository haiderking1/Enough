// PORT: mirrors backend/memory/soul.go

import fs from "node:fs";
import path from "node:path";
import { home_dir } from "../enoughhome/home";
import { threatPatternIDs, ScanScope } from "./scan";

// SOUL.md — the agent's primary identity. When present, its content becomes
// the first stable block of the system prompt, replacing the default Enough
// persona. disclosurePolicy (anti base-model disclosure) always follows SOUL;
// it does not override the user's chosen name or persona.

export const soulMaxChars = 24000;

// DefaultSoul is seeded on first run when ~/.hollow/SOUL.md is missing. It is
// user-editable; edits take effect on the next session.
export const DefaultSoul = `# SOUL.md — agent identity

This file is your primary identity. Edit it to change your display name,
persona, and tone. Changes take effect on the next session (/new).

Replace "Enough" below with any name you prefer (e.g. smoke).

---

You are Enough, a coding agent optimized for fast, precise execution.
You are helpful, knowledgeable, and direct. You assist with writing and
editing code, analyzing repositories, answering questions, and executing
actions via your tools. You communicate clearly, admit uncertainty when
appropriate, and prioritize being genuinely useful over being verbose.
Be targeted and efficient in your exploration and investigations.
`;

export function SoulPath(): string {
  return path.join(home_dir(), "SOUL.md");
}

// EnsureSoul seeds the default SOUL.md if missing. Best-effort.
export function EnsureSoul(): void {
  const p = SoulPath();
  try {
    if (fs.existsSync(p)) {
      return;
    }
  } catch {
    // ignore
  }

  try {
    fs.mkdirSync(path.dirname(p), { recursive: true, mode: 0o700 });
  } catch {
    // ignore
  }

  try {
    fs.writeFileSync(p, DefaultSoul, { mode: 0o600 });
  } catch {
    // ignore
  }
}

// LoadSoul loads SOUL.md, seeding the default on first run. The content is
// threat-scanned before injection: a poisoned SOUL.md yields a blocked
// placeholder instead of the file content (the file on disk is untouched so
// the user can inspect and fix it). Returns "" when the file is missing or
// empty, in which case the caller falls back to the built-in identity.
export function LoadSoul(): string {
  EnsureSoul();

  let data: Buffer;
  try {
    data = fs.readFileSync(SoulPath());
  } catch {
    return "";
  }

  const content = data.toString("utf8").trim();
  if (content === "") {
    return "";
  }

  const ids = threatPatternIDs(content, ScanScope.ScopeContext);
  if (ids.length > 0) {
    return `[BLOCKED: SOUL.md contained threat pattern(s): ${ids.join(", ")}. Its content was removed from the system prompt. Inspect and fix ~/.hollow/SOUL.md, then start a new session.]`;
  }

  return truncateMiddle(content, "SOUL.md", soulMaxChars);
}

// truncateMiddle keeps the head and tail of oversized content with a marker
// in the middle, matching Hermes' context-file truncation.
export function truncateMiddle(content: string, filename: string, maxChars: number): string {
  if (content.length <= maxChars) {
    return content;
  }
  const headChars = Math.floor(maxChars * 7 / 10);
  const tailChars = Math.floor(maxChars * 2 / 10);
  const marker = `\n\n[...truncated ${filename}: kept ${headChars}+${tailChars} of ${content.length} chars. Use file tools to read the full file.]\n\n`;
  return content.slice(0, headChars) + marker + content.slice(content.length - tailChars);
}

/*
PORT STATUS
source path: backend/memory/soul.go
source lines: 92
draft lines: 104
confidence: high
status: phase_b_compile
*/
