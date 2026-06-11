package skills

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed enough_skill/SKILL.md
var enoughSkillBundle []byte

// ExtractEnoughSkillIfMissing ensures the default "enough" skill is written to ~/.enough/skills/enough/SKILL.md
func ExtractEnoughSkillIfMissing() error {
	dir := filepath.Join(SkillsDir(), "enough")
	target := filepath.Join(dir, "SKILL.md")

	if _, err := os.Stat(target); err == nil {
		return nil // Already exists
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(target, enoughSkillBundle, 0644)
}
