package skills

import "strings"

// ResolveSkillLookupName normalizes legacy skill names from older Enough builds
// or ported Hermes docs to the canonical bundled reference skill.
func ResolveSkillLookupName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "enough", "hermes-agent", "enough-agent":
		return "enough-agent"
	default:
		return strings.TrimSpace(name)
	}
}
