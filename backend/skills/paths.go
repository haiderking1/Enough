package skills

import (
	"path/filepath"

	"github.com/enough/enough/backend/enoughhome"
)

func HomeDir() string {
	return enoughhome.HomeDir()
}

func SkillsDir() string {
	return filepath.Join(enoughhome.HomeDir(), "skills")
}

func LegacyAgentSkillsDir() string {
	return filepath.Join(enoughhome.HomeDir(), "agent", "skills")
}

func SnapshotPath() string {
	return filepath.Join(enoughhome.HomeDir(), ".skills_prompt_snapshot.json")
}

func SkillBundlesDir() string {
	return filepath.Join(enoughhome.HomeDir(), "skill-bundles")
}

func UsagePath() string {
	return filepath.Join(enoughhome.HomeDir(), "skills", ".usage.json")
}

func ArchiveDir() string {
	return filepath.Join(enoughhome.HomeDir(), "skills", ".archive")
}
