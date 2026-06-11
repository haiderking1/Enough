package evidence

import (
	"encoding/json"
	"testing"
)

func TestAppendAndCount(t *testing.T) {
	l := NewLedger("turn-1")
	if l.Count() != 0 {
		t.Fatalf("new ledger count = %d, want 0", l.Count())
	}

	e, err := l.Append(KindReadFile, ReadFilePayload{Path: "/a/b.go", ContentHash: "abc", LineCount: 10})
	if err != nil {
		t.Fatal(err)
	}
	if e.ID != "ev_1" || e.TurnID != "turn-1" || e.Kind != KindReadFile {
		t.Fatalf("unexpected entry: %+v", e)
	}
	if l.Count() != 1 {
		t.Fatalf("count = %d, want 1", l.Count())
	}

	var p ReadFilePayload
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.Path != "/a/b.go" || p.LineCount != 10 {
		t.Fatalf("payload roundtrip failed: %+v", p)
	}
}

func TestHasRead(t *testing.T) {
	l := NewLedger("t")
	if l.HasRead("/x.go") {
		t.Fatal("HasRead true on empty ledger")
	}

	if _, err := l.Append(KindReadFile, ReadFilePayload{Path: "/x.go"}); err != nil {
		t.Fatal(err)
	}
	if !l.HasRead("/x.go") {
		t.Fatal("HasRead false after read entry")
	}
	if l.HasRead("/y.go") {
		t.Fatal("HasRead true for unread path")
	}

	// A write entry must not count as a read.
	if _, err := l.Append(KindWriteFile, MutationPayload{Path: "/z.go"}); err != nil {
		t.Fatal(err)
	}
	if l.HasRead("/z.go") {
		t.Fatal("write entry counted as read")
	}
}

func TestEntriesIsCopy(t *testing.T) {
	l := NewLedger("t")
	_, _ = l.Append(KindReadFile, ReadFilePayload{Path: "/x.go"})
	es := l.Entries()
	if len(es) != 1 {
		t.Fatalf("len = %d, want 1", len(es))
	}
	es[0].Kind = KindWebSearch
	if l.Entries()[0].Kind != KindReadFile {
		t.Fatal("Entries returned a shared slice")
	}
}

func TestHashBytes(t *testing.T) {
	a := HashBytes([]byte("hello"))
	b := HashBytes([]byte("hello"))
	c := HashBytes([]byte("world"))
	if a != b {
		t.Fatal("hash not deterministic")
	}
	if a == c {
		t.Fatal("distinct content hashed equal")
	}
	if len(a) != 64 {
		t.Fatalf("hash length = %d, want 64 hex chars", len(a))
	}
}
