package agent

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/enough/enough/backend/core"
	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/session"
)

var overflowPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)prompt is too long`),
	regexp.MustCompile(`(?i)request_too_large`),
	regexp.MustCompile(`(?i)input is too long for requested model`),
	regexp.MustCompile(`(?i)exceeds the context window`),
	regexp.MustCompile(`(?i)exceeds (?:the )?(?:model'?s )?maximum context length`),
	regexp.MustCompile(`(?i)input token count.*exceeds the maximum`),
	regexp.MustCompile(`(?i)maximum prompt length is \d+`),
	regexp.MustCompile(`(?i)reduce the length of the messages`),
	regexp.MustCompile(`(?i)maximum context length is \d+ tokens`),
	regexp.MustCompile(`(?i)input \(\d+ tokens\) is longer than the model'?s context length`),
	regexp.MustCompile(`(?i)exceeds the limit of \d+`),
	regexp.MustCompile(`(?i)exceeds the available context size`),
	regexp.MustCompile(`(?i)greater than the context length`),
	regexp.MustCompile(`(?i)context window exceeds limit`),
	regexp.MustCompile(`(?i)exceeded model token limit`),
	regexp.MustCompile(`(?i)too large for model with \d+ maximum context length`),
	regexp.MustCompile(`(?i)model_context_window_exceeded`),
	regexp.MustCompile(`(?i)prompt too long; exceeded (?:max )?context length`),
	regexp.MustCompile(`(?i)context[_ ]length[_ ]exceeded`),
	regexp.MustCompile(`(?i)too many tokens`),
	regexp.MustCompile(`(?i)token limit exceeded`),
}

func IsContextOverflowError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, p := range overflowPatterns {
		if p.MatchString(msg) {
			return true
		}
	}
	return false
}

func GetContextWindow(model string) int {
	return ModelContextWindow(model, 0)
}

func (a *Agent) ReloadMessagesFromSession() {
	if a.session == nil {
		return
	}
	sessionMsgs := a.session.BuildSessionContext().Messages
	a.messages = append([]opencode.Message{{Role: "system", Content: opencode.StringContent(systemPrompt)}}, sessionMsgs...)
}

func (a *Agent) emitCompactionEnd(reason string, result *session.CompactionResult, aborted, willRetry bool, errMsg string) {
	if a.emit != nil {
		a.emit(core.Event{
			Kind: core.EventCompactionEnd,
			Data: core.CompactionEndEvent{
				Reason:       reason,
				Result:       result,
				Aborted:      aborted,
				WillRetry:    willRetry,
				ErrorMessage: errMsg,
			},
		})
	}
}

// Compact manually runs compaction on the agent's session.
func (a *Agent) Compact(ctx context.Context, customInstructions string) (*session.CompactionResult, error) {
	a.Abort()

	if a.emit != nil {
		a.emit(core.Event{
			Kind: core.EventCompactionStart,
			Data: core.CompactionStartEvent{Reason: "manual"},
		})
	}

	compactionCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.compactionCancel = cancel
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.compactionCancel = nil
		a.mu.Unlock()
	}()

	if a.session == nil {
		err := fmt.Errorf("no session manager available")
		a.emitCompactionEnd("manual", nil, false, false, err.Error())
		return nil, err
	}

	pathEntries := a.session.GetBranch(a.session.LeafID())

	messageCount := 0
	for _, entry := range pathEntries {
		if entry.Type == session.TypeMessage {
			messageCount++
		}
	}
	if messageCount < 2 {
		err := fmt.Errorf("Nothing to compact (no messages yet)")
		a.emitCompactionEnd("manual", nil, false, false, err.Error())
		return nil, err
	}

	settings := a.cfg.Compaction

	prep := session.PrepareManualCompaction(pathEntries, settings)
	if prep == nil {
		// Check if already compacted
		if len(pathEntries) > 0 && pathEntries[len(pathEntries)-1].Type == session.TypeCompaction {
			err := fmt.Errorf("Already compacted")
			a.emitCompactionEnd("manual", nil, false, false, err.Error())
			return nil, err
		}
		err := fmt.Errorf("Nothing to compact (session too small)")
		a.emitCompactionEnd("manual", nil, false, false, err.Error())
		return nil, err
	}

	// Support compaction hooks
	var extCompaction *session.CompactionResult
	fromExt := false
	for _, hook := range session.ExtensionHooks {
		evt := session.BeforeCompactEvent{
			Preparation:        prep,
			BranchEntries:      pathEntries,
			CustomInstructions: customInstructions,
			Context:            compactionCtx,
		}
		res, err := hook.BeforeCompact(evt)
		if err == nil && res != nil {
			if res.Cancel {
				err := fmt.Errorf("Compaction cancelled")
				a.emitCompactionEnd("manual", nil, true, false, err.Error())
				return nil, err
			}
			if res.Compaction != nil {
				extCompaction = res.Compaction
				fromExt = true
				break
			}
		}
	}

	var summary string
	var firstKeptEntryID string
	var tokensBefore int
	var details any

	if extCompaction != nil {
		summary = extCompaction.Summary
		firstKeptEntryID = extCompaction.FirstKeptEntryID
		tokensBefore = extCompaction.TokensBefore
		details = extCompaction.Details
	} else {
		res, err := session.Compact(compactionCtx, a.client, prep, customInstructions)
		if err != nil {
			aborted := errors.Is(err, context.Canceled)
			errMsg := ""
			if !aborted {
				errMsg = fmt.Sprintf("Compaction failed: %v", err)
			}
			a.emitCompactionEnd("manual", nil, aborted, false, errMsg)
			return nil, err
		}
		summary = res.Summary
		firstKeptEntryID = res.FirstKeptEntryID
		tokensBefore = res.TokensBefore
		details = res.Details
	}

	if compactionCtx.Err() != nil {
		err := fmt.Errorf("Compaction cancelled")
		a.emitCompactionEnd("manual", nil, true, false, err.Error())
		return nil, err
	}

	err := a.session.AppendCompaction(summary, firstKeptEntryID, tokensBefore, details, fromExt)
	if err != nil {
		a.emitCompactionEnd("manual", nil, false, false, err.Error())
		return nil, err
	}

	a.mu.Lock()
	a.ReloadMessagesFromSession()
	a.mu.Unlock()

	// Call OnCompact hooks
	newEntries := a.session.ParsedEntries()
	var savedEntry *session.FileEntry
	for i := len(newEntries) - 1; i >= 0; i-- {
		if newEntries[i].Type == session.TypeCompaction && newEntries[i].Summary == summary {
			savedEntry = &newEntries[i]
			break
		}
	}
	if savedEntry != nil {
		for _, hook := range session.ExtensionHooks {
			_ = hook.OnCompact(session.CompactEvent{
				CompactionEntry: *savedEntry,
				FromExtension:   fromExt,
			})
		}
	}

	compactionResult := &session.CompactionResult{
		Summary:          summary,
		FirstKeptEntryID: firstKeptEntryID,
		TokensBefore:     tokensBefore,
	}
	a.emitCompactionEnd("manual", compactionResult, false, false, "")
	return compactionResult, nil
}

func (a *Agent) RunAutoCompaction(ctx context.Context, reason string, willRetry bool) (bool, error) {
	if a.emit != nil {
		a.emit(core.Event{
			Kind: core.EventCompactionStart,
			Data: core.CompactionStartEvent{Reason: reason},
		})
	}

	compactionCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.compactionCancel = cancel
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.compactionCancel = nil
		a.mu.Unlock()
	}()

	pathEntries := a.session.GetBranch(a.session.LeafID())
	settings := a.cfg.Compaction

	prep := session.PrepareCompaction(pathEntries, settings)
	if prep == nil {
		a.emitCompactionEnd(reason, nil, false, false, "")
		return false, nil
	}

	// Support compaction hooks
	var extCompaction *session.CompactionResult
	fromExt := false
	for _, hook := range session.ExtensionHooks {
		evt := session.BeforeCompactEvent{
			Preparation:        prep,
			BranchEntries:      pathEntries,
			CustomInstructions: "",
			Context:            compactionCtx,
		}
		res, err := hook.BeforeCompact(evt)
		if err == nil && res != nil {
			if res.Cancel {
				a.emitCompactionEnd(reason, nil, true, false, "Compaction cancelled")
				return false, nil
			}
			if res.Compaction != nil {
				extCompaction = res.Compaction
				fromExt = true
				break
			}
		}
	}

	var summary string
	var firstKeptEntryID string
	var tokensBefore int
	var details any

	if extCompaction != nil {
		summary = extCompaction.Summary
		firstKeptEntryID = extCompaction.FirstKeptEntryID
		tokensBefore = extCompaction.TokensBefore
		details = extCompaction.Details
	} else {
		res, err := session.Compact(compactionCtx, a.client, prep, "")
		if err != nil {
			aborted := errors.Is(err, context.Canceled)
			errMsg := ""
			if !aborted {
				errMsg = fmt.Sprintf("Auto-compaction failed: %v", err)
			}
			a.emitCompactionEnd(reason, nil, aborted, false, errMsg)
			return false, err
		}
		summary = res.Summary
		firstKeptEntryID = res.FirstKeptEntryID
		tokensBefore = res.TokensBefore
		details = res.Details
	}

	if compactionCtx.Err() != nil {
		a.emitCompactionEnd(reason, nil, true, false, "Compaction cancelled")
		return false, nil
	}

	err := a.session.AppendCompaction(summary, firstKeptEntryID, tokensBefore, details, fromExt)
	if err != nil {
		a.emitCompactionEnd(reason, nil, false, false, err.Error())
		return false, err
	}

	a.mu.Lock()
	a.ReloadMessagesFromSession()
	a.mu.Unlock()

	// Call OnCompact hooks
	newEntries := a.session.ParsedEntries()
	var savedEntry *session.FileEntry
	for i := len(newEntries) - 1; i >= 0; i-- {
		if newEntries[i].Type == session.TypeCompaction && newEntries[i].Summary == summary {
			savedEntry = &newEntries[i]
			break
		}
	}
	if savedEntry != nil {
		for _, hook := range session.ExtensionHooks {
			_ = hook.OnCompact(session.CompactEvent{
				CompactionEntry: *savedEntry,
				FromExtension:   fromExt,
			})
		}
	}

	compactionResult := &session.CompactionResult{
		Summary:          summary,
		FirstKeptEntryID: firstKeptEntryID,
		TokensBefore:     tokensBefore,
	}
	a.emitCompactionEnd(reason, compactionResult, false, willRetry, "")
	return true, nil
}

type NavigateOptions struct {
	Summarize          bool
	CustomInstructions string
}

func (a *Agent) NavigateToEntry(ctx context.Context, targetID string, opts NavigateOptions) (bool, error) {
	a.AbortAndWait()

	if a.session == nil {
		return false, fmt.Errorf("no session manager available")
	}

	oldLeaf := a.session.LeafID()
	if oldLeaf != nil && *oldLeaf == targetID {
		return false, nil // no-op if already at target
	}

	// Resolve target entry
	var targetEntry *session.FileEntry
	for _, entry := range a.session.ParsedEntries() {
		if entry.ID == targetID {
			targetEntry = &entry
			break
		}
	}
	if targetEntry == nil {
		return false, fmt.Errorf("entry %s not found", targetID)
	}

	// Collect entries for summary
	prepResult := session.CollectEntriesForBranchSummary(a.session.ParsedEntries(), oldLeaf, targetID)

	customInstructions := opts.CustomInstructions
	replaceInstructions := false // default behavior

	// Setup custom instructions/event signal
	// Extension hook session_before_tree
	var extSummary *session.BranchSummaryResult
	fromExt := false

	for _, hook := range session.ExtensionHooks {
		evt := session.BeforeTreeEvent{
			Preparation: session.TreePreparation{
				TargetID:           targetID,
				OldLeafID:          oldLeaf,
				CommonAncestorID:   prepResult.CommonAncestorID,
				EntriesToSummarize: prepResult.Entries,
				UserWantsSummary:   opts.Summarize,
				CustomInstructions: customInstructions,
			},
			Context: ctx,
		}
		res, err := hook.BeforeTree(evt)
		if err == nil && res != nil {
			if res.Cancel {
				return false, fmt.Errorf("navigation cancelled by extension")
			}
			if res.Summary != nil {
				extSummary = res.Summary
				fromExt = true
			}
			if res.CustomInstructions != nil {
				customInstructions = *res.CustomInstructions
			}
			if res.ReplaceInstructions != nil {
				replaceInstructions = *res.ReplaceInstructions
			}
		}
	}

	// Setup start event
	if a.emit != nil {
		a.emit(core.Event{
			Kind: core.EventBranchSummaryStart,
			Data: core.BranchSummaryStartEvent{TargetID: targetID},
		})
	}

	var summaryText string
	var summaryDetails any

	if opts.Summarize && len(prepResult.Entries) > 0 && extSummary == nil {
		// Generate branch summary using LLM
		contextWindow := ModelContextWindow(a.cfg.Model, a.cfg.Compaction.ContextWindow)
		reserveTokens := a.cfg.Compaction.ReserveTokens
		if reserveTokens <= 0 {
			reserveTokens = 16384
		}

		genOpts := session.GenerateBranchSummaryOptions{
			Client:              a.client,
			CustomInstructions:  customInstructions,
			ReplaceInstructions: replaceInstructions,
			ReserveTokens:       reserveTokens,
			ContextWindow:       contextWindow,
		}

		res, err := session.GenerateBranchSummary(ctx, a.session.ParsedEntries(), genOpts)
		if err != nil {
			if a.emit != nil {
				a.emit(core.Event{
					Kind: core.EventBranchSummaryEnd,
					Data: core.BranchSummaryEndEvent{
						TargetID:     targetID,
						Aborted:      errors.Is(err, context.Canceled),
						ErrorMessage: err.Error(),
					},
				})
			}
			return false, err
		}

		if res.Aborted {
			if a.emit != nil {
				a.emit(core.Event{
					Kind: core.EventBranchSummaryEnd,
					Data: core.BranchSummaryEndEvent{
						TargetID: targetID,
						Aborted:  true,
					},
				})
			}
			return false, fmt.Errorf("branch summarization aborted")
		}

		summaryText = res.Summary
		summaryDetails = session.BranchSummaryDetails{
			ReadFiles:     res.ReadFiles,
			ModifiedFiles: res.ModifiedFiles,
		}
	} else if extSummary != nil {
		summaryText = extSummary.Summary
		summaryDetails = session.BranchSummaryDetails{
			ReadFiles:     extSummary.ReadFiles,
			ModifiedFiles: extSummary.ModifiedFiles,
		}
	}

	// Determine new leaf position based on target type
	var newLeafID string
	if targetEntry.Type == session.TypeMessage && targetEntry.Message != nil && targetEntry.Message.Role == "user" {
		if targetEntry.ParentID != nil {
			newLeafID = *targetEntry.ParentID
		}
	} else if targetEntry.Type == session.TypeCustomMessage {
		if targetEntry.ParentID != nil {
			newLeafID = *targetEntry.ParentID
		}
	} else {
		newLeafID = targetID
	}

	var summaryEntry *session.FileEntry
	if summaryText != "" {
		// Branch with summary
		var parentPtr *string
		if newLeafID != "" {
			parentPtr = &newLeafID
		}
		summaryID, branchErr := a.session.BranchWithSummary(parentPtr, summaryText, summaryDetails, fromExt)
		if branchErr != nil {
			return false, branchErr
		}

		// Find the appended branch summary entry
		for _, e := range a.session.ParsedEntries() {
			if e.ID == summaryID {
				summaryEntry = &e
				break
			}
		}
	} else if newLeafID == "" {
		a.session.ResetLeaf()
	} else {
		a.session.Branch(newLeafID)
	}

	// Re-load messages from context
	a.mu.Lock()
	a.ReloadMessagesFromSession()
	a.mu.Unlock()

	// Call OnTree hook
	if a.emit != nil {
		var resultSummary *session.BranchSummaryResult
		if summaryText != "" {
			var rf []string
			var mf []string
			if d, ok := summaryDetails.(session.BranchSummaryDetails); ok {
				rf = d.ReadFiles
				mf = d.ModifiedFiles
			} else if d, ok := summaryDetails.(*session.BranchSummaryDetails); ok && d != nil {
				rf = d.ReadFiles
				mf = d.ModifiedFiles
			}
			resultSummary = &session.BranchSummaryResult{
				Summary:       summaryText,
				ReadFiles:     rf,
				ModifiedFiles: mf,
			}
		}

		a.emit(core.Event{
			Kind: core.EventBranchSummaryEnd,
			Data: core.BranchSummaryEndEvent{
				TargetID: targetID,
				Result:   resultSummary,
				Aborted:  false,
			},
		})
	}

	// Call OnTree extension hooks
	var newLeafStr string
	if a.session.LeafID() != nil {
		newLeafStr = *a.session.LeafID()
	}
	for _, hook := range session.ExtensionHooks {
		_ = hook.OnTree(session.TreeEvent{
			NewLeafID:     newLeafStr,
			OldLeafID:     oldLeaf,
			SummaryEntry:  summaryEntry,
			FromExtension: fromExt,
		})
	}

	return true, nil
}
