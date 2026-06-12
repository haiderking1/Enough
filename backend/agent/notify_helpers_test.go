package agent

import (
	"testing"

	"github.com/enough/enough/backend/approval"
)

func TestNotifyStagedWriteSkills(t *testing.T) {
	var got []string
	var prompts [][2]string
	a := &Agent{
		notify: func(s string) { got = append(got, s) },
		approvalPrompt: func(subsystem, id string) {
			prompts = append(prompts, [2]string{subsystem, id})
		},
	}
	a.notifyStagedWrite(`{
  "success": true,
  "staged": true,
  "pending_id": "a1b2",
  "gist": "patch 'foo' SKILL.md (+3/-1 lines)",
  "message": "Staged for approval (skills.write_approval is on). Not yet saved — review with /skills pending."
}`)
	if len(got) != 1 {
		t.Fatalf("expected one notify, got %v", got)
	}
	if got[0] != "⏳ Staged for approval: patch 'foo' SKILL.md (+3/-1 lines) — use /skills approve a1b2" {
		t.Fatalf("unexpected notify: %q", got[0])
	}
	if len(prompts) != 1 || prompts[0][0] != approval.SubsystemSkills || prompts[0][1] != "a1b2" {
		t.Fatalf("unexpected approval prompt: %v", prompts)
	}
}

func TestNotifyStagedWriteMemory(t *testing.T) {
	var got []string
	var prompts [][2]string
	a := &Agent{
		notify: func(s string) { got = append(got, s) },
		approvalPrompt: func(subsystem, id string) {
			prompts = append(prompts, [2]string{subsystem, id})
		},
	}
	a.notifyStagedWrite(`{
  "success": true,
  "staged": true,
  "pending_id": "c3d4",
  "target": "memory",
  "message": "Staged for approval (memory.write_approval is on). Not yet saved — review with /memory pending."
}`)
	if len(got) != 1 || got[0] != "⏳ Staged for approval: Staged for approval (memory.write_approval is on). Not yet saved — review with /memory pending. — use /memory approve c3d4" {
		t.Fatalf("unexpected notify: %v", got)
	}
	if len(prompts) != 1 || prompts[0][0] != approval.SubsystemMemory || prompts[0][1] != "c3d4" {
		t.Fatalf("unexpected approval prompt: %v", prompts)
	}
}

func TestNotifyDirectMemoryWrite(t *testing.T) {
	var got []string
	a := &Agent{notify: func(s string) { got = append(got, s) }}
	a.notifyDirectMemoryWrite(
		`{"action":"add","target":"user","content":"Name is haider (lowercase h)"}`,
		`{"success": true, "target": "user", "message": "Entry added."}`,
	)
	if len(got) != 1 {
		t.Fatalf("expected one notify, got %v", got)
	}
	want := "💾 Saved to USER.md: Name is haider (lowercase h)"
	if got[0] != want {
		t.Fatalf("got %q want %q", got[0], want)
	}
}

func TestNotifyDirectMemoryWriteSkipsStaged(t *testing.T) {
	var got []string
	a := &Agent{notify: func(s string) { got = append(got, s) }}
	a.notifyDirectMemoryWrite(
		`{"action":"add","target":"user","content":"x"}`,
		`{"success": true, "staged": true, "pending_id": "ab12"}`,
	)
	if len(got) != 0 {
		t.Fatalf("staged write should not direct-notify: %v", got)
	}
}
