package skills

import (
	"path/filepath"
	"testing"
)

func TestPathsResolving(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)

	expectedSkills := filepath.Join(tempHome, "skills")
	if got := SkillsDir(); got != expectedSkills {
		t.Errorf("expected SkillsDir to be %q, got %q", expectedSkills, got)
	}

	expectedBundles := filepath.Join(tempHome, "skill-bundles")
	if got := SkillBundlesDir(); got != expectedBundles {
		t.Errorf("expected SkillBundlesDir to be %q, got %q", expectedBundles, got)
	}

	expectedSnapshot := filepath.Join(tempHome, ".skills_prompt_snapshot.json")
	if got := SnapshotPath(); got != expectedSnapshot {
		t.Errorf("expected SnapshotPath to be %q, got %q", expectedSnapshot, got)
	}

	expectedLegacy := filepath.Join(tempHome, "agent", "skills")
	if got := LegacyAgentSkillsDir(); got != expectedLegacy {
		t.Errorf("expected LegacyAgentSkillsDir to be %q, got %q", expectedLegacy, got)
	}

	expectedUsage := filepath.Join(tempHome, "skills", ".usage.json")
	if got := UsagePath(); got != expectedUsage {
		t.Errorf("expected UsagePath to be %q, got %q", expectedUsage, got)
	}

	expectedArchive := filepath.Join(tempHome, "skills", ".archive")
	if got := ArchiveDir(); got != expectedArchive {
		t.Errorf("expected ArchiveDir to be %q, got %q", expectedArchive, got)
	}
}
