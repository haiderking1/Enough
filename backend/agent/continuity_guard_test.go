package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enough/enough/backend/agent/evidence"
	"github.com/enough/enough/backend/core"
	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/session"
)

func newSessionAgent(t *testing.T) (*Agent, string) {
	t.Helper()
	dir := t.TempDir()
	sm, err := session.ContinueRecent(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := &Agent{workDir: dir, cfg: evidenceRuntime(false), session: sm}
	return a, dir
}

// seedNextTurn simulates what Prompt() does at turn start.
func seedNextTurn(a *Agent, turnID string) {
	a.resetEvidenceLedger(turnID)
	evidence.SeedContinuityReads(a.evidenceLedger(), sessionFingerprints(a.session))
}

// Matrix #4: agent wrote the file last turn, unchanged on disk → first
// write/edit this turn succeeds with zero rejects and zero reads.
func TestCrossTurnContinuityAllowsEditWithoutRead(t *testing.T) {
	a, dir := newSessionAgent(t)
	ctx := context.Background()

	a.resetEvidenceLedger("t1")
	res := a.executeTool(ctx, "w1", "write_file", `{"path":"count.py","content":"print(1)\n"}`)
	if res.isErr {
		t.Fatalf("turn 1 write failed: %s", res.output)
	}

	seedNextTurn(a, "t2")

	res = a.executeTool(ctx, "e1", "edit_file", `{"path":"count.py","old_string":"print(1)","new_string":"print(2)"}`)
	if res.isErr {
		t.Fatalf("cross-turn edit rejected despite continuity: %s", res.output)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "count.py"))
	if string(data) != "print(2)\n" {
		t.Fatalf("edit not applied: %q", data)
	}
}

// Matrix #5: user edited the file on disk between turns → no credit, reject,
// then a real read unlocks the edit.
func TestCrossTurnExternalEditForcesRead(t *testing.T) {
	a, dir := newSessionAgent(t)
	ctx := context.Background()

	a.resetEvidenceLedger("t1")
	a.executeTool(ctx, "w1", "write_file", `{"path":"count.py","content":"print(1)\n"}`)

	// External edit between turns.
	if err := os.WriteFile(filepath.Join(dir, "count.py"), []byte("user changed this\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	seedNextTurn(a, "t2")

	res := a.executeTool(ctx, "e1", "edit_file", `{"path":"count.py","old_string":"user changed this","new_string":"x"}`)
	if !res.isErr || !strings.Contains(res.output, "REJECTED") {
		t.Fatalf("stale-content edit was not rejected: %+v", res)
	}

	if res := a.executeTool(ctx, "r1", "read_file", `{"path":"count.py"}`); res.isErr {
		t.Fatalf("read failed: %s", res.output)
	}
	if res := a.executeTool(ctx, "e2", "edit_file", `{"path":"count.py","old_string":"user changed this","new_string":"x"}`); res.isErr {
		t.Fatalf("edit after read rejected: %s", res.output)
	}
}

// Matrix #6: write then edit the same file in the same turn — author credit.
func TestSameTurnWriteThenEdit(t *testing.T) {
	a, _ := newGuardTestAgent(t)
	ctx := context.Background()

	if res := a.executeTool(ctx, "w1", "write_file", `{"path":"new.py","content":"a = 1\n"}`); res.isErr {
		t.Fatalf("write failed: %s", res.output)
	}
	if res := a.executeTool(ctx, "e1", "edit_file", `{"path":"new.py","old_string":"a = 1","new_string":"a = 2"}`); res.isErr {
		t.Fatalf("same-turn edit after write rejected: %s", res.output)
	}
}

// Matrix #7: a path only read last turn (never written) earns no continuity.
func TestContinuityDoesNotCreditReadOnlyPaths(t *testing.T) {
	a, dir := newSessionAgent(t)
	ctx := context.Background()
	mustWrite(t, filepath.Join(dir, "lib.py"), "x = 1\n")

	a.resetEvidenceLedger("t1")
	if res := a.executeTool(ctx, "r1", "read_file", `{"path":"lib.py"}`); res.isErr {
		t.Fatal(res.output)
	}

	seedNextTurn(a, "t2")

	res := a.executeTool(ctx, "e1", "edit_file", `{"path":"lib.py","old_string":"x = 1","new_string":"x = 2"}`)
	if !res.isErr {
		t.Fatal("read-only path earned cross-turn credit")
	}
}

// Matrix #8: workers are isolated — a parent's fingerprints never seed a
// worker's ledger (workers have no session and seed nothing).
func TestContinuityNotLeakedToSwarmWorker(t *testing.T) {
	a, dir := newSessionAgent(t)
	ctx := context.Background()

	a.resetEvidenceLedger("t1")
	a.executeTool(ctx, "w1", "write_file", `{"path":"count.py","content":"print(1)\n"}`)

	worker := &Agent{cfg: a.cfg, workDir: dir, swarmDepth: 1}
	res := worker.executeTool(ctx, "e1", "edit_file", `{"path":"count.py","old_string":"print(1)","new_string":"print(2)"}`)
	if !res.isErr {
		t.Fatal("worker edited parent-authored file without reading")
	}
}

// End-to-end regression of the reported session: turn 1 writes count.py,
// turn 2 over-engineers it. The turn-2 write must succeed on the first
// attempt — no REJECTED round, no read_file call.
func TestOverEngineeredCountScriptRegression(t *testing.T) {
	dir := t.TempDir()
	sm, err := session.ContinueRecent(dir)
	if err != nil {
		t.Fatal(err)
	}

	srv := scriptedServer(t, func(req opencode.ChatRequest) (string, []toolCallJSON) {
		turn2 := false
		userCount := 0
		for _, m := range req.Messages {
			if m.Role == "user" && strings.Contains(opencode.ContentString(m), "over-engineered") {
				turn2 = true
			}
			if m.Role == "user" {
				userCount++
			}
		}
		last := req.Messages[len(req.Messages)-1]

		if last.Role == "user" && !turn2 {
			return "", []toolCallJSON{
				{Index: 0, ID: "c1", Type: "function", Function: toolFnJSON{Name: "write_file", Arguments: `{"path":"count.py","content":"print(1)\n"}`}},
				{Index: 1, ID: "c2", Type: "function", Function: toolFnJSON{Name: "bash", Arguments: `{"command":"cat count.py"}`}},
			}
		}
		if last.Role == "user" && turn2 {
			// The whole point: turn 2 writes FIRST, no read.
			return "", []toolCallJSON{
				{Index: 0, ID: "c3", Type: "function", Function: toolFnJSON{Name: "write_file", Arguments: `{"path":"count.py","content":"class Counter:\n    pass\n"}`}},
				{Index: 1, ID: "c4", Type: "function", Function: toolFnJSON{Name: "bash", Arguments: `{"command":"cat count.py"}`}},
			}
		}
		_ = userCount
		return "done", nil
	})
	defer srv.Close()

	a := &Agent{cfg: evidenceRuntime(false), client: opencode.NewClient(srv.URL, "k", "test-model"), workDir: dir, session: sm}

	promptWith(t, a, srv.URL, "write hello world counter", func(core.Event) {})
	promptWith(t, a, srv.URL, "make it over-engineered", func(core.Event) {})

	// First write of turn 2 succeeded: file holds the new content.
	data, _ := os.ReadFile(filepath.Join(dir, "count.py"))
	if !strings.Contains(string(data), "class Counter") {
		t.Fatalf("turn 2 write did not land: %q", data)
	}

	// Zero rejects and zero read_file calls anywhere in the transcript.
	for _, m := range a.messages {
		if m.Role == "tool" && strings.Contains(opencode.ContentString(m), "REJECTED") {
			t.Fatalf("guard rejected a call: %s", opencode.ContentString(m))
		}
		if m.Role == "tool" && m.Name == "read_file" {
			t.Fatal("a read_file round was wasted")
		}
	}
	if a.obligations.HasOpen() {
		t.Fatalf("turn 2 ended with open obligations: %+v", a.obligations.Open())
	}
}

// Matrix #10: continuity off (evidence on) restores reject-then-read.
func TestContinuityDisabledRejectsCrossTurnEdit(t *testing.T) {
	a, _ := newSessionAgent(t)
	off := false
	a.cfg.Evidence.ContinuityReads = &off
	ctx := context.Background()

	a.resetEvidenceLedger("t1")
	a.executeTool(ctx, "w1", "write_file", `{"path":"count.py","content":"print(1)\n"}`)

	// Turn start honoring the flag, as Prompt() does.
	a.resetEvidenceLedger("t2")
	if a.cfg.Evidence.ContinuityEnabled() {
		t.Fatal("flag not honored")
	}

	res := a.executeTool(ctx, "e1", "edit_file", `{"path":"count.py","old_string":"print(1)","new_string":"print(2)"}`)
	if !res.isErr {
		t.Fatal("continuity disabled but cross-turn edit allowed")
	}
}
