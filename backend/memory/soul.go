package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/enough/enough/backend/enoughhome"
)

// SOUL.md — the agent's primary identity. When present, its content becomes
// the first stable block of the system prompt, replacing the default Enough
// persona. disclosurePolicy (anti base-model disclosure) always follows SOUL;
// it does not override the user's chosen name or persona.

const soulMaxChars = 24000

// DefaultSoul is seeded on first run when ~/.enough/SOUL.md is missing. It is
// user-editable; edits take effect on the next session.
const DefaultSoul = `# SOUL.md — agent identity

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
`

func SoulPath() string {
	return filepath.Join(enoughhome.HomeDir(), "SOUL.md")
}

// EnsureSoul seeds the default SOUL.md if missing. Best-effort.
func EnsureSoul() {
	path := SoulPath()
	if _, err := os.Stat(path); err == nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	_ = os.WriteFile(path, []byte(DefaultSoul), 0o600)
}

// LoadSoul loads SOUL.md, seeding the default on first run. The content is
// threat-scanned before injection: a poisoned SOUL.md yields a blocked
// placeholder instead of the file content (the file on disk is untouched so
// the user can inspect and fix it). Returns "" when the file is missing or
// empty, in which case the caller falls back to the built-in identity.
func LoadSoul() string {
	EnsureSoul()

	data, err := os.ReadFile(SoulPath())
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return ""
	}

	if ids := threatPatternIDs(content, ScopeContext); len(ids) > 0 {
		return fmt.Sprintf(
			"[BLOCKED: SOUL.md contained threat pattern(s): %s. Its content was removed from the "+
				"system prompt. Inspect and fix ~/.enough/SOUL.md, then start a new session.]",
			strings.Join(ids, ", "))
	}

	return truncateMiddle(content, "SOUL.md", soulMaxChars)
}

// truncateMiddle keeps the head and tail of oversized content with a marker
// in the middle, matching Hermes' context-file truncation.
func truncateMiddle(content, filename string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}
	headChars := maxChars * 7 / 10
	tailChars := maxChars * 2 / 10
	marker := fmt.Sprintf(
		"\n\n[...truncated %s: kept %d+%d of %d chars. Use file tools to read the full file.]\n\n",
		filename, headChars, tailChars, len(content))
	return content[:headChars] + marker + content[len(content)-tailChars:]
}
