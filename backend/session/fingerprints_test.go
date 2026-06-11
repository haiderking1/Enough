package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFingerprintUpsertAndList(t *testing.T) {
	dir := t.TempDir()
	side := filepath.Join(dir, "s1.jsonl")
	s := NewFingerprintStore(func() string { return side })

	s.Upsert(FileFingerprint{Path: "/a.py", AfterHash: "h1", Kind: "write_file", TurnID: "t1", Timestamp: time.Now()})
	s.Upsert(FileFingerprint{Path: "/a.py", AfterHash: "h2", Kind: "edit_file", TurnID: "t2", Timestamp: time.Now()})
	s.Upsert(FileFingerprint{Path: "/b.py", AfterHash: "h3", Kind: "write_file", TurnID: "t2", Timestamp: time.Now()})

	fps := s.List()
	if len(fps) != 2 {
		t.Fatalf("len = %d, want 2 (last write wins per path)", len(fps))
	}
	byPath := map[string]string{}
	for _, fp := range fps {
		byPath[fp.Path] = fp.AfterHash
	}
	if byPath["/a.py"] != "h2" || byPath["/b.py"] != "h3" {
		t.Fatalf("wrong hashes: %+v", byPath)
	}

	// Sidecar persisted next to the session file, never the contents.
	sidecar := filepath.Join(dir, "s1.fingerprints.json")
	if _, err := os.Stat(sidecar); err != nil {
		t.Fatalf("sidecar not written: %v", err)
	}
}

func TestFingerprintSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	side := filepath.Join(dir, "s1.jsonl")

	s1 := NewFingerprintStore(func() string { return side })
	s1.Upsert(FileFingerprint{Path: "/a.py", AfterHash: "h1"})

	// Fresh store for the same session (resume): loads from sidecar.
	s2 := NewFingerprintStore(func() string { return side })
	fps := s2.List()
	if len(fps) != 1 || fps[0].AfterHash != "h1" {
		t.Fatalf("resume lost fingerprints: %+v", fps)
	}
}

func TestFingerprintSessionSwitchDropsEntries(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "s1.jsonl")
	s := NewFingerprintStore(func() string { return current })
	s.Upsert(FileFingerprint{Path: "/a.py", AfterHash: "h1"})

	current = filepath.Join(dir, "s2.jsonl") // NewSession switched files
	if len(s.List()) != 0 {
		t.Fatal("fingerprints leaked across sessions")
	}
}

func TestFingerprintPreFlushCarryOver(t *testing.T) {
	dir := t.TempDir()
	path := "" // session not flushed yet
	s := NewFingerprintStore(func() string { return path })
	s.Upsert(FileFingerprint{Path: "/a.py", AfterHash: "h1"})

	path = filepath.Join(dir, "s1.jsonl") // first flush assigns the file
	fps := s.List()
	if len(fps) != 1 || fps[0].AfterHash != "h1" {
		t.Fatalf("pre-flush fingerprints lost: %+v", fps)
	}
}

func TestFingerprintIgnoresEmpty(t *testing.T) {
	s := NewFingerprintStore(func() string { return "" })
	s.Upsert(FileFingerprint{Path: "", AfterHash: "h"})
	s.Upsert(FileFingerprint{Path: "/a", AfterHash: ""})
	if len(s.List()) != 0 {
		t.Fatal("empty fingerprints stored")
	}
}
