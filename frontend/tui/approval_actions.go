package tui

import (
	"fmt"

	"github.com/enough/enough/backend/approval"
	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/memory"
	"github.com/enough/enough/backend/skills"
)

func (a *App) pendingWriteDiff(subsystem, id string) (string, error) {
	r, err := approval.GetPending(subsystem, id)
	if err != nil || r == nil {
		return "", fmt.Errorf("staged update '%s' not found", id)
	}
	if subsystem == approval.SubsystemSkills {
		return approval.SkillPendingDiff(*r), nil
	}
	return approval.MemoryPendingDiff(*r), nil
}

func (a *App) approvePendingWrite(subsystem, id string) (string, error) {
	r, err := approval.GetPending(subsystem, id)
	if err != nil || r == nil {
		return "", fmt.Errorf("staged update '%s' not found", id)
	}

	switch subsystem {
	case approval.SubsystemSkills:
		opts := skills.SkillManageOptions{
			GuardEnabled:       true,
			MarkCreatedAsAgent: r.Origin == "agent",
		}
		res, err := skills.ApplySkillPending(r.Payload, opts)
		if err != nil {
			return "", err
		}
		if !res.Success {
			if res.Error != "" {
				return "", fmt.Errorf("%s", res.Error)
			}
			return "", fmt.Errorf("failed to apply staged update")
		}
		_, _ = approval.DiscardPending(subsystem, id)
		msg := res.Message
		if msg == "" {
			msg = r.Summary
		}
		return fmt.Sprintf("Staged update %s approved and applied: %s", id, msg), nil

	case approval.SubsystemMemory:
		cfg, err := config.LoadRuntime()
		if err != nil {
			return "", err
		}
		ag := a.ensureAgent(cfg)
		store := ag.MemoryStore()
		if store == nil {
			return "", fmt.Errorf("memory store is not initialized or disabled")
		}
		res := memory.ApplyMemoryPending(r.Payload, store)
		if !res.Success {
			if res.Error != "" {
				return "", fmt.Errorf("%s", res.Error)
			}
			return "", fmt.Errorf("failed to apply memory update")
		}
		_, _ = approval.DiscardPending(subsystem, id)
		summary := r.Summary
		if summary == "" {
			summary = "memory update applied"
		}
		return fmt.Sprintf("Staged update %s approved and applied: %s", id, summary), nil

	default:
		return "", fmt.Errorf("unknown subsystem %q", subsystem)
	}
}

func (a *App) rejectPendingWrite(subsystem, id string) (string, error) {
	ok, err := approval.DiscardPending(subsystem, id)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("staged update '%s' not found", id)
	}
	return fmt.Sprintf("Staged update %s rejected.", id), nil
}
