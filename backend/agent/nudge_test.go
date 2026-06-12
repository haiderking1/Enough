package agent

import (
	"testing"

	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/session"
)

func newTestSession(t *testing.T) *session.Manager {
	t.Helper()
	sm, err := session.ContinueRecent(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return sm
}

func TestNudgeCounterHydrationFromHistory(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	sm := newTestSession(t)

	// 13 prior user turns with nudge_interval 10 → counter resumes at 3.
	for i := 0; i < 13; i++ {
		_ = sm.AppendMessage(opencode.Message{Role: "user", Content: opencode.StringContent("q")})
		_ = sm.AppendMessage(opencode.Message{Role: "assistant", Content: opencode.StringContent("a")})
	}

	a := &Agent{cfg: testRuntime(), session: sm}
	a.hydrateNudgeCountersLocked()
	if a.userTurnCount != 13 {
		t.Fatalf("userTurnCount = %d, want 13", a.userTurnCount)
	}
	if a.turnsSinceMemory != 3 {
		t.Fatalf("turnsSinceMemory = %d, want 13 %% 10 = 3", a.turnsSinceMemory)
	}

	// Idempotent: a second call must not re-hydrate.
	a.turnsSinceMemory = 5
	a.hydrateNudgeCountersLocked()
	if a.turnsSinceMemory != 5 {
		t.Fatal("hydration ran twice")
	}
}

func TestNudgeCounterHydrationEmptySession(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	a := &Agent{cfg: testRuntime(), session: newTestSession(t)}
	a.hydrateNudgeCountersLocked()
	if a.userTurnCount != 0 || a.turnsSinceMemory != 0 {
		t.Fatal("empty session should leave counters at zero")
	}
}

func TestSessionPersistsCachedSystemPrompt(t *testing.T) {
	t.Setenv("ENOUGH_HOME", t.TempDir())
	sm := newTestSession(t)

	// Persistence requires an assistant message in the session (the manager
	// defers flushing until then).
	_ = sm.AppendMessage(opencode.Message{Role: "user", Content: opencode.StringContent("hi")})
	_ = sm.AppendMessage(opencode.Message{Role: "assistant", Content: opencode.StringContent("hello")})

	if err := sm.SetSystemPrompt("THE CACHED PROMPT"); err != nil {
		t.Fatal(err)
	}
	if sm.StoredSystemPrompt() != "THE CACHED PROMPT" {
		t.Fatal("stored prompt not readable in-memory")
	}

	// The prompt entry never enters the LLM context.
	for _, msg := range sm.Messages() {
		if opencode.ContentString(msg) == "THE CACHED PROMPT" {
			t.Fatal("system prompt entry leaked into LLM messages")
		}
	}

	// A fresh manager on the same cwd restores it.
	sm2, err := session.ContinueRecent(sm.CWD())
	if err != nil {
		t.Fatal(err)
	}
	if sm2.SessionID() != sm.SessionID() {
		t.Fatalf("expected to resume the same session")
	}
	if sm2.StoredSystemPrompt() != "THE CACHED PROMPT" {
		t.Fatal("stored prompt not restored from disk")
	}

	// Agent.New replays the stored prompt verbatim.
	a := New(testRuntime(), sm.CWD(), sm2)
	if a.systemPrompt() != "THE CACHED PROMPT" {
		t.Fatal("resumed agent did not replay stored prompt verbatim")
	}

	// /new clears it and rebuilds.
	if err := a.Reset(); err != nil {
		t.Fatal(err)
	}
	if a.systemPrompt() == "THE CACHED PROMPT" {
		t.Fatal("reset must rebuild the prompt")
	}
}
