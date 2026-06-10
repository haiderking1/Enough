package session

import (
	"github.com/enough/enough/backend/opencode"
)

// SessionContext matches Flame's resolved context.
type SessionContext struct {
	Messages      []opencode.Message
	ThinkingLevel string
	Model         *ModelInfo
}

type ModelInfo struct {
	Provider string
	ModelID  string
}

// BuildSessionContext resolves the messages and settings on the branch of leafID.
// If leafID is nil, it uses the last entry.
func BuildSessionContext(entries []FileEntry, leafID *string) SessionContext {
	// Build map by ID
	byId := make(map[string]FileEntry)
	for _, entry := range entries {
		if entry.ID != "" {
			byId[entry.ID] = entry
		}
	}

	var leaf *FileEntry
	if leafID != nil {
		if val, ok := byId[*leafID]; ok {
			leaf = &val
		}
	} else if len(entries) > 0 {
		// Default to the last entry
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].Type != TypeSession {
				leaf = &entries[i]
				break
			}
		}
	}

	if leaf == nil {
		return SessionContext{Messages: nil, ThinkingLevel: "off", Model: nil}
	}

	// Walk from leaf to root to collect active branch path
	var path []FileEntry
	current := leaf
	for current != nil {
		path = append([]FileEntry{*current}, path...)
		if current.ParentID != nil {
			if val, ok := byId[*current.ParentID]; ok {
				current = &val
			} else {
				current = nil
			}
		} else {
			current = nil
		}
	}

	// Resolve active settings
	thinkingLevel := "off"
	var model *ModelInfo
	var compaction *FileEntry

	for _, entry := range path {
		switch entry.Type {
		case TypeThinkingLevelChange:
			thinkingLevel = entry.ThinkingLevel
		case TypeModelChange:
			model = &ModelInfo{Provider: entry.Provider, ModelID: entry.ModelID}
		case TypeMessage:
			if entry.Message != nil && entry.Message.Role == "assistant" {
				// We can derive model/provider from assistant if available in Message
			}
		case TypeCompaction:
			compaction = &entry
		}
	}

	var messages []opencode.Message

	appendMessage := func(entry FileEntry) {
		switch entry.Type {
		case TypeMessage:
			if entry.Message != nil {
				messages = append(messages, *entry.Message)
			}
		case TypeCustomMessage:
			if entry.Content != nil {
				var contentStr string
				if s, ok := entry.Content.(string); ok {
					contentStr = s
				}
				messages = append(messages, opencode.Message{
					Role:    "user",
					Content: opencode.StringContent(contentStr),
				})
			}
		case TypeBranchSummary:
			if entry.Summary != "" {
				messages = append(messages, opencode.Message{
					Role:       "branchSummary",
					Content:    opencode.StringContent(entry.Summary),
					ToolCallID: entry.FromID, // Reuse ToolCallID or other fields if needed, but Content is key
				})
			}
		}
	}

	if compaction != nil {
		// Emit summary first
		messages = append(messages, opencode.Message{
			Role:    "compactionSummary",
			Content: opencode.StringContent(compaction.Summary),
		})

		// Find compaction index in path
		compactionIdx := -1
		for i, entry := range path {
			if entry.Type == TypeCompaction && entry.ID == compaction.ID {
				compactionIdx = i
				break
			}
		}

		// Emit kept messages (before compaction, starting from firstKeptEntryId)
		foundFirstKept := false
		for i := 0; i < compactionIdx; i++ {
			entry := path[i]
			if entry.ID == compaction.FirstKeptEntryID {
				foundFirstKept = true
			}
			if foundFirstKept {
				appendMessage(entry)
			}
		}

		// Emit messages after compaction
		for i := compactionIdx + 1; i < len(path); i++ {
			entry := path[i]
			appendMessage(entry)
		}
	} else {
		// No compaction - emit all messages
		for _, entry := range path {
			appendMessage(entry)
		}
	}

	return SessionContext{
		Messages:      messages,
		ThinkingLevel: thinkingLevel,
		Model:         model,
	}
}

// GetBranch returns all entries from root to leafID in path order.
func GetBranch(entries []FileEntry, leafID *string) []FileEntry {
	byId := make(map[string]FileEntry)
	for _, entry := range entries {
		if entry.ID != "" {
			byId[entry.ID] = entry
		}
	}

	var leaf *FileEntry
	if leafID != nil {
		if val, ok := byId[*leafID]; ok {
			leaf = &val
		}
	} else if len(entries) > 0 {
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].Type != TypeSession {
				leaf = &entries[i]
				break
			}
		}
	}

	if leaf == nil {
		return nil
	}

	var path []FileEntry
	current := leaf
	for current != nil {
		path = append([]FileEntry{*current}, path...)
		if current.ParentID != nil {
			if val, ok := byId[*current.ParentID]; ok {
				current = &val
			} else {
				current = nil
			}
		} else {
			current = nil
		}
	}

	return path
}
