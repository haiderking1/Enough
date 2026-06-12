package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/skills"
)

func TestReviewWhitelistDeniesOtherTools(t *testing.T) {
	a := &Agent{cfg: testRuntime(), allowedTools: reviewToolWhitelist}
	res := a.guardTool("bash", `{"command":"ls"}`)
	if res == nil || !res.isErr || !strings.Contains(res.output, "not permitted") {
		t.Fatalf("bash should be denied in review fork: %+v", res)
	}
	if a.guardTool("memory", `{"action":"read","target":"memory"}`) != nil {
		t.Fatal("memory should be permitted in review fork")
	}
	if a.guardTool("skill_manage", `{}`) != nil {
		t.Fatal("skill_manage should be permitted in review fork")
	}
}

func TestToolMenuFiltersByWhitelist(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	cfg := testRuntime()
	cfg.Skills.Enabled = true
	a := &Agent{cfg: cfg, allowedTools: reviewToolWhitelist}
	for _, tool := range a.toolMenu() {
		if !reviewToolWhitelist[tool.Function.Name] {
			t.Fatalf("non-whitelisted tool in fork menu: %s", tool.Function.Name)
		}
	}
	// Memory tool present when enabled.
	found := false
	for _, tool := range a.toolMenu() {
		if tool.Function.Name == "memory" {
			found = true
		}
	}
	if !found {
		t.Fatal("memory tool missing from fork menu")
	}
}

func TestSkillProvenanceByWriteOrigin(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	cfg := testRuntime()
	cfg.Skills.Enabled = true

	skillContent := func(name string) string {
		return "---\nname: " + name + "\ndescription: test skill\n---\n\nSteps here.\n"
	}
	createArgs := func(name string) string {
		b, _ := json.Marshal(map[string]string{
			"action": "create", "name": name, "content": skillContent(name),
		})
		return string(b)
	}

	// Foreground create: NOT marked agent-created.
	fg := &Agent{cfg: cfg, writeOrigin: WriteOriginForeground}
	if res := fg.toolSkillManage(createArgs("user-owned-skill")); res.isErr {
		t.Fatalf("foreground create failed: %s", res.output)
	}
	rec := skills.LoadUsage()["user-owned-skill"]
	if rec.CreatedBy != nil && *rec.CreatedBy == "agent" {
		t.Fatal("foreground create must not be marked agent-created")
	}

	// Background review create: marked agent-created.
	bg := &Agent{cfg: cfg, writeOrigin: WriteOriginBackgroundReview}
	if res := bg.toolSkillManage(createArgs("agent-sediment-skill")); res.isErr {
		t.Fatalf("background create failed: %s", res.output)
	}
	rec = skills.LoadUsage()["agent-sediment-skill"]
	if rec.CreatedBy == nil || *rec.CreatedBy != "agent" {
		t.Fatal("background-review create must be marked agent-created")
	}
}

func TestSummarizeReviewActions(t *testing.T) {
	mkTool := func(payload string) opencode.Message {
		return opencode.Message{Role: "tool", Content: opencode.StringContent(payload)}
	}
	msgs := []opencode.Message{
		mkTool(`{"success":true,"message":"Entry added.","target":"user"}`),
		mkTool(`{"success":true,"message":"Skill 'foo' created."}`),
		mkTool(`{"success":false,"error":"nope"}`),
		mkTool(`not json`),
		mkTool(`{"success":true,"message":"Entry added.","target":"user"}`), // dedup
	}
	actions := summarizeReviewActions(msgs)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %v", actions)
	}
	if actions[0] != "User profile updated" || !strings.Contains(actions[1], "created") {
		t.Fatalf("unexpected actions: %v", actions)
	}
}

func TestBackgroundDeleteArchivesInsteadOfRemoving(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	cfg := testRuntime()
	cfg.Skills.Enabled = true
	bg := &Agent{cfg: cfg, writeOrigin: WriteOriginBackgroundReview}

	create := `{"action":"create","name":"doomed-skill","content":"---\nname: doomed-skill\ndescription: d\n---\n\nBody.\n"}`
	if res := bg.toolSkillManage(create); res.isErr {
		t.Fatalf("create failed: %s", res.output)
	}
	if res := bg.toolSkillManage(`{"action":"delete","name":"doomed-skill","absorbed_into":""}`); res.isErr {
		t.Fatalf("delete failed: %s", res.output)
	}
	// Archived, not destroyed.
	if _, err := os.Stat(filepath.Join(skills.ArchiveDir(), "doomed-skill")); err != nil {
		t.Fatal("background delete must archive, not remove")
	}
	rec := skills.LoadUsage()["doomed-skill"]
	if rec.State != "archived" {
		t.Fatalf("usage record should be archived, got %q", rec.State)
	}

	// Protected builtin refused outright (legacy alias names resolve too).
	enoughDir := filepath.Join(skills.SkillsDir(), "enough-agent")
	_ = os.MkdirAll(enoughDir, 0o700)
	_ = os.WriteFile(filepath.Join(enoughDir, "SKILL.md"), []byte("---\nname: enough-agent\ndescription: e\n---\n\nBody.\n"), 0o600)
	if res := bg.toolSkillManage(`{"action":"delete","name":"enough"}`); !res.isErr || !strings.Contains(res.output, "protected") {
		t.Fatalf("protected builtin delete must be refused: %s", res.output)
	}
	if _, err := os.Stat(filepath.Join(enoughDir, "SKILL.md")); err != nil {
		t.Fatal("enough-agent skill must survive")
	}
}

func TestMaybeSpawnBackgroundReviewTriggersAndResets(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	cfg := testRuntime()
	cfg.Skills.Enabled = true

	a := &Agent{cfg: cfg, writeOrigin: WriteOriginForeground, itersSinceSkill: 10}
	a.initMemoryStore()
	a.cachedSystemPrompt = "cached"
	a.messages = []opencode.Message{
		{Role: "system", Content: opencode.StringContent("cached")},
		{Role: "user", Content: opencode.StringContent("hi")},
		{Role: "assistant", Content: opencode.StringContent("done")},
	}

	// Trigger fires: counter resets and the fork runs (endpoint is
	// unreachable, so the pass fails fast with no side effects).
	a.maybeSpawnBackgroundReview(false)
	a.WaitForBackgroundReviews()
	if a.itersSinceSkill != 0 {
		t.Fatalf("skill nudge counter not reset: %d", a.itersSinceSkill)
	}

	// No triggers → no spawn, counter untouched below the interval.
	a.itersSinceSkill = 3
	a.maybeSpawnBackgroundReview(false)
	a.WaitForBackgroundReviews()
	if a.itersSinceSkill != 3 {
		t.Fatal("counter should be untouched below the interval")
	}
}

func TestForegroundMemoryWriteResetsNudgeCounter(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	a := &Agent{cfg: testRuntime(), writeOrigin: WriteOriginForeground, turnsSinceMemory: 7}
	a.initMemoryStore()
	if res := a.toolMemory(`{"action":"add","target":"memory","content":"a durable fact"}`); res.isErr {
		t.Fatalf("memory add failed: %s", res.output)
	}
	if a.turnsSinceMemory != 0 {
		t.Fatalf("foreground memory write should reset nudge counter, got %d", a.turnsSinceMemory)
	}

	// Background-review writes do NOT reset the parent counter semantics.
	b := &Agent{cfg: testRuntime(), writeOrigin: WriteOriginBackgroundReview, turnsSinceMemory: 7, memStore: a.memStore}
	if res := b.toolMemory(`{"action":"add","target":"memory","content":"another durable fact"}`); res.isErr {
		t.Fatalf("memory add failed: %s", res.output)
	}
	if b.turnsSinceMemory != 7 {
		t.Fatal("background write must not reset the nudge counter")
	}
}
