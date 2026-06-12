package memory

import (
	"fmt"
	"strings"

	"github.com/enough/enough/backend/skills"
)

// Threat scanning for content that gets injected into the system prompt
// (SOUL.md, AGENTS.md context files, MEMORY.md / USER.md entries).
//
// Patterns are shared with the skill guard (backend/skills/guard_patterns.go)
// — the single source of truth. Two scopes, mirroring Hermes:
//
//   - "strict" (memory entries): critical + high severity across ALL
//     categories, plus invisible unicode. Memory entries are user-curated
//     and enter the system prompt as a FROZEN snapshot, so a poisoned entry
//     persists for the entire session and across sessions until removed.
//   - "context" (SOUL.md, AGENTS.md): injection-category patterns plus
//     invisible unicode. Context files legitimately mention shell commands,
//     credentials directories, installs, etc. — only prompt-injection
//     content is blocked.
//
// Medium/low findings (e.g. unpinned installs) are never blocked:
// declarative facts like "project installs deps with pip install -r
// requirements.txt" are legitimate memory content.

type ScanScope int

const (
	// ScopeStrict blocks critical/high findings in any category.
	ScopeStrict ScanScope = iota
	// ScopeContext blocks injection-category findings only.
	ScopeContext
)

// threatPatternIDs returns the IDs of all blocking threat patterns matched in
// content for the given scope, deduplicated. Empty when content is clean.
func threatPatternIDs(content string, scope ScanScope) []string {
	var ids []string
	seen := make(map[string]bool)
	lines := strings.Split(content, "\n")

	for _, p := range skills.SkillGuardThreatPatterns {
		switch scope {
		case ScopeStrict:
			if p.Severity != "critical" && p.Severity != "high" {
				continue
			}
		case ScopeContext:
			if p.Category != "injection" {
				continue
			}
			if p.Severity != "critical" && p.Severity != "high" {
				continue
			}
		}
		if seen[p.PatternID] {
			continue
		}
		for _, line := range lines {
			if p.Regex.MatchString(line) {
				seen[p.PatternID] = true
				ids = append(ids, p.PatternID)
				break
			}
		}
	}

	for _, char := range skills.InvisibleChars {
		if strings.Contains(content, char) {
			if !seen["invisible_unicode"] {
				seen["invisible_unicode"] = true
				ids = append(ids, "invisible_unicode")
			}
			break
		}
	}

	return ids
}

// FirstThreatMessage scans memory content (strict scope) and returns an error
// message naming the matched pattern(s), or "" when clean.
func FirstThreatMessage(content string) string {
	ids := threatPatternIDs(content, ScopeStrict)
	if len(ids) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"Blocked: content matched threat pattern(s): %s. Memory content is injected into the "+
			"system prompt, so injection/exfiltration patterns are rejected. Rephrase the entry "+
			"as a plain declarative fact.",
		strings.Join(ids, ", "))
}

// ContextThreatIDs scans context-file content (SOUL.md, AGENTS.md) for prompt
// injection. Returns matched pattern IDs, empty when clean.
func ContextThreatIDs(content string) []string {
	return threatPatternIDs(content, ScopeContext)
}
