package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSeedContinuityHashMatch(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "a.py", "print('hi')\n")

	l := NewLedger("t2")
	n := SeedContinuityReads(l, []Fingerprint{{Path: p, AfterHash: HashBytes([]byte("print('hi')\n"))}})
	if n != 1 {
		t.Fatalf("seeded %d, want 1", n)
	}
	if !l.HasRead(p) {
		t.Fatal("continuity seed did not grant read credit")
	}

	var payload ReadFilePayload
	if err := json.Unmarshal(l.Entries()[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Source != ReadSourceContinuity {
		t.Fatalf("source = %q, want continuity", payload.Source)
	}
	if payload.LineCount != 1 {
		t.Fatalf("line count = %d, want 1", payload.LineCount)
	}
}

func TestSeedContinuityHashMismatch(t *testing.T) {
	dir := t.TempDir()
	p := writeTemp(t, dir, "a.py", "externally changed\n")

	l := NewLedger("t2")
	n := SeedContinuityReads(l, []Fingerprint{{Path: p, AfterHash: HashBytes([]byte("original\n"))}})
	if n != 0 || l.HasRead(p) {
		t.Fatal("external edit was credited")
	}
}

func TestSeedContinuityPartialMatch(t *testing.T) {
	dir := t.TempDir()
	ok := writeTemp(t, dir, "ok.py", "same\n")
	changed := writeTemp(t, dir, "changed.py", "new content\n")
	missing := filepath.Join(dir, "deleted.py")

	l := NewLedger("t2")
	n := SeedContinuityReads(l, []Fingerprint{
		{Path: ok, AfterHash: HashBytes([]byte("same\n"))},
		{Path: changed, AfterHash: HashBytes([]byte("old content\n"))},
		{Path: missing, AfterHash: HashBytes([]byte("x"))},
	})
	if n != 1 {
		t.Fatalf("seeded %d, want 1", n)
	}
	if !l.HasRead(ok) || l.HasRead(changed) || l.HasRead(missing) {
		t.Fatal("wrong paths credited")
	}
}

func TestSeedContinuityEmptyStore(t *testing.T) {
	l := NewLedger("t2")
	if n := SeedContinuityReads(l, nil); n != 0 || l.Count() != 0 {
		t.Fatal("empty fingerprint store seeded entries")
	}
}

func TestNoteAuthorCredit(t *testing.T) {
	l := NewLedger("t1")
	l.NoteAuthorCredit("/x.py", "hash")
	if !l.HasRead("/x.py") {
		t.Fatal("author credit did not grant read")
	}
	var payload ReadFilePayload
	if err := json.Unmarshal(l.Entries()[0].Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Source != ReadSourceAuthor {
		t.Fatalf("source = %q, want author", payload.Source)
	}
	// Author-credit entries are reads, not mutations.
	if len(l.MutatedPaths()) != 0 {
		t.Fatal("author credit counted as mutation")
	}
}
