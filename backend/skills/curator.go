package skills

// Curator — background skill maintenance (deterministic phase + state).
//
// The curator is inactivity-triggered (no cron daemon): when the agent is
// idle and the last run was longer than interval_hours ago, a pass runs.
// This file holds the persistent scheduler state, the static gates, and the
// pure (no-LLM) lifecycle transitions. The LLM review fork lives in
// backend/agent/curator.go (the agent package owns model access).
//
// Strict invariants:
//   - Only touches agent-created skills (created_by == "agent" in
//     .usage.json), except bundled skills under the prune_builtins path.
//   - The bundled `enough` skill is protected: NEVER archived or
//     consolidated, regardless of any flag.
//   - Never auto-deletes — only archives. Archive is recoverable.
//   - Pinned skills bypass all auto-transitions.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/enough/enough/backend/config"
)

// CuratorProtectedBuiltins are bundled skills the curator must never archive
// or consolidate, regardless of prune_builtins, pin state, or LLM judgment.
var CuratorProtectedBuiltins = map[string]bool{
	"enough": true,
}

// CuratorState is the persistent scheduler + status record
// (~/.enough/skills/.curator_state).
type CuratorState struct {
	LastRunAt              string  `json:"last_run_at,omitempty"`
	LastRunDurationSeconds float64 `json:"last_run_duration_seconds,omitempty"`
	LastRunSummary         string  `json:"last_run_summary,omitempty"`
	LastReportPath         string  `json:"last_report_path,omitempty"`
	Paused                 bool    `json:"paused"`
	RunCount               int     `json:"run_count"`
}

func CuratorStatePath() string {
	return filepath.Join(SkillsDir(), ".curator_state")
}

// CuratorSuppressedPath lists bundled skills the curator archived, so a
// future re-seed of bundled skills keeps them archived.
func CuratorSuppressedPath() string {
	return filepath.Join(SkillsDir(), ".curator_suppressed")
}

func LoadCuratorState() CuratorState {
	var st CuratorState
	data, err := os.ReadFile(CuratorStatePath())
	if err != nil {
		return st
	}
	_ = json.Unmarshal(data, &st)
	return st
}

func SaveCuratorState(st CuratorState) {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	_ = atomicWrite(CuratorStatePath(), data)
}

func SetCuratorPaused(paused bool) {
	st := LoadCuratorState()
	st.Paused = paused
	SaveCuratorState(st)
}

// IsProtectedBuiltin reports whether the curator must never touch this skill.
func IsProtectedBuiltin(name string) bool {
	return CuratorProtectedBuiltins[name]
}

// IsBundledSkillName reports whether the skill ships with Enough. Currently
// only the `enough` skill is bundled (see bundle.go).
func IsBundledSkillName(name string) bool {
	return name == "enough"
}

// loadSuppressed returns the set of bundled skills previously archived by the
// curator.
func loadSuppressed() map[string]bool {
	out := make(map[string]bool)
	data, err := os.ReadFile(CuratorSuppressedPath())
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out[line] = true
		}
	}
	return out
}

// MarkSuppressed records a bundled skill the curator archived.
func MarkSuppressed(name string) {
	sup := loadSuppressed()
	if sup[name] {
		return
	}
	sup[name] = true
	var lines []string
	for n := range sup {
		lines = append(lines, n)
	}
	_ = atomicWrite(CuratorSuppressedPath(), []byte(strings.Join(lines, "\n")+"\n"))
}

// IsSuppressed reports whether the curator previously archived this bundled
// skill (so re-seeds keep it archived).
func IsSuppressed(name string) bool {
	return loadSuppressed()[name]
}

// ShouldRunCurator evaluates the static gates: enabled, not paused, and
// last_run_at older than the interval. First-run behavior: when there is no
// last_run_at, the state is seeded to now and the first real pass is deferred
// by one full interval — the curator should never mutate the library on the
// first tick after install. Explicit /curator-run bypasses this gate.
func ShouldRunCurator(cfg config.CuratorSettings, now time.Time) bool {
	if !cfg.Enabled {
		return false
	}
	st := LoadCuratorState()
	if st.Paused {
		return false
	}
	if st.LastRunAt == "" {
		st.LastRunAt = now.UTC().Format(time.RFC3339)
		st.LastRunSummary = "deferred first run — curator seeded, will run after one interval; use /curator-run dry-run to preview now"
		SaveCuratorState(st)
		return false
	}
	last, err := time.Parse(time.RFC3339, st.LastRunAt)
	if err != nil {
		return false
	}
	interval := time.Duration(cfg.IntervalHours) * time.Hour
	return now.Sub(last) >= interval
}

// CuratorTransitionCounts reports what the deterministic pass changed.
type CuratorTransitionCounts struct {
	Checked     int
	MarkedStale int
	Archived    int
	Reactivated int
}

func (c CuratorTransitionCounts) Summary() string {
	var parts []string
	if c.MarkedStale > 0 {
		parts = append(parts, fmt.Sprintf("%d marked stale", c.MarkedStale))
	}
	if c.Archived > 0 {
		parts = append(parts, fmt.Sprintf("%d archived", c.Archived))
	}
	if c.Reactivated > 0 {
		parts = append(parts, fmt.Sprintf("%d reactivated", c.Reactivated))
	}
	if len(parts) == 0 {
		return "no changes"
	}
	return strings.Join(parts, ", ")
}

// ApplyAutomaticTransitions walks every curator-managed skill and moves
// active/stale/archived based on the latest real activity timestamp. Pinned
// skills are never touched; protected builtins are never candidates. Never
// deletes — archive only.
func ApplyAutomaticTransitions(cfg config.CuratorSettings, now time.Time) CuratorTransitionCounts {
	staleCutoff := now.Add(-time.Duration(cfg.StaleAfterDays) * 24 * time.Hour)
	archiveCutoff := now.Add(-time.Duration(cfg.ArchiveAfterDays) * 24 * time.Hour)

	var counts CuratorTransitionCounts

	for _, row := range AgentCreatedReport() {
		counts.Checked++
		if row.Pinned || IsProtectedBuiltin(row.Name) {
			continue
		}

		// If never active, anchor on created_at so new skills don't
		// immediately archive themselves.
		anchor, ok := parseIso(row.LastActivityAt)
		if !ok {
			created := row.CreatedAt
			if t, cok := parseIso(&created); cok {
				anchor = t
			} else {
				anchor = now
			}
		}

		current := row.State
		switch {
		case !anchor.After(archiveCutoff) && current != "archived":
			if ok, _ := ArchiveSkill(row.Name); ok {
				counts.Archived++
			}
		case !anchor.After(staleCutoff) && current == "active":
			SetState(row.Name, "stale")
			counts.MarkedStale++
		case anchor.After(staleCutoff) && current == "stale":
			// Skill got used again after being marked stale — reactivate.
			SetState(row.Name, "active")
			counts.Reactivated++
		}
	}

	return counts
}

// RenderCuratorCandidateList builds the agent-readable list of agent-created
// skills with usage stats for the LLM review prompt.
func RenderCuratorCandidateList() string {
	rows := AgentCreatedReport()
	var eligible []UsageReportRow
	for _, r := range rows {
		if IsProtectedBuiltin(r.Name) {
			continue
		}
		eligible = append(eligible, r)
	}
	if len(eligible) == 0 {
		return "No agent-created skills to review."
	}
	lines := []string{fmt.Sprintf("Agent-created skills (%d):\n", len(eligible))}
	for _, r := range eligible {
		pinned := "no"
		if r.Pinned {
			pinned = "yes"
		}
		lastActivity := "never"
		if r.LastActivityAt != nil && *r.LastActivityAt != "" {
			lastActivity = *r.LastActivityAt
		}
		lines = append(lines, fmt.Sprintf(
			"- %s  state=%s  pinned=%s  activity=%d  use=%d  view=%d  patches=%d  last_activity=%s",
			r.Name, r.State, pinned, r.ActivityCount, r.UseCount, r.ViewCount, r.PatchCount, lastActivity))
	}
	return strings.Join(lines, "\n")
}

// CuratorStatusString renders the /curator-status output.
func CuratorStatusString(cfg config.CuratorSettings) string {
	st := LoadCuratorState()
	var b strings.Builder
	b.WriteString("Curator status:\n")
	fmt.Fprintf(&b, "- enabled: %v\n", cfg.Enabled)
	fmt.Fprintf(&b, "- paused: %v\n", st.Paused)
	if st.LastRunAt == "" {
		b.WriteString("- last run: never\n")
	} else {
		fmt.Fprintf(&b, "- last run: %s\n", st.LastRunAt)
	}
	fmt.Fprintf(&b, "- runs: %d\n", st.RunCount)
	fmt.Fprintf(&b, "- interval: %dh, min idle: %.1fh, stale after: %dd, archive after: %dd, prune builtins: %v\n",
		cfg.IntervalHours, cfg.MinIdleHours, cfg.StaleAfterDays, cfg.ArchiveAfterDays, cfg.PruneBuiltins)
	if st.LastRunSummary != "" {
		fmt.Fprintf(&b, "- last summary: %s\n", st.LastRunSummary)
	}
	if st.LastReportPath != "" {
		fmt.Fprintf(&b, "- last report: %s\n", st.LastReportPath)
	}
	return strings.TrimRight(b.String(), "\n")
}
