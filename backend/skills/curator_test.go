package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/enough/enough/backend/config"
)

func writeTestSkill(t *testing.T, name string) {
	t.Helper()
	dir := filepath.Join(SkillsDir(), name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: test skill\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func seedAgentRecord(t *testing.T, name string, lastActivity time.Time, state string, pinned bool) {
	t.Helper()
	um := LoadUsage()
	agent := "agent"
	ts := lastActivity.UTC().Format(time.RFC3339Nano)
	um[name] = UsageRecord{
		CreatedBy:  &agent,
		UseCount:   1,
		LastUsedAt: &ts,
		CreatedAt:  ts,
		State:      state,
		Pinned:     pinned,
	}
	SaveUsage(um)
}

func TestCuratorDeterministicTransitions(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	now := time.Now()
	cfg := config.DefaultCurator() // stale 30d, archive 90d

	writeTestSkill(t, "fresh-skill")
	seedAgentRecord(t, "fresh-skill", now.Add(-24*time.Hour), "active", false)

	writeTestSkill(t, "stale-skill")
	seedAgentRecord(t, "stale-skill", now.Add(-40*24*time.Hour), "active", false)

	writeTestSkill(t, "ancient-skill")
	seedAgentRecord(t, "ancient-skill", now.Add(-100*24*time.Hour), "stale", false)

	writeTestSkill(t, "pinned-ancient")
	seedAgentRecord(t, "pinned-ancient", now.Add(-200*24*time.Hour), "active", true)

	writeTestSkill(t, "revived-skill")
	seedAgentRecord(t, "revived-skill", now.Add(-time.Hour), "stale", false)

	counts := ApplyAutomaticTransitions(cfg, now)
	if counts.MarkedStale != 1 || counts.Archived != 1 || counts.Reactivated != 1 {
		t.Fatalf("unexpected counts: %+v", counts)
	}

	um := LoadUsage()
	if um["stale-skill"].State != "stale" {
		t.Fatal("stale-skill should be stale")
	}
	if um["ancient-skill"].State != "archived" {
		t.Fatal("ancient-skill should be archived")
	}
	if _, err := os.Stat(filepath.Join(ArchiveDir(), "ancient-skill")); err != nil {
		t.Fatal("ancient-skill directory should be in .archive/")
	}
	if um["pinned-ancient"].State != "active" {
		t.Fatal("pinned skill must be immune")
	}
	if um["revived-skill"].State != "active" {
		t.Fatal("revived skill should be reactivated")
	}
	if _, err := os.Stat(filepath.Join(SkillsDir(), "fresh-skill", "SKILL.md")); err != nil {
		t.Fatal("fresh skill must be untouched")
	}
}

func TestCuratorNeverTouchesProtectedOrNonAgentSkills(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	now := time.Now()
	cfg := config.DefaultCurator()

	// Bundled `enough` skill, even with a (bogus) agent-created record and
	// ancient activity, must survive.
	writeTestSkill(t, "enough")
	seedAgentRecord(t, "enough", now.Add(-365*24*time.Hour), "active", false)

	// A user-created skill (no created_by) with ancient activity must not be
	// a candidate at all.
	writeTestSkill(t, "user-skill")
	um := LoadUsage()
	ts := now.Add(-365 * 24 * time.Hour).UTC().Format(time.RFC3339Nano)
	um["user-skill"] = UsageRecord{UseCount: 5, LastUsedAt: &ts, CreatedAt: ts, State: "active"}
	SaveUsage(um)

	counts := ApplyAutomaticTransitions(cfg, now)
	if counts.Archived != 0 || counts.MarkedStale != 0 {
		t.Fatalf("protected/non-agent skills were touched: %+v", counts)
	}
	if _, err := os.Stat(filepath.Join(SkillsDir(), "enough", "SKILL.md")); err != nil {
		t.Fatal("bundled enough skill must never be archived")
	}
	if _, err := os.Stat(filepath.Join(SkillsDir(), "user-skill", "SKILL.md")); err != nil {
		t.Fatal("user skill must never be archived by the curator")
	}
}

func TestShouldRunCuratorFirstRunDefersAndSeeds(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	_ = os.MkdirAll(SkillsDir(), 0o700)
	cfg := config.DefaultCurator()
	now := time.Now()

	if ShouldRunCurator(cfg, now) {
		t.Fatal("first observation must defer, not run")
	}
	st := LoadCuratorState()
	if st.LastRunAt == "" || !strings.Contains(st.LastRunSummary, "deferred first run") {
		t.Fatalf("state not seeded: %+v", st)
	}

	// Still within the interval: no run.
	if ShouldRunCurator(cfg, now.Add(time.Hour)) {
		t.Fatal("should not run within interval")
	}
	// Past the interval: run.
	if !ShouldRunCurator(cfg, now.Add(time.Duration(cfg.IntervalHours+1)*time.Hour)) {
		t.Fatal("should run after interval")
	}

	// Paused blocks everything.
	SetCuratorPaused(true)
	if ShouldRunCurator(cfg, now.Add(time.Duration(cfg.IntervalHours+1)*time.Hour)) {
		t.Fatal("paused curator must not run")
	}
	SetCuratorPaused(false)

	// Disabled blocks everything.
	cfg.Enabled = false
	if ShouldRunCurator(cfg, now.Add(time.Duration(cfg.IntervalHours+1)*time.Hour)) {
		t.Fatal("disabled curator must not run")
	}
}

func TestCuratorCandidateListExcludesProtected(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	now := time.Now()

	writeTestSkill(t, "enough")
	seedAgentRecord(t, "enough", now, "active", false)
	writeTestSkill(t, "real-candidate")
	seedAgentRecord(t, "real-candidate", now, "active", false)

	list := RenderCuratorCandidateList()
	if strings.Contains(list, "- enough ") {
		t.Fatalf("protected builtin in candidate list:\n%s", list)
	}
	if !strings.Contains(list, "real-candidate") {
		t.Fatalf("candidate missing:\n%s", list)
	}
}
