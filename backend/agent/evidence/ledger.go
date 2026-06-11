// Package evidence implements the append-only evidence ledger for the v2
// proof-obligation runtime. Only the runtime appends entries — never the
// model. Entries are facts recorded from successful tool executions and are
// the sole input to obligation closure and tool-guard decisions.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type Kind string

const (
	KindReadFile     Kind = "read_file"
	KindWriteFile    Kind = "write_file"
	KindEditFile     Kind = "edit_file"
	KindCommandRun   Kind = "command_run"
	KindSearch       Kind = "search"
	KindWebSearch    Kind = "web_search"
	KindVerifierPass Kind = "verifier_pass"
	KindVerifierFail Kind = "verifier_fail"
)

type Entry struct {
	ID        string          `json:"id"`
	TurnID    string          `json:"turn_id"`
	Kind      Kind            `json:"kind"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// Read-credit provenance. Only the runtime appends non-tool sources.
const (
	ReadSourceTool       = ""           // a real read_file tool call
	ReadSourceContinuity = "continuity" // runtime seed: prior-turn agent mutation, hash verified
	ReadSourceAuthor     = "author"     // same-turn mutation: the author knows the content
)

// ReadFilePayload records read credit for a path.
type ReadFilePayload struct {
	Path        string `json:"path"`
	ContentHash string `json:"content_hash"`
	LineCount   int    `json:"line_count"`
	Source      string `json:"source,omitempty"`
}

// MutationPayload records a successful write_file or edit_file.
type MutationPayload struct {
	Path       string `json:"path"`
	BeforeHash string `json:"before_hash"` // empty when the file did not exist
	AfterHash  string `json:"after_hash"`
}

// CommandRunPayload records a bash execution, success or failure.
type CommandRunPayload struct {
	Command    string `json:"command"`
	Cwd        string `json:"cwd"`
	ExitCode   int    `json:"exit_code"`
	OutputHash string `json:"output_hash"`
	DurationMs int64  `json:"duration_ms"`
}

// VerifierPayload records a verifier pass or fail for a turn.
type VerifierPayload struct {
	TurnID   string   `json:"turn_id"`
	Failures []string `json:"failures,omitempty"`
}

// Ledger is an append-only per-turn record. Safe for concurrent use.
type Ledger struct {
	mu      sync.Mutex
	turnID  string
	entries []Entry
	seq     int
}

func NewLedger(turnID string) *Ledger {
	return &Ledger{turnID: turnID}
}

func (l *Ledger) TurnID() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.turnID
}

// Append records a new entry and returns it. The payload must marshal to JSON.
func (l *Ledger) Append(kind Kind, payload any) (Entry, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Entry{}, fmt.Errorf("evidence payload: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq++
	e := Entry{
		ID:        fmt.Sprintf("ev_%d", l.seq),
		TurnID:    l.turnID,
		Kind:      kind,
		Timestamp: time.Now(),
		Payload:   raw,
	}
	l.entries = append(l.entries, e)
	return e, nil
}

// HasRead reports whether a read_file entry for path exists in this turn.
// Paths are compared exactly; callers must resolve to absolute paths first.
func (l *Ledger) HasRead(path string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range l.entries {
		if e.Kind != KindReadFile {
			continue
		}
		var p ReadFilePayload
		if json.Unmarshal(e.Payload, &p) == nil && p.Path == path {
			return true
		}
	}
	return false
}

// NoteAuthorCredit grants read credit for a path the agent just mutated this
// turn: the author knows the content it wrote, so a subsequent edit in the
// same turn must not demand a redundant read_file.
func (l *Ledger) NoteAuthorCredit(path, contentHash string) {
	_, _ = l.Append(KindReadFile, ReadFilePayload{
		Path:        path,
		ContentHash: contentHash,
		Source:      ReadSourceAuthor,
	})
}

// MutatedPaths returns the distinct paths touched by write/edit entries,
// in first-mutation order.
func (l *Ledger) MutatedPaths() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, e := range l.entries {
		if e.Kind != KindWriteFile && e.Kind != KindEditFile {
			continue
		}
		var p MutationPayload
		if json.Unmarshal(e.Payload, &p) == nil && p.Path != "" && !seen[p.Path] {
			seen[p.Path] = true
			out = append(out, p.Path)
		}
	}
	return out
}

// Entries returns a copy of all entries in append order.
func (l *Ledger) Entries() []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Entry, len(l.entries))
	copy(out, l.entries)
	return out
}

// Count returns the number of entries.
func (l *Ledger) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// HashBytes returns the hex SHA256 of data.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
