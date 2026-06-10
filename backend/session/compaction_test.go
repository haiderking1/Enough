package session

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/opencode"
)

func migrateV1ToV2(entries []FileEntry) {
	var prevID *string
	for i := range entries {
		entry := &entries[i]
		if entry.Type == TypeSession {
			entry.Version = 2
			continue
		}
		id := fmt.Sprintf("migrated-id-%d", i)
		entry.ID = id
		entry.ParentID = prevID
		
		// Capture the pointer of the new ID for the next loop iteration
		savedID := id
		prevID = &savedID
	}
}

func loadLargeSessionEntries(t *testing.T) []FileEntry {
	content, err := os.ReadFile("testdata/large-session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")
	var entries []FileEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry FileEntry
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			if entry.Type != TypeSession {
				entries = append(entries, entry)
			}
		}
	}
	migrateV1ToV2(entries)
	return entries
}

func createMockUsage(input, output int, cacheRead, cacheWrite int) opencode.Usage {
	return opencode.Usage{
		Input:       input,
		Output:      output,
		CacheRead:   cacheRead,
		CacheWrite:  cacheWrite,
		TotalTokens: input + output + cacheRead + cacheWrite,
	}
}

func createUserMessage(text string) opencode.Message {
	return opencode.Message{Role: "user", Content: opencode.StringContent(text)}
}

func createAssistantMessage(text string, usage opencode.Usage) opencode.Message {
	return opencode.Message{
		Role:    "assistant",
		Content: opencode.StringContent(text),
		Usage:   &usage,
	}
}

type entryHelper struct {
	counter int
	lastID  *string
}

func (h *entryHelper) nextID() string {
	id := fmt.Sprintf("test-id-%d", h.counter)
	h.counter++
	h.lastID = &id
	return id
}

func (h *entryHelper) createMessageEntry(msg opencode.Message) FileEntry {
	parent := h.lastID
	id := h.nextID()
	return FileEntry{
		SessionEntry: SessionEntry{
			Type:      TypeMessage,
			ID:        id,
			ParentID:  parent,
			Timestamp: time.Now().Format(time.RFC3339),
			Message:   &msg,
		},
	}
}

func (h *entryHelper) createCompactionEntry(summary string, firstKeptEntryID string) FileEntry {
	parent := h.lastID
	id := h.nextID()
	return FileEntry{
		SessionEntry: SessionEntry{
			Type:             TypeCompaction,
			ID:               id,
			ParentID:         parent,
			Timestamp:        time.Now().Format(time.RFC3339),
			Summary:          summary,
			FirstKeptEntryID: firstKeptEntryID,
			TokensBefore:     10000,
		},
	}
}

func (h *entryHelper) createModelChangeEntry(provider string, modelID string) FileEntry {
	parent := h.lastID
	id := h.nextID()
	return FileEntry{
		SessionEntry: SessionEntry{
			Type:      TypeModelChange,
			ID:        id,
			ParentID:  parent,
			Timestamp: time.Now().Format(time.RFC3339),
			Provider:  provider,
			ModelID:   modelID,
		},
	}
}

func (h *entryHelper) createThinkingLevelEntry(level string) FileEntry {
	parent := h.lastID
	id := h.nextID()
	return FileEntry{
		SessionEntry: SessionEntry{
			Type:          TypeThinkingLevelChange,
			ID:            id,
			ParentID:      parent,
			Timestamp:     time.Now().Format(time.RFC3339),
			ThinkingLevel: level,
		},
	}
}

func TestTokenCalculation(t *testing.T) {
	usage := createMockUsage(1000, 500, 200, 100)
	if got := CalculateContextTokens(usage); got != 1800 {
		t.Fatalf("CalculateContextTokens = %d, want 1800", got)
	}

	zeroUsage := createMockUsage(0, 0, 0, 0)
	if got := CalculateContextTokens(zeroUsage); got != 0 {
		t.Fatalf("CalculateContextTokens = %d, want 0", got)
	}
}

func TestGetLastAssistantUsage(t *testing.T) {
	h := &entryHelper{}
	entries := []FileEntry{
		h.createMessageEntry(createUserMessage("Hello")),
		h.createMessageEntry(createAssistantMessage("Hi", createMockUsage(100, 50, 0, 0))),
		h.createMessageEntry(createUserMessage("How are you?")),
		h.createMessageEntry(createAssistantMessage("Good", createMockUsage(200, 100, 0, 0))),
	}

	usage := GetLastAssistantUsage(entries)
	if usage == nil {
		t.Fatal("expected last usage to be non-nil")
	}
	if usage.Input != 200 {
		t.Fatalf("usage.Input = %d, want 200", usage.Input)
	}
}

func TestShouldCompact(t *testing.T) {
	settings := config.CompactionSettings{
		Enabled:          true,
		ReserveTokens:    10000,
		KeepRecentTokens: 20000,
	}

	if !ShouldCompact(95000, 100000, settings) {
		t.Fatal("should compact at 95000 tokens")
	}
	if ShouldCompact(89000, 100000, settings) {
		t.Fatal("should not compact at 89000 tokens")
	}

	disabledSettings := config.CompactionSettings{
		Enabled:          false,
		ReserveTokens:    10000,
		KeepRecentTokens: 20000,
	}
	if ShouldCompact(95000, 100000, disabledSettings) {
		t.Fatal("should not compact when disabled")
	}
}

func TestFindCutPoint(t *testing.T) {
	h := &entryHelper{}
	var entries []FileEntry
	for i := 0; i < 10; i++ {
		entries = append(entries, h.createMessageEntry(createUserMessage(fmt.Sprintf("User %d", i))))
		entries = append(entries, h.createMessageEntry(createAssistantMessage(fmt.Sprintf("Assistant %d", i), createMockUsage(0, 100, (i+1)*1000, 0))))
	}

	result := FindCutPoint(entries, 0, len(entries), 2500)
	cutEntry := entries[result.FirstKeptEntryIndex]
	if cutEntry.Type != TypeMessage {
		t.Fatalf("expected cut point to be a message entry, got %v", cutEntry.Type)
	}
}

func TestBuildSessionContext(t *testing.T) {
	h := &entryHelper{}
	entries := []FileEntry{
		h.createMessageEntry(createUserMessage("1")),
		h.createMessageEntry(createAssistantMessage("a", createMockUsage(100, 50, 0, 0))),
		h.createMessageEntry(createUserMessage("2")),
		h.createMessageEntry(createAssistantMessage("b", createMockUsage(100, 50, 0, 0))),
	}

	loaded := BuildSessionContext(entries, nil)
	if len(loaded.Messages) != 4 {
		t.Fatalf("loaded.Messages len = %d, want 4", len(loaded.Messages))
	}

	// Test single compaction
	h2 := &entryHelper{}
	u1 := h2.createMessageEntry(createUserMessage("1"))
	a1 := h2.createMessageEntry(createAssistantMessage("a", createMockUsage(100, 50, 0, 0)))
	u2 := h2.createMessageEntry(createUserMessage("2"))
	a2 := h2.createMessageEntry(createAssistantMessage("b", createMockUsage(100, 50, 0, 0)))
	compaction := h2.createCompactionEntry("Summary of 1,a,2,b", u2.ID)
	u3 := h2.createMessageEntry(createUserMessage("3"))
	a3 := h2.createMessageEntry(createAssistantMessage("c", createMockUsage(100, 50, 0, 0)))

	entries2 := []FileEntry{u1, a1, u2, a2, compaction, u3, a3}
	loaded2 := BuildSessionContext(entries2, nil)

	// summary + kept (u2, a2) + after (u3, a3) = 5
	if len(loaded2.Messages) != 5 {
		t.Fatalf("loaded2.Messages len = %d, want 5", len(loaded2.Messages))
	}
	if loaded2.Messages[0].Role != "compactionSummary" {
		t.Fatalf("expected first message to be compactionSummary, got %v", loaded2.Messages[0].Role)
	}
}

func TestLargeSessionFixture(t *testing.T) {
	entries := loadLargeSessionEntries(t)
	if len(entries) <= 100 {
		t.Fatalf("expected > 100 entries, got %d", len(entries))
	}

	result := FindCutPoint(entries, 0, len(entries), 20000)
	cutEntry := entries[result.FirstKeptEntryIndex]
	t.Logf("result: FirstKeptEntryIndex=%d, IsSplitTurn=%v, TurnStartIndex=%d, cutEntry.Type=%v",
		result.FirstKeptEntryIndex, result.IsSplitTurn, result.TurnStartIndex, cutEntry.Type)
	if cutEntry.Type != TypeMessage {
		t.Fatalf("expected cut point to be message, got %v", cutEntry.Type)
	}

	loaded := BuildSessionContext(entries, nil)
	if len(loaded.Messages) <= 100 {
		t.Fatalf("expected > 100 messages in loaded context, got %d", len(loaded.Messages))
	}
}

func TestSerializeConversation(t *testing.T) {
	longContent := strings.Repeat("x", 5000)
	messages := []opencode.Message{
		{
			Role:       "tool",
			ToolCallID: "tc1",
			Content:    opencode.StringContent(longContent),
		},
	}

	result := SerializeConversation(messages)
	if !strings.Contains(result, "[Tool result]:") {
		t.Fatal("expected Tool result header")
	}
	if !strings.Contains(result, "more characters truncated") {
		t.Fatal("expected truncated indicator")
	}
	// First 2000 characters should be kept
	if !strings.Contains(result, strings.Repeat("x", 2000)) {
		t.Fatal("expected first 2000 characters to be kept")
	}

	shortContent := strings.Repeat("x", 1500)
	messagesShort := []opencode.Message{
		{
			Role:       "tool",
			ToolCallID: "tc1",
			Content:    opencode.StringContent(shortContent),
		},
	}
	resultShort := SerializeConversation(messagesShort)
	if resultShort != fmt.Sprintf("[Tool result]: %s", shortContent) {
		t.Fatalf("unexpected short content output: %q", resultShort)
	}
}

type mockHook struct {
	cancel        bool
	customCompact *CompactionResult
	calledBefore  bool
	calledOn      bool
}

func (h *mockHook) BeforeCompact(evt BeforeCompactEvent) (*BeforeCompactResult, error) {
	h.calledBefore = true
	return &BeforeCompactResult{
		Cancel:     h.cancel,
		Compaction: h.customCompact,
	}, nil
}

func (h *mockHook) OnCompact(evt CompactEvent) error {
	h.calledOn = true
	return nil
}

func (h *mockHook) BeforeTree(evt BeforeTreeEvent) (*BeforeTreeResult, error) {
	return nil, nil
}

func (h *mockHook) OnTree(evt TreeEvent) error {
	return nil
}

func TestExtensionHooks(t *testing.T) {
	// Register extension hook
	hook := &mockHook{cancel: true}
	ExtensionHooks = append(ExtensionHooks, hook)
	defer func() {
		ExtensionHooks = nil
	}()

	// Create compaction details to prep compaction
	h := &entryHelper{}
	u1 := h.createMessageEntry(createUserMessage(strings.Repeat("word ", 500)))
	a1 := h.createMessageEntry(createAssistantMessage(strings.Repeat("reply ", 500), createMockUsage(100, 50, 0, 0)))
	u2 := h.createMessageEntry(createUserMessage("2"))
	a2 := h.createMessageEntry(createAssistantMessage("b", createMockUsage(100, 50, 0, 0)))
	path := []FileEntry{u1, a1, u2, a2}

	settings := config.CompactionSettings{
		Enabled:          true,
		ReserveTokens:    100,
		KeepRecentTokens: 50,
	}

	prep := PrepareCompaction(path, settings)
	if prep == nil {
		t.Fatal("expected prep to be non-nil")
	}

	// Trigger hook cancel test
	evt := BeforeCompactEvent{
		Preparation: prep,
		BranchEntries: path,
	}
	
	res, err := hook.BeforeCompact(evt)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Cancel {
		t.Fatal("expected cancel to be true")
	}
}

