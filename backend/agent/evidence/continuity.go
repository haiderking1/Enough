package evidence

import (
	"os"
	"strings"
)

// Fingerprint is the cross-turn record of an agent mutation: the content
// hash a path had after this agent last successfully wrote or edited it.
type Fingerprint struct {
	Path      string // absolute path
	AfterHash string // hex SHA256 after the mutation
}

// SeedContinuityReads grants read credit for agent-authored paths whose
// on-disk content still matches the last recorded mutation fingerprint. The
// guard stays hostile to everything else: a missing file, a hash mismatch
// (external edit), or a path the agent never wrote gets no credit and still
// requires a real read_file this turn. Returns the number of seeded entries.
func SeedContinuityReads(ledger *Ledger, fps []Fingerprint) int {
	if ledger == nil {
		return 0
	}
	n := 0
	for _, fp := range fps {
		if fp.Path == "" || fp.AfterHash == "" {
			continue
		}
		data, err := os.ReadFile(fp.Path)
		if err != nil {
			continue // deleted or unreadable: no credit
		}
		if HashBytes(data) != fp.AfterHash {
			continue // external edit: forces a real read
		}
		lines := strings.Count(string(data), "\n")
		if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
			lines++
		}
		if _, err := ledger.Append(KindReadFile, ReadFilePayload{
			Path:        fp.Path,
			ContentHash: fp.AfterHash,
			LineCount:   lines,
			Source:      ReadSourceContinuity,
		}); err == nil {
			n++
		}
	}
	return n
}
