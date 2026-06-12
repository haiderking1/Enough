package skills

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed bundled/autonomous-ai-agents/enough-agent/SKILL.md
var agentReferenceSkill []byte

// ExtractEnoughSkillIfMissing ensures the canonical enough-agent reference skill
// exists under ~/.enough/skills/ (SyncSkills is the primary path; this is a
// lightweight fallback for first-run before sync completes).
func ExtractEnoughSkillIfMissing() error {
	dir := filepath.Join(SkillsDir(), "autonomous-ai-agents", "enough-agent")
	target := filepath.Join(dir, "SKILL.md")

	if _, err := os.Stat(target); err == nil {
		return nil
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	return os.WriteFile(target, agentReferenceSkill, 0o644)
}
