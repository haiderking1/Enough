package session

import (
	"testing"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/opencode"
)

func TestPrepareManualCompactionSmallSession(t *testing.T) {
	h := &entryHelper{}
	path := []FileEntry{
		h.createMessageEntry(opencode.Message{Role: "user", Content: opencode.StringContent("hi")}),
		h.createMessageEntry(createAssistantMessage("hello", createMockUsage(10, 10, 0, 0))),
	}

	settings := config.CompactionSettings{
		Enabled:          true,
		ReserveTokens:    16384,
		KeepRecentTokens: 20000,
	}

	prep := PrepareManualCompaction(path, settings)
	if prep == nil {
		t.Fatal("expected manual prep on short session")
	}
	if !prep.IsSplitTurn && len(prep.MessagesToSummarize) == 0 && len(prep.TurnPrefixMessages) == 0 {
		t.Fatal("expected something to summarize on manual compact")
	}
}

func TestPrepareCompactionKeepsRecentBudget(t *testing.T) {
	h := &entryHelper{}
	path := []FileEntry{
		h.createMessageEntry(opencode.Message{Role: "user", Content: opencode.StringContent("hi")}),
		h.createMessageEntry(createAssistantMessage("hello", createMockUsage(10, 10, 0, 0))),
	}

	settings := config.CompactionSettings{
		Enabled:          true,
		ReserveTokens:    16384,
		KeepRecentTokens: 20000,
	}

	prep := PrepareCompaction(path, settings)
	if prep == nil {
		t.Fatal("expected auto prep object")
	}
	if prep.FirstKeptEntryID != path[0].ID {
		t.Fatalf("auto prep should keep from first entry on tiny session, got %s", prep.FirstKeptEntryID)
	}
}
