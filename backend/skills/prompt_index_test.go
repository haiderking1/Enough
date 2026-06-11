package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enough/enough/backend/config"
)

func TestPromptIndexCategorical(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)

	// Clean/reset cache for testing
	ClearSkillsPromptCache()

	// 1. Returns empty string when no skills exist
	cfg := config.Runtime{
		Skills: config.SkillsSettings{Enabled: true},
	}
	result := BuildIndexPrompt(tempHome, cfg, []string{"skills_list", "skill_view"})
	if result != "" {
		t.Fatalf("expected empty prompt when no skills exist, got: %q", result)
	}

	// 2. Builds categorical index with skill_view guidance
	githubDir := filepath.Join(tempHome, "skills", "github")
	codeReviewDir := filepath.Join(githubDir, "code-review")
	if err := os.MkdirAll(codeReviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(githubDir, "DESCRIPTION.md"), []byte("---\ndescription: GitHub workflows\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codeReviewDir, "SKILL.md"), []byte("---\nname: code-review\ndescription: Review pull requests carefully\n---\n# Review\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result = BuildIndexPrompt(tempHome, cfg, []string{"skills_list", "skill_view"})
	if !strings.Contains(result, "## Skills (mandatory)") {
		t.Error("expected prompt to contain ## Skills (mandatory)")
	}
	if !strings.Contains(result, "skill_view(name)") {
		t.Error("expected prompt to contain skill_view(name)")
	}
	if !strings.Contains(result, "github: GitHub workflows") {
		t.Errorf("expected prompt to contain github: GitHub workflows, got: %s", result)
	}
	if !strings.Contains(result, "- code-review:") {
		t.Error("expected prompt to contain - code-review:")
	}

	// 3. Writes disk snapshot on cold scan
	snapshotPath := SnapshotPath()
	if _, err := os.Stat(snapshotPath); err != nil {
		t.Fatalf("expected snapshot to be written on disk: %v", err)
	}
	snapshotBytes, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(snapshotBytes), "code-review") {
		t.Error("expected snapshot to contain code-review")
	}

	// 4. LRU cache returns same result without rescanning
	ClearSkillsPromptCache()
	first := BuildIndexPrompt(tempHome, cfg, []string{"skills_list", "skill_view"})
	second := BuildIndexPrompt(tempHome, cfg, []string{"skills_list", "skill_view"})
	if first != second {
		t.Fatal("expected cached results to be identical")
	}
	if len(promptCache) == 0 {
		t.Error("expected prompt cache to be populated")
	}
}

func TestPromptIndexConditionalTools(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)
	ClearSkillsPromptCache()

	condDir := filepath.Join(tempHome, "skills", "conditional")
	if err := os.MkdirAll(condDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillContent := `---
name: conditional
description: Needs memory tool
metadata:
  hermes:
    requires_tools: [memory]
---
`
	if err := os.WriteFile(filepath.Join(condDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Runtime{
		Skills: config.SkillsSettings{Enabled: true},
	}

	// 1. Without memory tool -> empty
	without := BuildIndexPrompt(tempHome, cfg, []string{"skills_list", "skill_view"})
	if strings.Contains(without, "conditional") {
		t.Fatal("expected conditional skill to be hidden when memory tool is missing")
	}

	// 2. With memory tool -> present
	withTool := BuildIndexPrompt(tempHome, cfg, []string{"skills_list", "skill_view", "memory"})
	if !strings.Contains(withTool, "conditional") {
		t.Fatal("expected conditional skill to be present when memory tool is provided")
	}
}
