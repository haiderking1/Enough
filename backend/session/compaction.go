package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/opencode"
)

const SummarizationSystemPrompt = `You are a context summarization assistant. Your task is to read a conversation between a user and an AI coding assistant, then produce a structured summary following the exact format specified.

Do NOT continue the conversation. Do NOT respond to any questions in the conversation. ONLY output the structured summary.`

const SummarizationPrompt = `The messages above are a conversation to summarize. Create a structured context checkpoint summary that another LLM will use to continue the work.

Use this EXACT format:

## Goal
[What is the user trying to accomplish? Can be multiple items if the session covers different tasks.]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by user]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Current work]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, or references needed to continue]
- [Or "(none)" if not applicable]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

const UpdateSummarizationPrompt = `The messages above are NEW conversation messages to incorporate into the existing summary provided in <previous-summary> tags.

Update the existing structured summary with new information. RULES:
- PRESERVE all existing information from the previous summary
- ADD new progress, decisions, and context from the new messages
- UPDATE the Progress section: move items from "In Progress" to "Done" when completed
- UPDATE "Next Steps" based on what was accomplished
- PRESERVE exact file paths, function names, and error messages
- If something is no longer relevant, you may remove it

Use this EXACT format:

## Goal
[Preserve existing goals, add new ones if the task expanded]

## Constraints & Preferences
- [Preserve existing, add new ones discovered]

## Progress
### Done
- [x] [Include previously done items AND newly completed items]

### In Progress
- [ ] [Current work - update based on progress]

### Blocked
- [Current blockers - remove if resolved]

## Key Decisions
- **[Decision]**: [Brief rationale] (preserve all previous, add new)

## Next Steps
1. [Update based on current state]

## Critical Context
- [Preserve important context, add new if needed]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

const TurnPrefixSummarizationPrompt = `This is the PREFIX of a turn that was too large to keep. The SUFFIX (recent work) is retained.

Summarize the prefix to provide context for the retained suffix:

## Original Request
[What did the user ask for in this turn?]

## Early Progress
- [Key decisions and work done in the prefix]

## Context for Suffix
- [Information needed to understand the retained recent work]

Be concise. Focus on what's needed to understand the kept suffix.`

type CompactionPreparation struct {
	FirstKeptEntryID    string
	MessagesToSummarize []opencode.Message
	TurnPrefixMessages  []opencode.Message
	IsSplitTurn         bool
	TokensBefore        int
	PreviousSummary     string
	FileOps             FileOperations
	Settings            config.CompactionSettings
}

type CompactionResult struct {
	Summary          string             `json:"summary"`
	FirstKeptEntryID string             `json:"firstKeptEntryId"`
	TokensBefore     int                `json:"tokensBefore"`
	Details          *CompactionDetails `json:"details,omitempty"`
}
type BeforeCompactEvent struct {
	Preparation        *CompactionPreparation
	BranchEntries      []FileEntry
	CustomInstructions string
	Context            context.Context
}

type BeforeCompactResult struct {
	Cancel     bool
	Compaction *CompactionResult
}

type CompactEvent struct {
	CompactionEntry FileEntry
	FromExtension   bool
}

type TreePreparation struct {
	TargetID            string
	OldLeafID           *string
	CommonAncestorID    *string
	EntriesToSummarize  []FileEntry
	UserWantsSummary    bool
	CustomInstructions  string
	ReplaceInstructions bool
	Label               string
}

type BeforeTreeEvent struct {
	Preparation TreePreparation
	Context     context.Context
}

type BeforeTreeResult struct {
	Cancel              bool
	Summary             *BranchSummaryResult
	CustomInstructions  *string
	ReplaceInstructions *bool
	Label               *string
}

type TreeEvent struct {
	NewLeafID     string
	OldLeafID     *string
	SummaryEntry  *FileEntry
	FromExtension bool
}

type ExtensionHook interface {
	BeforeCompact(evt BeforeCompactEvent) (*BeforeCompactResult, error)
	OnCompact(evt CompactEvent) error
	BeforeTree(evt BeforeTreeEvent) (*BeforeTreeResult, error)
	OnTree(evt TreeEvent) error
}

var ExtensionHooks []ExtensionHook
func GenerateSummary(
	ctx context.Context,
	client *opencode.Client,
	currentMessages []opencode.Message,
	reserveTokens int,
	customInstructions string,
	previousSummary string,
) (string, error) {
	basePrompt := SummarizationPrompt
	if previousSummary != "" {
		basePrompt = UpdateSummarizationPrompt
	}
	if customInstructions != "" {
		basePrompt = basePrompt + "\n\nAdditional focus: " + customInstructions
	}

	llmMessages := ConvertToLlm(currentMessages)
	conversationText := SerializeConversation(llmMessages)

	promptText := fmt.Sprintf("<conversation>\n%s\n</conversation>\n\n", conversationText)
	if previousSummary != "" {
		promptText += fmt.Sprintf("<previous-summary>\n%s\n</previous-summary>\n\n", previousSummary)
	}
	promptText += basePrompt

	req := opencode.ChatRequest{
		Messages: []opencode.Message{
			{Role: "system", Content: opencode.StringContent(SummarizationSystemPrompt)},
			{Role: "user", Content: opencode.StringContent(promptText)},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return opencode.ContentString(resp.Choices[0].Message), nil
}

func GenerateTurnPrefixSummary(
	ctx context.Context,
	client *opencode.Client,
	messages []opencode.Message,
	reserveTokens int,
) (string, error) {
	llmMessages := ConvertToLlm(messages)
	conversationText := SerializeConversation(llmMessages)
	promptText := fmt.Sprintf("<conversation>\n%s\n</conversation>\n\n%s", conversationText, TurnPrefixSummarizationPrompt)

	req := opencode.ChatRequest{
		Messages: []opencode.Message{
			{Role: "system", Content: opencode.StringContent(SummarizationSystemPrompt)},
			{Role: "user", Content: opencode.StringContent(promptText)},
		},
	}

	resp, err := client.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	return opencode.ContentString(resp.Choices[0].Message), nil
}

func PrepareCompaction(pathEntries []FileEntry, settings config.CompactionSettings) *CompactionPreparation {
	return prepareCompaction(pathEntries, settings)
}

// PrepareManualCompaction compacts as aggressively as possible, keeping only the
// most recent turn. Manual /compact should work even on short sessions.
func PrepareManualCompaction(pathEntries []FileEntry, settings config.CompactionSettings) *CompactionPreparation {
	manual := settings
	manual.KeepRecentTokens = 0
	return prepareCompaction(pathEntries, manual)
}

func prepareCompaction(pathEntries []FileEntry, settings config.CompactionSettings) *CompactionPreparation {
	if len(pathEntries) > 0 && pathEntries[len(pathEntries)-1].Type == TypeCompaction {
		return nil
	}

	prevCompactionIndex := -1
	for i := len(pathEntries) - 1; i >= 0; i-- {
		if pathEntries[i].Type == TypeCompaction {
			prevCompactionIndex = i
			break
		}
	}

	previousSummary := ""
	boundaryStart := 0
	if prevCompactionIndex >= 0 {
		prevCompaction := pathEntries[prevCompactionIndex]
		previousSummary = prevCompaction.Summary
		firstKeptIndex := -1
		for i, entry := range pathEntries {
			if entry.ID == prevCompaction.FirstKeptEntryID {
				firstKeptIndex = i
				break
			}
		}
		if firstKeptIndex >= 0 {
			boundaryStart = firstKeptIndex
		} else {
			boundaryStart = prevCompactionIndex + 1
		}
	}
	boundaryEnd := len(pathEntries)

	tokensBefore := EstimateContextTokens(BuildSessionContext(pathEntries, nil).Messages).Tokens

	cutPoint := FindCutPoint(pathEntries, boundaryStart, boundaryEnd, settings.KeepRecentTokens)

	if cutPoint.FirstKeptEntryIndex >= len(pathEntries) {
		return nil
	}
	firstKeptEntry := pathEntries[cutPoint.FirstKeptEntryIndex]
	if firstKeptEntry.ID == "" {
		return nil
	}
	firstKeptEntryID := firstKeptEntry.ID

	historyEnd := cutPoint.FirstKeptEntryIndex
	if cutPoint.IsSplitTurn {
		historyEnd = cutPoint.TurnStartIndex
	}

	// Messages to summarize (will be discarded after summary)
	var messagesToSummarize []opencode.Message
	for i := boundaryStart; i < historyEnd; i++ {
		entry := pathEntries[i]
		if entry.Type == TypeMessage && entry.Message != nil {
			messagesToSummarize = append(messagesToSummarize, *entry.Message)
		}
	}

	// Messages for turn prefix summary (if splitting a turn)
	var turnPrefixMessages []opencode.Message
	if cutPoint.IsSplitTurn {
		for i := cutPoint.TurnStartIndex; i < cutPoint.FirstKeptEntryIndex; i++ {
			entry := pathEntries[i]
			if entry.Type == TypeMessage && entry.Message != nil {
				turnPrefixMessages = append(turnPrefixMessages, *entry.Message)
			}
		}
	}

	// Extract file operations from messages and previous compaction
	fileOps := ExtractFileOperations(messagesToSummarize, pathEntries, prevCompactionIndex)

	// Also extract file ops from turn prefix if splitting
	if cutPoint.IsSplitTurn {
		for _, msg := range turnPrefixMessages {
			ExtractFileOpsFromMessage(msg, fileOps)
		}
	}

	return &CompactionPreparation{
		FirstKeptEntryID:    firstKeptEntryID,
		MessagesToSummarize: messagesToSummarize,
		TurnPrefixMessages:  turnPrefixMessages,
		IsSplitTurn:         cutPoint.IsSplitTurn,
		TokensBefore:        tokensBefore,
		PreviousSummary:     previousSummary,
		FileOps:             fileOps,
		Settings:            settings,
	}
}

func ExtractFileOperations(messages []opencode.Message, entries []FileEntry, prevCompactionIndex int) FileOperations {
	fileOps := NewFileOps()
	if prevCompactionIndex >= 0 {
		prevComp := entries[prevCompactionIndex]
		if !prevComp.FromHook && prevComp.Details != nil {
			if detailsMap, ok := prevComp.Details.(map[string]any); ok {
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
			} else if detailsComp, ok := prevComp.Details.(CompactionDetails); ok {
				for _, f := range detailsComp.ReadFiles {
					fileOps.Read[f] = true
				}
				for _, f := range detailsComp.ModifiedFiles {
					fileOps.Edited[f] = true
				}
			}
		}
	}
	for _, msg := range messages {
		ExtractFileOpsFromMessage(msg, fileOps)
	}
	return fileOps
}

func Compact(
	ctx context.Context,
	client *opencode.Client,
	prep *CompactionPreparation,
	customInstructions string,
) (*CompactionResult, error) {
	var summary string
	if prep.IsSplitTurn && len(prep.TurnPrefixMessages) > 0 {
		var historySummary string
		var prefixSummary string
		var err1 error
		var err2 error

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			if len(prep.MessagesToSummarize) > 0 {
				historySummary, err1 = GenerateSummary(ctx, client, prep.MessagesToSummarize, prep.Settings.ReserveTokens, customInstructions, prep.PreviousSummary)
			} else {
				historySummary = "No prior history."
			}
		}()

		go func() {
			defer wg.Done()
			prefixSummary, err2 = GenerateTurnPrefixSummary(ctx, client, prep.TurnPrefixMessages, prep.Settings.ReserveTokens)
		}()

		wg.Wait()

		if err1 != nil {
			return nil, err1
		}
		if err2 != nil {
			return nil, err2
		}

		summary = fmt.Sprintf("%s\n\n---\n\n**Turn Context (split turn):**\n\n%s", historySummary, prefixSummary)
	} else {
		if len(prep.MessagesToSummarize) == 0 {
			summary = "No prior history."
		} else {
			var err error
			summary, err = GenerateSummary(ctx, client, prep.MessagesToSummarize, prep.Settings.ReserveTokens, customInstructions, prep.PreviousSummary)
			if err != nil {
				return nil, err
			}
		}
	}

	readFiles, modifiedFiles := ComputeFileLists(prep.FileOps)
	summary += FormatFileOperations(readFiles, modifiedFiles)

	return &CompactionResult{
		Summary:          summary,
		FirstKeptEntryID: prep.FirstKeptEntryID,
		TokensBefore:     prep.TokensBefore,
		Details: &CompactionDetails{
			ReadFiles:     readFiles,
			ModifiedFiles: modifiedFiles,
		},
	}, nil
}
