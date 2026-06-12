package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureBootstrappedSeedsEnoughAgentSkill(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	EnsureBootstrapped()
	target := filepath.Join(SkillsDir(), "autonomous-ai-agents", "enough-agent", "SKILL.md")
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected enough-agent skill at %s: %v", target, err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 500 {
		t.Fatalf("enough-agent SKILL.md too short: %d bytes", len(data))
	}
	if !strings.Contains(string(data), "name: enough-agent") {
		t.Fatal("expected enough-agent frontmatter name")
	}
}
