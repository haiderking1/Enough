package session

import (
	"encoding/json"
	"os"
	"strings"
	"sync"
	"time"
)

// FileFingerprint records the content hash a path had after the agent's last
// successful mutation of it. Paths and hashes only — never file contents.
type FileFingerprint struct {
	Path      string    `json:"path"`       // absolute path
	AfterHash string    `json:"after_hash"` // hex SHA256 after the mutation
	Kind      string    `json:"kind"`       // write_file | edit_file
	TurnID    string    `json:"turn_id"`
	Timestamp time.Time `json:"timestamp"`
}

// FingerprintStore holds the session-wide latest fingerprint per path (last
// mutation wins, across branches) and persists it to a sidecar file next to
// the session JSONL. The sidecar is independent of the message log, so
// compaction never touches it; loading a session loads its sidecar.
type FingerprintStore struct {
	mu         sync.Mutex
	pathFn     func() string // current sidecar path; "" until session flushes
	byPath     map[string]FileFingerprint
	loadedFrom string
}

// NewFingerprintStore creates a store whose sidecar location is resolved
// lazily via pathFn (the session file may not exist yet).
func NewFingerprintStore(pathFn func() string) *FingerprintStore {
	return &FingerprintStore{pathFn: pathFn, byPath: map[string]FileFingerprint{}}
}

// sidecarPath derives the fingerprint file from a session JSONL path.
func sidecarPath(sessionFile string) string {
	if sessionFile == "" {
		return ""
	}
	return strings.TrimSuffix(sessionFile, ".jsonl") + ".fingerprints.json"
}

// sync reloads from disk when the sidecar location changed (new/loaded
// session). Entries recorded before the session first flushed (no path yet)
// are carried over; entries from a previous session file are dropped.
func (s *FingerprintStore) syncLocked() {
	path := sidecarPath(s.pathFn())
	if path == s.loadedFrom {
		return
	}
	pending := s.byPath
	carryOver := s.loadedFrom == "" && len(pending) > 0

	s.loadedFrom = path
	s.byPath = map[string]FileFingerprint{}
	if path == "" {
		return
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s.byPath)
	}
	if carryOver {
		for p, fp := range pending { // pre-flush entries are newest: they win
			s.byPath[p] = fp
		}
	}
}

// Upsert records the latest mutation fingerprint for a path and persists the
// snapshot when the session has a file on disk.
func (s *FingerprintStore) Upsert(fp FileFingerprint) {
	if fp.Path == "" || fp.AfterHash == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncLocked()
	s.byPath[fp.Path] = fp

	if s.loadedFrom == "" {
		// Session not flushed yet; retry persisting on a later upsert/list.
		return
	}
	if data, err := json.MarshalIndent(s.byPath, "", " "); err == nil {
		_ = os.WriteFile(s.loadedFrom, data, 0o600)
	}
}

// List returns all fingerprints for the current session.
func (s *FingerprintStore) List() []FileFingerprint {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncLocked()
	out := make([]FileFingerprint, 0, len(s.byPath))
	for _, fp := range s.byPath {
		out = append(out, fp)
	}
	return out
}

// Fingerprints returns the manager's fingerprint store, bound to the current
// session file (it follows NewSession/load switches automatically).
func (m *Manager) Fingerprints() *FingerprintStore {
	m.fpOnce.Do(func() {
		m.fingerprints = NewFingerprintStore(func() string { return m.SessionFile() })
	})
	return m.fingerprints
}
