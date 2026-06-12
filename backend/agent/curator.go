package agent

// Curator orchestration — deterministic phase delegation + the LLM review
// fork. The persistent state, gates and pure transitions live in
// backend/skills/curator.go; this file owns model access.
//
// Trigger model (no cron): the TUI calls MaybeRunCurator on session start and
// on idle ticks. Gates: curator.enabled, not paused, last run older than
// interval_hours, and (when a measurement is provided) agent idle for at
// least min_idle_hours. Explicit /curator-run bypasses the gates.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/skills"
)

// curatorMaxIterations bounds the curator fork's model calls per pass.
const curatorMaxIterations = 8

// curatorToolWhitelist — read tools, skill management, and bash for archive
// moves. Enforced in guardTool.
var curatorToolWhitelist = map[string]bool{
	"skills_list":  true,
	"skill_view":   true,
	"skill_manage": true,
	"bash":         true,
}

// curatorDryRunBanner — ported from Hermes agent/curator.py, paths and
// commands adapted to Enough.
const curatorDryRunBanner = "═══════════════════════════════════════════════════════════════\n" +
	"DRY-RUN — REPORT ONLY. DO NOT MUTATE THE SKILL LIBRARY.\n" +
	"═══════════════════════════════════════════════════════════════\n" +
	"\n" +
	"This is a PREVIEW pass. Follow every instruction below EXCEPT:\n" +
	"\n" +
	"  • DO NOT call skill_manage with action=patch, create, delete, " +
	"write_file, or remove_file.\n" +
	"  • DO NOT call bash to mv skill directories into .archive/.\n" +
	"  • DO NOT call bash to mv, cp, rm, or rewrite any file under " +
	"~/.enough/skills/.\n" +
	"  • skills_list and skill_view are FINE — read as much as you need.\n" +
	"\n" +
	"Your output IS the deliverable. Produce the exact same " +
	"human-readable summary and structured YAML block you would " +
	"produce on a live run — but describe the actions you WOULD take, " +
	"not actions you took. A downstream reviewer will read the report " +
	"and decide whether to approve a live run with /curator-run (no flag).\n" +
	"\n" +
	"If you accidentally take a mutating action, say so explicitly in " +
	"the summary so the reviewer can revert it.\n" +
	"═══════════════════════════════════════════════════════════════"

// curatorReviewPrompt — ported from Hermes agent/curator.py
// (CURATOR_REVIEW_PROMPT), adapted to Enough: ~/.enough/skills paths, bash
// instead of terminal, /curator-* commands, protected built-in `enough-agent`, no
// cron-job rewriting.
const curatorReviewPrompt = "You are running as Enough's background skill CURATOR. This is an " +
	"UMBRELLA-BUILDING consolidation pass, not a passive audit and not a " +
	"duplicate-finder.\n\n" +
	"The goal of the skill collection is a LIBRARY OF CLASS-LEVEL " +
	"INSTRUCTIONS AND EXPERIENTIAL KNOWLEDGE. A collection of hundreds of " +
	"narrow skills where each one captures one session's specific bug is " +
	"a FAILURE of the library — not a feature. An agent searching skills " +
	"matches on descriptions, not on exact names; one broad umbrella " +
	"skill with labeled subsections beats five narrow siblings for " +
	"discoverability, not the other way around.\n\n" +
	"The right target shape is CLASS-LEVEL skills with rich SKILL.md " +
	"bodies + `references/`, `templates/`, and `scripts/` subfiles for " +
	"session-specific detail — not one-session-one-skill micro-entries.\n\n" +
	"Hard rules — do not violate:\n" +
	"1. DO NOT touch bundled or externally-installed skills. The candidate " +
	"list below is already filtered to agent-created skills only.\n" +
	"2. DO NOT delete any skill. Archiving (moving the skill's directory " +
	"into ~/.enough/skills/.archive/) is the maximum destructive action. " +
	"Archives are recoverable; deletion is not.\n" +
	"3. DO NOT touch skills shown as pinned=yes. Skip them entirely.\n" +
	"3b. DO NOT archive, delete, consolidate, move, or otherwise modify any " +
	"skill named in the protected built-ins list (currently: enough-agent). These " +
	"back load-bearing UX and are filtered out of the candidate list below — " +
	"never resurrect one as an archive or absorb target.\n" +
	"4. DO NOT use usage counters as a reason to skip consolidation. The " +
	"counters are new and often mostly zero. Judge overlap on CONTENT, " +
	"not on use_count. 'use=0' is not evidence a skill is valuable; it's " +
	"absence of evidence either way.\n" +
	"5. DO NOT reject consolidation on the grounds that 'each skill has " +
	"a distinct trigger'. Pairwise distinctness is the wrong bar. The " +
	"right bar is: 'would a human maintainer write this as N separate " +
	"skills, or as one skill with N labeled subsections?' When the " +
	"answer is the latter, merge.\n\n" +
	"How to work — not optional:\n" +
	"1. Scan the full candidate list. Identify PREFIX CLUSTERS (skills " +
	"sharing a first word or domain keyword).\n" +
	"2. For each cluster with 2+ members, do NOT ask 'are these pairs " +
	"overlapping?' — ask 'what is the UMBRELLA CLASS these skills all " +
	"serve? Would a maintainer name that class and write one skill for " +
	"it?' If yes, pick (or create) the umbrella and absorb the siblings " +
	"into it.\n" +
	"3. Three ways to consolidate — use the right one per cluster:\n" +
	"   a. MERGE INTO EXISTING UMBRELLA — one skill in the cluster is " +
	"already broad enough to be the umbrella. Patch it to add a labeled " +
	"section for each sibling's unique insight, then archive the " +
	"siblings.\n" +
	"   b. CREATE A NEW UMBRELLA SKILL.md — no existing member is broad " +
	"enough. Use skill_manage action=create to write a new class-level " +
	"skill whose SKILL.md covers the shared workflow and has short " +
	"labeled subsections. Archive the now-absorbed narrow siblings.\n" +
	"   c. DEMOTE TO REFERENCES/TEMPLATES/SCRIPTS — a sibling has " +
	"narrow-but-valuable session-specific content. Move it into the " +
	"umbrella's appropriate support directory:\n" +
	"      • `references/<topic>.md` for session-specific detail OR " +
	"condensed knowledge banks (quoted research, API docs excerpts, " +
	"domain notes, provider quirks, reproduction recipes)\n" +
	"      • `templates/<name>.<ext>` for starter files meant to be " +
	"copied and modified\n" +
	"      • `scripts/<name>.<ext>` for statically re-runnable actions " +
	"(verification scripts, fixture generators, probes)\n" +
	"      Then archive the old sibling. Use `bash` with `mkdir -p " +
	"~/.enough/skills/<umbrella>/references/ && mv ... <umbrella>/" +
	"references/<topic>.md` (or templates/ / scripts/).\n\n" +
	"Package integrity — not optional:\n" +
	"Before demoting or archiving a skill, inspect it as a COMPLETE " +
	"directory package, not just SKILL.md. A skill root may include " +
	"`references/`, `templates/`, `scripts/`, and `assets/`; `skill_view` " +
	"discovers those relative to the skill root. A reference markdown file " +
	"inside another skill is NOT a new skill root and does not get its own " +
	"linked-file discovery.\n" +
	"If the source skill has support files OR SKILL.md contains relative " +
	"links such as `references/...`, `templates/...`, `scripts/...`, or " +
	"`assets/...`, DO NOT flatten only SKILL.md into " +
	"`<umbrella>/references/<old>.md`. Choose one safe path instead:\n" +
	"   • keep it as a standalone skill, OR\n" +
	"   • fully merge it by re-homing every needed support file into the " +
	"umbrella's canonical `references/`, `templates/`, `scripts/`, or " +
	"`assets/` directories AND rewrite the destination instructions to " +
	"the new paths, OR\n" +
	"   • archive the entire original skill package unchanged.\n" +
	"Never leave archived/demoted instructions pointing at files that were " +
	"left behind under the old skill directory.\n" +
	"4. Also flag skills whose NAME is too narrow (contains a PR number, " +
	"a feature codename, a specific error string, an 'audit' / " +
	"'diagnosis' / 'salvage' session artifact). These almost always " +
	"belong as a subsection or support file under a class-level umbrella.\n" +
	"5. Iterate. After one consolidation round, scan the remaining set " +
	"and look for the NEXT umbrella opportunity.\n\n" +
	"Your toolset:\n" +
	"  - skills_list, skill_view        — read the current landscape\n" +
	"  - skill_manage action=patch      — add sections to the umbrella\n" +
	"  - skill_manage action=create     — create a new umbrella SKILL.md\n" +
	"  - skill_manage action=write_file — add a references/, templates/, " +
	"or scripts/ file under an existing skill (the skill must already " +
	"exist)\n" +
	"  - skill_manage action=delete     — archive a skill. MUST pass " +
	"`absorbed_into=<umbrella>` when you've merged its content into another " +
	"skill, or `absorbed_into=\"\"` when you're truly pruning with no " +
	"forwarding target.\n" +
	"  - bash                           — mv a sibling into the archive " +
	"OR move its content into a support subfile\n\n" +
	"'keep' is a legitimate decision ONLY when the skill is already a " +
	"class-level umbrella and none of the proposed merges would improve " +
	"discoverability. 'This is narrow but distinct from its siblings' " +
	"is NOT a reason to keep — it's a reason to move it under an " +
	"umbrella as a subsection or support file.\n\n" +
	"When done, write a human summary AND a structured machine-readable " +
	"block so downstream tooling can distinguish consolidation from " +
	"pruning. Format EXACTLY:\n\n" +
	"## Structured summary (required)\n" +
	"```yaml\n" +
	"consolidations:\n" +
	"  - from: <old-skill-name>\n" +
	"    into: <umbrella-skill-name>\n" +
	"    reason: <one short sentence — why merged, not just 'similar'>\n" +
	"prunings:\n" +
	"  - name: <skill-name>\n" +
	"    reason: <one short sentence — why archived with no merge target>\n" +
	"```\n\n" +
	"Every skill you moved to .archive/ MUST appear in exactly one of the " +
	"two lists. If you consolidated X into umbrella Y, X goes under " +
	"`consolidations` with `into: Y`. If you archived X with no absorption " +
	"— truly stale, irrelevant, or obsolete — X goes under `prunings`. " +
	"Leave a list empty (`consolidations: []`) if none. Do not omit the " +
	"block. The block comes AFTER your human-readable summary of clusters " +
	"processed, patches made, and decisions left alone."

const curatorPruneBuiltinsNote = "\n\nPRUNE-BUILTINS MODE IS ON: bundled built-in skills " +
	"MAY be archived for staleness/irrelevance, overriding hard rule #1 " +
	"for bundled skills ONLY — EXCEPT the protected built-ins (enough-agent), " +
	"which remain strictly off-limits. Treat a stale built-in the same as " +
	"a stale agent-created skill: archive it (never delete)."

// CuratorRunResult is what a curator pass reports back to the caller.
type CuratorRunResult struct {
	StartedAt   time.Time
	AutoCounts  skills.CuratorTransitionCounts
	AutoSummary string
}

// MaybeRunCurator runs a curator pass when all gates pass. idleFor < 0 means
// "no measurement" (only the static gates are enforced, e.g. at session
// start). Returns true when a pass was started. Never blocks on the LLM
// phase (it runs in a goroutine) and never panics.
func MaybeRunCurator(cfg config.Runtime, idleFor time.Duration, notify func(string)) bool {
	defer func() { _ = recover() }()

	if !skills.ShouldRunCurator(cfg.Curator, time.Now()) {
		return false
	}
	if idleFor >= 0 {
		minIdle := time.Duration(cfg.Curator.MinIdleHours * float64(time.Hour))
		if idleFor < minIdle {
			return false
		}
	}
	RunCuratorReview(cfg, false, false, notify)
	return true
}

// RunCuratorReview executes a single curator pass:
//  1. Deterministic state transitions (pure, no LLM). Skipped on dry runs.
//  2. An LLM review fork over the agent-created candidate list.
//  3. State + report updates, then notify with a user-visible summary.
//
// When synchronous is false the LLM phase runs in a daemon goroutine and the
// call returns immediately after phase 1.
func RunCuratorReview(cfg config.Runtime, dryRun, synchronous bool, notify func(string)) CuratorRunResult {
	start := time.Now().UTC()

	var counts skills.CuratorTransitionCounts
	if dryRun {
		counts.Checked = len(skills.AgentCreatedReport())
	} else {
		counts = skills.ApplyAutomaticTransitions(cfg.Curator, start)
	}
	autoSummary := counts.Summary()

	// Persist state before the LLM pass so a crash mid-review still records
	// the run and doesn't immediately re-trigger. Dry runs don't bump
	// last_run_at or run_count — a preview shouldn't push out the next
	// scheduled real pass.
	prefix := "auto: "
	if dryRun {
		prefix = "dry-run auto: "
	}
	st := skills.LoadCuratorState()
	if !dryRun {
		st.LastRunAt = start.Format(time.RFC3339)
		st.RunCount++
	}
	st.LastRunSummary = prefix + autoSummary
	skills.SaveCuratorState(st)

	llmPass := func() {
		defer func() { _ = recover() }()
		if notify != nil {
			notify("🧹 Curator review running…")
		}

		finalSummary := runCuratorLLMPass(cfg, dryRun, prefix, autoSummary, notify)

		st := skills.LoadCuratorState()
		st.LastRunDurationSeconds = time.Since(start).Seconds()
		st.LastRunSummary = finalSummary

		if reportPath := writeCuratorReport(start, finalSummary); reportPath != "" {
			st.LastReportPath = reportPath
		}
		skills.SaveCuratorState(st)

		if notify != nil {
			notify("🧹 Curator: " + finalSummary)
		}
	}

	if synchronous {
		llmPass()
	} else {
		go llmPass()
	}

	return CuratorRunResult{StartedAt: start, AutoCounts: counts, AutoSummary: autoSummary}
}

// runCuratorLLMPass spawns the curator fork and returns the run summary.
func runCuratorLLMPass(cfg config.Runtime, dryRun bool, prefix, autoSummary string, notify func(string)) string {
	candidateList := skills.RenderCuratorCandidateList()
	if strings.Contains(candidateList, "No agent-created skills") {
		return prefix + autoSummary + "; llm: skipped (no candidates)"
	}

	prompt := curatorReviewPrompt
	if cfg.Curator.PruneBuiltins {
		prompt += curatorPruneBuiltinsNote
	}
	if dryRun {
		prompt = curatorDryRunBanner + "\n\n" + prompt
	}
	prompt += "\n\n" + candidateList

	childCfg := cfg
	childCfg.Compaction.Enabled = false
	childCfg.Evidence.Enabled = false
	// The curator must never spawn its own review.
	childCfg.Memory.NudgeInterval = 0
	childCfg.Memory.SkillNudgeInterval = 0

	curator := &Agent{
		cfg:           childCfg,
		client:        opencode.NewClient(childCfg.Endpoint, childCfg.APIKey, childCfg.Model),
		workDir:       skills.SkillsDir(),
		session:       nil,
		allowedTools:  curatorToolWhitelist,
		writeOrigin:   WriteOriginBackgroundReview,
		maxIterations: curatorMaxIterations,
		swarmDepth:    maxSwarmDepth,
		notify:        notify,
	}
	curator.cachedSystemPrompt = "You are Enough's background skill curator. You maintain the " +
		"skill library at ~/.enough/skills/. Follow the task instructions exactly."
	curator.messages = []opencode.Message{
		{Role: "system", Content: opencode.StringContent(curator.cachedSystemPrompt)},
		{Role: "user", Content: opencode.StringContent(prompt)},
	}

	if err := curator.runLoop(context.Background()); err != nil {
		return fmt.Sprintf("%s%s; llm: error (%v)", prefix, autoSummary, err)
	}

	final := ""
	for i := len(curator.messages) - 1; i >= 0; i-- {
		if curator.messages[i].Role == "assistant" && len(curator.messages[i].ToolCalls) == 0 {
			final = strings.TrimSpace(opencode.ContentString(curator.messages[i]))
			break
		}
	}
	llmSummary := final
	if len(llmSummary) > 240 {
		llmSummary = llmSummary[:240] + "…"
	}
	if llmSummary == "" {
		llmSummary = "no change"
	}
	return prefix + autoSummary + "; llm: " + llmSummary
}

// writeCuratorReport writes a per-run report under ~/.enough/logs/curator/.
// Best-effort; returns the report path or "".
func writeCuratorReport(start time.Time, summary string) string {
	root := filepath.Join(skills.HomeDir(), "logs", "curator")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return ""
	}
	path := filepath.Join(root, start.Format("20060102-150405")+"-REPORT.md")
	content := fmt.Sprintf("# Curator run — %s\n\n%s\n\n## Recovery\n\n"+
		"- Restore an archived skill: /skill-restore <name>\n"+
		"- All archives live under ~/.enough/skills/.archive/ and are recoverable by mv\n",
		start.Format(time.RFC3339), summary)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return ""
	}
	return path
}
