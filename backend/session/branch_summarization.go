package session

import (
	"context"
	"fmt"

	"github.com/enough/enough/backend/opencode"
)

type BranchSummaryResult struct {
	Summary       string   `json:"summary"`
	ReadFiles     []string `json:"readFiles"`
	ModifiedFiles []string `json:"modifiedFiles"`
	Aborted       bool     `json:"aborted"`
	Error         string   `json:"error"`
}

type BranchSummaryDetails struct {
	ReadFiles     []string `json:"readFiles"`
	ModifiedFiles []string `json:"modifiedFiles"`
}

type BranchPreparation struct {
	Messages    []opencode.Message
	FileOps     FileOperations
	TotalTokens int
}

type CollectEntriesResult struct {
	Entries          []FileEntry
	CommonAncestorID *string
}

type GenerateBranchSummaryOptions struct {
	Client              *opencode.Client
	CustomInstructions  string
	ReplaceInstructions bool
	ReserveTokens       int
	ContextWindow       int
}

// CollectEntriesForBranchSummary collects entries to summarize when navigating from oldLeafID to targetID.
func CollectEntriesForBranchSummary(entries []FileEntry, oldLeafID *string, targetID string) CollectEntriesResult {
	if oldLeafID == nil {
		return CollectEntriesResult{Entries: nil, CommonAncestorID: nil}
	}

	oldBranch := GetBranch(entries, oldLeafID)
	oldPathSet := make(map[string]bool)
	for _, e := range oldBranch {
		oldPathSet[e.ID] = true
	}

	targetBranch := GetBranch(entries, &targetID)
	var commonAncestorID *string
	for i := len(targetBranch) - 1; i >= 0; i-- {
		if oldPathSet[targetBranch[i].ID] {
			id := targetBranch[i].ID
			commonAncestorID = &id
			break
		}
	}

	// Collect entries from old leaf back to common ancestor
	var collected []FileEntry
	byId := make(map[string]FileEntry)
	for _, e := range entries {
		if e.ID != "" {
			byId[e.ID] = e
		}
	}

	current := oldLeafID
	for current != nil {
		if commonAncestorID != nil && *current == *commonAncestorID {
			break
		}
		entry, ok := byId[*current]
		if !ok {
			break
		}
		collected = append(collected, entry)
		current = entry.ParentID
	}

	// Reverse collected to get chronological order
	for i, j := 0, len(collected)-1; i < j; i, j = i+1, j-1 {
		collected[i], collected[j] = collected[j], collected[i]
	}

	return CollectEntriesResult{
		Entries:          collected,
		CommonAncestorID: commonAncestorID,
	}
}

func getMessageFromEntry(entry FileEntry) *opencode.Message {
	switch entry.Type {
	case TypeMessage:
		if entry.Message == nil || entry.Message.Role == "tool" {
			return nil
		}
		return entry.Message
	case TypeCustomMessage:
		if entry.Content != nil {
			var contentStr string
			if s, ok := entry.Content.(string); ok {
				contentStr = s
			}
			return &opencode.Message{
				Role:    "user",
				Content: opencode.StringContent(contentStr),
			}
		}
		return nil
	case TypeBranchSummary:
		if entry.Summary != "" {
			return &opencode.Message{
				Role:       "branchSummary",
				Content:    opencode.StringContent(entry.Summary),
				ToolCallID: entry.FromID,
			}
		}
		return nil
	case TypeCompaction:
		if entry.Summary != "" {
			return &opencode.Message{
				Role:    "compactionSummary",
				Content: opencode.StringContent(entry.Summary),
			}
		}
		return nil
	}
	return nil
}

func extractFileOpsFromDetails(details any, fileOps FileOperations) {
	if details == nil {
		return
	}
	if detailsMap, ok := details.(map[string]any); ok {
		if rf, ok := detailsMap["readFiles"].([]any); ok {
			for _, f := range rf {
				if fs, ok := f.(string); ok {
					fileOps.Read[fs] = true
				}
			}
		}
		if mf, ok := detailsMap["modifiedFiles"].([]any); ok {
			for _, f := range mf {
				if fs, ok := f.(string); ok {
					fileOps.Edited[fs] = true
				}
			}
		}
	} else if detailsComp, ok := details.(*CompactionDetails); ok && detailsComp != nil {
		for _, f := range detailsComp.ReadFiles {
			fileOps.Read[f] = true
		}
		for _, f := range detailsComp.ModifiedFiles {
			fileOps.Edited[f] = true
		}
	} else if detailsComp, ok := details.(CompactionDetails); ok {
		for _, f := range detailsComp.ReadFiles {
			fileOps.Read[f] = true
		}
		for _, f := range detailsComp.ModifiedFiles {
			fileOps.Edited[f] = true
		}
	}
}

func PrepareBranchEntries(entries []FileEntry, tokenBudget int) BranchPreparation {
	messages := []opencode.Message{}
	fileOps := NewFileOps()
	totalTokens := 0

	for _, entry := range entries {
		if entry.Type == TypeBranchSummary && !entry.FromHook && entry.Details != nil {
			extractFileOpsFromDetails(entry.Details, fileOps)
		}
	}

	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		msg := getMessageFromEntry(entry)
		if msg == nil {
			continue
		}

		ExtractFileOpsFromMessage(*msg, fileOps)
		tokens := EstimateMessageTokens(*msg)

		if tokenBudget > 0 && totalTokens+tokens > tokenBudget {
			if entry.Type == TypeCompaction || entry.Type == TypeBranchSummary {
				if float64(totalTokens) < float64(tokenBudget)*0.9 {
					messages = append([]opencode.Message{*msg}, messages...)
					totalTokens += tokens
				}
			}
			break
		}

		messages = append([]opencode.Message{*msg}, messages...)
		totalTokens += tokens
	}

	return BranchPreparation{
		Messages:    messages,
		FileOps:     fileOps,
		TotalTokens: totalTokens,
	}
}

const BranchSummaryPreamble = "The user explored a different conversation branch before returning here.\nSummary of that exploration:\n\n"

const BranchSummaryPrompt = `Create a structured summary of this conversation branch for context when returning later.

Use this EXACT format:

## Goal
[What was the user trying to accomplish in this branch?]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Work that was started but not finished]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [What should happen next to continue this work]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

func GenerateBranchSummary(
	ctx context.Context,
	entries []FileEntry,
	options GenerateBranchSummaryOptions,
) (BranchSummaryResult, error) {
	reserveTokens := options.ReserveTokens
	if reserveTokens <= 0 {
		reserveTokens = 16384
	}

	contextWindow := options.ContextWindow
	if contextWindow <= 0 {
		contextWindow = 128000
	}
	tokenBudget := contextWindow - reserveTokens

	prep := PrepareBranchEntries(entries, tokenBudget)
	if len(prep.Messages) == 0 {
		return BranchSummaryResult{Summary: "No content to summarize"}, nil
	}

	llmMessages := ConvertToLlm(prep.Messages)
	conversationText := SerializeConversation(llmMessages)

	var instructions string
	if options.ReplaceInstructions && options.CustomInstructions != "" {
		instructions = options.CustomInstructions
	} else if options.CustomInstructions != "" {
		instructions = fmt.Sprintf("%s\n\nAdditional focus: %s", BranchSummaryPrompt, options.CustomInstructions)
	} else {
		instructions = BranchSummaryPrompt
	}

	promptText := fmt.Sprintf("<conversation>\n%s\n</conversation>\n\n%s", conversationText, instructions)

	req := opencode.ChatRequest{
		Messages: []opencode.Message{
			{Role: "system", Content: opencode.StringContent(SummarizationSystemPrompt)},
			{Role: "user", Content: opencode.StringContent(promptText)},
		},
	}

	resp, err := options.Client.Chat(ctx, req)
	if err != nil {
		return BranchSummaryResult{Error: err.Error()}, err
	}

	summary := opencode.ContentString(resp.Choices[0].Message)
	summary = BranchSummaryPreamble + summary

	readFiles, modifiedFiles := ComputeFileLists(prep.FileOps)
	summary += FormatFileOperations(readFiles, modifiedFiles)

	return BranchSummaryResult{
		Summary:       summary,
		ReadFiles:     readFiles,
		ModifiedFiles: modifiedFiles,
	}, nil
}
