package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enough/enough/backend/agent/evidence"
	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/core"
)

func newGuardTestAgent(t *testing.T) (*Agent, string) {
	t.Helper()
	dir := t.TempDir()
	a := &Agent{
		workDir: dir,
		cfg:     config.Runtime{Evidence: config.DefaultEvidence()},
	}
	return a, dir
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEditWithoutReadRejected(t *testing.T) {
	a, dir := newGuardTestAgent(t)
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := mustJSON(t, map[string]string{"path": "f.txt", "old_string": "original", "new_string": "changed"})
	res := a.executeTool(context.Background(), "t1", "edit_file", args)

	if !res.isErr {
		t.Fatal("edit without prior read was not rejected")
	}
	if !strings.Contains(res.output, "REJECTED") || !strings.Contains(res.output, "evidence ledger") {
		t.Fatalf("unexpected rejection message: %q", res.output)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatalf("file changed on disk despite rejection: %q", data)
	}
}

func TestWriteExistingWithoutReadRejected(t *testing.T) {
	a, dir := newGuardTestAgent(t)
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := mustJSON(t, map[string]string{"path": "f.txt", "content": "overwritten"})
	res := a.executeTool(context.Background(), "t1", "write_file", args)

	if !res.isErr {
		t.Fatal("overwrite without prior read was not rejected")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatalf("file changed on disk despite rejection: %q", data)
	}
}

func TestReadThenEditSucceedsWithLedgerEntries(t *testing.T) {
	a, dir := newGuardTestAgent(t)
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	readRes := a.executeTool(context.Background(), "t1", "read_file", mustJSON(t, map[string]string{"path": "f.txt"}))
	if readRes.isErr {
		t.Fatalf("read failed: %s", readRes.output)
	}

	editArgs := mustJSON(t, map[string]string{"path": "f.txt", "old_string": "original", "new_string": "changed"})
	editRes := a.executeTool(context.Background(), "t2", "edit_file", editArgs)
	if editRes.isErr {
		t.Fatalf("edit after read rejected: %s", editRes.output)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "changed\n" {
		t.Fatalf("edit not applied: %q", data)
	}

	// read + edit + the edit's author-credit read entry
	entries := a.evidenceLedger().Entries()
	if len(entries) != 3 {
		t.Fatalf("ledger has %d entries, want 3", len(entries))
	}
	if entries[0].Kind != evidence.KindReadFile || entries[1].Kind != evidence.KindEditFile {
		t.Fatalf("unexpected entry kinds: %s, %s", entries[0].Kind, entries[1].Kind)
	}

	var mut evidence.MutationPayload
	if err := json.Unmarshal(entries[1].Payload, &mut); err != nil {
		t.Fatal(err)
	}
	if mut.BeforeHash == "" || mut.AfterHash == "" || mut.BeforeHash == mut.AfterHash {
		t.Fatalf("mutation hashes wrong: before=%q after=%q", mut.BeforeHash, mut.AfterHash)
	}
}

func TestWriteNewFileAllowedWithoutRead(t *testing.T) {
	a, dir := newGuardTestAgent(t)

	args := mustJSON(t, map[string]string{"path": "new.txt", "content": "hello"})
	res := a.executeTool(context.Background(), "t1", "write_file", args)
	if res.isErr {
		t.Fatalf("write of new file rejected: %s", res.output)
	}

	data, err := os.ReadFile(filepath.Join(dir, "new.txt"))
	if err != nil || string(data) != "hello" {
		t.Fatalf("new file not written: %v %q", err, data)
	}

	// write entry plus its author-credit read entry
	entries := a.evidenceLedger().Entries()
	if len(entries) != 2 || entries[0].Kind != evidence.KindWriteFile {
		t.Fatalf("expected write_file entry first, got %+v", entries)
	}
	var mut evidence.MutationPayload
	_ = json.Unmarshal(entries[0].Payload, &mut)
	if mut.BeforeHash != "" {
		t.Fatalf("new file should have empty before hash, got %q", mut.BeforeHash)
	}
}

func TestGuardDisabledRestoresV1Behavior(t *testing.T) {
	a, dir := newGuardTestAgent(t)
	a.cfg.Evidence.Enabled = false

	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	args := mustJSON(t, map[string]string{"path": "f.txt", "content": "overwritten"})
	res := a.executeTool(context.Background(), "t1", "write_file", args)
	if res.isErr {
		t.Fatalf("guard active while disabled: %s", res.output)
	}
	if a.evidenceLedger().Count() != 0 {
		t.Fatal("ledger recorded entries while disabled")
	}
}

func TestEvidenceEventEmitted(t *testing.T) {
	a, dir := newGuardTestAgent(t)
	var events []core.Event
	a.emit = func(e core.Event) { events = append(events, e) }

	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a.executeTool(context.Background(), "t1", "read_file", mustJSON(t, map[string]string{"path": "f.txt"}))

	var found *core.EvidenceEvent
	for _, e := range events {
		if e.Kind == core.EventEvidenceAppend {
			if ev, ok := e.Data.(core.EvidenceEvent); ok {
				found = &ev
			}
		}
	}
	if found == nil {
		t.Fatal("no EventEvidenceAppend emitted")
	}
	if found.Kind != string(evidence.KindReadFile) || found.Count != 1 || found.Path != path {
		t.Fatalf("unexpected evidence event: %+v", found)
	}
}

func TestLedgerResetPerTurn(t *testing.T) {
	a, dir := newGuardTestAgent(t)
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	a.executeTool(context.Background(), "t1", "read_file", mustJSON(t, map[string]string{"path": "f.txt"}))
	if !a.evidenceLedger().HasRead(path) {
		t.Fatal("read not recorded")
	}

	a.resetEvidenceLedger("turn_2")
	if a.evidenceLedger().HasRead(path) {
		t.Fatal("read evidence survived turn reset")
	}

	args := mustJSON(t, map[string]string{"path": "f.txt", "old_string": "x", "new_string": "y"})
	res := a.executeTool(context.Background(), "t2", "edit_file", args)
	if !res.isErr {
		t.Fatal("edit allowed using stale evidence from previous turn")
	}
}

// Swarm workers are independent Agent values; evidence in one must not
// satisfy the read-before-write rule in another.
func TestSwarmWorkerLedgerIsolation(t *testing.T) {
	parent, dir := newGuardTestAgent(t)
	path := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	parent.executeTool(context.Background(), "t1", "read_file", mustJSON(t, map[string]string{"path": "f.txt"}))

	worker := &Agent{
		cfg:        parent.cfg,
		workDir:    dir,
		swarmDepth: 1,
	}
	args := mustJSON(t, map[string]string{"path": "f.txt", "old_string": "x", "new_string": "y"})
	res := worker.executeTool(context.Background(), "t2", "edit_file", args)
	if !res.isErr {
		t.Fatal("worker edit allowed using parent's read evidence")
	}
}
