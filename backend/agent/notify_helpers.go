package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/enough/enough/backend/approval"
)

// notifyStagedWrite surfaces write-approval staging to the TUI (Hermes-style
// persistent system line) in addition to the inline tool result block.
func (a *Agent) notifyStagedWrite(toolOutput string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	notify := a.notify
	prompt := a.approvalPrompt
	a.mu.Unlock()

	var data struct {
		Staged    bool   `json:"staged"`
		PendingID string `json:"pending_id"`
		Gist      string `json:"gist"`
		Message   string `json:"message"`
		Target    string `json:"target"`
	}
	if err := json.Unmarshal([]byte(toolOutput), &data); err != nil || !data.Staged {
		return
	}

	subsystem := approval.SubsystemSkills
	where := "/skills"
	if strings.Contains(strings.ToLower(data.Message), "memory.write_approval") ||
		data.Target == "memory" || data.Target == "user" {
		subsystem = approval.SubsystemMemory
		where = "/memory"
	}
	gist := strings.TrimSpace(data.Gist)
	if gist == "" {
		gist = strings.TrimSpace(data.Message)
	}
	if notify != nil {
		if data.PendingID != "" {
			notify(fmt.Sprintf("⏳ Staged for approval: %s — use %s approve %s",
				gist, where, data.PendingID))
		} else {
			notify(fmt.Sprintf("⏳ Staged for approval — check %s pending", where))
		}
	}
	if prompt != nil && data.PendingID != "" {
		prompt(subsystem, data.PendingID)
	}
}

// notifyDirectMemoryWrite surfaces successful immediate memory writes to the TUI.
// Staged writes are handled by notifyStagedWrite; this covers the common case
// where memory.write_approval is off and the tool applies directly.
func (a *Agent) notifyDirectMemoryWrite(argsJSON, toolOutput string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	notify := a.notify
	a.mu.Unlock()
	if notify == nil {
		return
	}

	var result struct {
		Success bool   `json:"success"`
		Staged  bool   `json:"staged"`
		Target  string `json:"target"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(toolOutput), &result); err != nil || !result.Success || result.Staged {
		return
	}

	var args struct {
		Action      string `json:"action"`
		Target      string `json:"target"`
		Content     string `json:"content"`
		Match       string `json:"match"`
		Replacement string `json:"replacement"`
	}
	_ = json.Unmarshal([]byte(argsJSON), &args)

	target := strings.TrimSpace(args.Target)
	if target == "" {
		target = strings.TrimSpace(result.Target)
	}
	if target == "" {
		target = "memory"
	}
	label := "MEMORY.md"
	if target == "user" {
		label = "USER.md"
	}

	var detail string
	switch args.Action {
	case "add":
		detail = strings.TrimSpace(args.Content)
	case "replace":
		detail = strings.TrimSpace(args.Replacement)
		if args.Match != "" {
			detail = fmt.Sprintf("%q → %s", args.Match, detail)
		}
	case "remove":
		detail = strings.TrimSpace(args.Match)
		if detail != "" {
			detail = "remove " + detail
		}
	default:
		detail = strings.TrimSpace(result.Message)
	}
	if detail == "" {
		detail = strings.TrimSpace(result.Message)
	}
	if detail == "" {
		detail = args.Action
	}
	if len(detail) > 120 {
		detail = detail[:117] + "..."
	}

	notify(fmt.Sprintf("💾 Saved to %s: %s", label, detail))
}
