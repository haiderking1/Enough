package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderAgentSwarmBlock(t *testing.T) {
	styles := NewStyles()
	args := `{
		"shared_context": "Current system already supports spawned subagents with roles.",
		"tasks": [
			{"id": "Professor Farnsworth", "prompt": "Explore Structured Interagent Mailboxes."},
			{"id": "Wernstrom", "prompt": "Explore Agent Tree Control Plane."},
			{"id": "Zoidberg", "prompt": "Explore Shared Blackboard for Agent Coordination."}
		]
	}`
	lines := renderAgentSwarmBlock(styles, toolRow{
		Kind:    toolKindSwarm,
		Args:    args,
		Pending: true,
	}, 100, false, 3)
	plain := ansi.Strip(strings.Join(lines, "\n"))

	for _, want := range []string{
		"Spawned",
		"Professor Farnsworth",
		"[worker]",
		"Structured Interagent Mailboxes",
		"Wernstrom",
		"Zoidberg",
		"Context:",
		"Current system already supports spawned subagents",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("missing %q in:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "Updated") {
		t.Fatalf("swarm block should not include group header: %q", plain)
	}
}

func TestRenderWebSearchBlock(t *testing.T) {
	styles := NewStyles()
	lines := renderWebSearchBlock(styles, toolRow{
		Kind:    toolKindWeb,
		Target:  "interesting random fact",
		Pending: true,
	}, 80, false)
	plain := ansi.Strip(strings.Join(lines, "\n"))

	if !strings.Contains(plain, "Search") || !strings.Contains(plain, "interesting random fact") {
		t.Fatalf("unexpected web search render: %q", plain)
	}
	if !strings.Contains(plain, "└") {
		t.Fatalf("expected tree branch: %q", plain)
	}
}

func TestSpawnBulletAnimatesWhileRunning(t *testing.T) {
	styles := NewStyles()
	spinning := ansi.Strip(spawnBullet(styles, true, 1))
	static := ansi.Strip(spawnBullet(styles, false, 1))
	if spinning != "*" || static != "*" {
		t.Fatalf("bullet should stay * for alignment, got spinning=%q static=%q", spinning, static)
	}
}

func TestRenderAgentSwarmBlockShowsStatusWhenDone(t *testing.T) {
	styles := NewStyles()
	lines := renderAgentSwarmBlock(styles, toolRow{
		Kind:   toolKindSwarm,
		Args:   `{"tasks":[{"id":"a","prompt":"do thing"}]}`,
		Output: "## a [ok] (2 turns)\ndone",
	}, 100, false, 0)
	plain := ansi.Strip(strings.Join(lines, "\n"))
	if !strings.Contains(plain, "(ok)") {
		t.Fatalf("expected done status: %q", plain)
	}
	if strings.Contains(plain, "running") {
		t.Fatalf("should not show running when done: %q", plain)
	}
}

func TestSingleSwarmNoGroupHeader(t *testing.T) {
	styles := NewStyles()
	out := renderToolGroup(styles, []chatMsg{{
		toolName: "agent_swarm",
		toolArgs: `{"tasks":[{"id":"a","prompt":"do thing"}]}`,
	}}, 100, false, 0)
	if strings.Contains(out, "Updated") {
		t.Fatalf("single swarm should not show group header: %q", out)
	}
	if !strings.Contains(ansi.Strip(out), "Spawned") {
		t.Fatalf("expected spawn header: %q", out)
	}
}
