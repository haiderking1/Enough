package skills

import (
	"os"

	"github.com/enough/enough/backend/enoughhome"
)

// EnsureBootstrapped seeds the skills library on first launch, mirroring
// Hermes' idempotent sync_skills(quiet=True) on every chat/TUI entry.
// Failures are swallowed — skills enhance the agent but are not hard-required.
func EnsureBootstrapped() {
	home := enoughhome.HomeDir()
	_ = os.MkdirAll(home, 0o700)
	_ = os.MkdirAll(SkillsDir(), 0o700)
	_ = ExtractEnoughSkillIfMissing()
	_, _ = SyncSkills(true)
}
