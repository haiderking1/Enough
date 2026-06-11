package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const VALID_SKILL = `---
name: test-skill
description: A test skill for manage tool
---
Do step one, then step two.
`

func TestToolManageCreate(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)
	ClearSkillsPromptCache()

	// 1. Happy path create
	createArgs := map[string]interface{}{
		"action":   "create",
		"name":     "test-skill",
		"content":  VALID_SKILL,
		"category": "dev",
	}
	argsJSON, _ := json.Marshal(createArgs)
	resStr, isErr := ExecuteSkillManage(string(argsJSON), SkillManageOptions{GuardEnabled: true})
	if isErr {
		t.Fatalf("create failed: %s", resStr)
	}

	var res struct {
		Success bool   `json:"success"`
		Path    string `json:"path"`
	}
	_ = json.Unmarshal([]byte(resStr), &res)
	if !res.Success {
		t.Fatal("expected success true")
	}

	skillMd := filepath.Join(tempHome, "skills", "dev", "test-skill", "SKILL.md")
	data, err := os.ReadFile(skillMd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Do step one") {
		t.Fatalf("expected skill content to contain 'Do step one', got %q", string(data))
	}

	// 2. Reject collision
	resStrDup, _ := ExecuteSkillManage(string(argsJSON), SkillManageOptions{GuardEnabled: true})
	if !strings.Contains(resStrDup, "already exists") {
		t.Fatalf("expected collision error, got: %s", resStrDup)
	}

	// 3. Reject invalid frontmatter
	badFmArgs := map[string]interface{}{
		"action":  "create",
		"name":    "bad-fm",
		"content": "no frontmatter here",
	}
	badFmJSON, _ := json.Marshal(badFmArgs)
	resBadFm, _ := ExecuteSkillManage(string(badFmJSON), SkillManageOptions{GuardEnabled: true})
	if !strings.Contains(resBadFm, "frontmatter") {
		t.Fatalf("expected frontmatter error, got: %s", resBadFm)
	}
}

func TestToolManageEditAndPatch(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)
	ClearSkillsPromptCache()

	// Create a skill first
	createArgs := map[string]interface{}{
		"action":  "create",
		"name":    "edit-me",
		"content": VALID_SKILL,
	}
	argsJSON, _ := json.Marshal(createArgs)
	_, _ = ExecuteSkillManage(string(argsJSON), SkillManageOptions{GuardEnabled: true})

	// Edit skill
	updatedSkill := strings.Replace(VALID_SKILL, "step two", "step three", 1)
	editArgs := map[string]interface{}{
		"action":  "edit",
		"name":    "edit-me",
		"content": updatedSkill,
	}
	editJSON, _ := json.Marshal(editArgs)
	resEdit, isErr := ExecuteSkillManage(string(editJSON), SkillManageOptions{GuardEnabled: true})
	if isErr {
		t.Fatalf("edit failed: %s", resEdit)
	}

	skillMd := filepath.Join(tempHome, "skills", "edit-me", "SKILL.md")
	data, _ := os.ReadFile(skillMd)
	if !strings.Contains(string(data), "step three") {
		t.Fatalf("expected edited content to contain 'step three'")
	}

	// Patch skill happy path
	patchArgs := map[string]interface{}{
		"action":     "patch",
		"name":       "edit-me",
		"old_string": "step one",
		"new_string": "phase one",
	}
	patchJSON, _ := json.Marshal(patchArgs)
	resPatch, isErr := ExecuteSkillManage(string(patchJSON), SkillManageOptions{GuardEnabled: true})
	if isErr {
		t.Fatalf("patch failed: %s", resPatch)
	}
	data, _ = os.ReadFile(skillMd)
	if !strings.Contains(string(data), "phase one") {
		t.Fatalf("expected patched content to contain 'phase one'")
	}

	// Patch no match error
	patchNoMatchArgs := map[string]interface{}{
		"action":     "patch",
		"name":       "edit-me",
		"old_string": "nonexistent xyz",
		"new_string": "replacement",
	}
	patchNoMatchJSON, _ := json.Marshal(patchNoMatchArgs)
	resNoMatch, _ := ExecuteSkillManage(string(patchNoMatchJSON), SkillManageOptions{GuardEnabled: true})
	if !strings.Contains(resNoMatch, "find") && !strings.Contains(resNoMatch, "match") {
		t.Fatalf("expected match error, got: %s", resNoMatch)
	}
}

func TestToolManageDelete(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)
	ClearSkillsPromptCache()

	createArgs := map[string]interface{}{
		"action":  "create",
		"name":    "gone",
		"content": VALID_SKILL,
	}
	argsJSON, _ := json.Marshal(createArgs)
	_, _ = ExecuteSkillManage(string(argsJSON), SkillManageOptions{GuardEnabled: true})

	deleteArgs := map[string]interface{}{
		"action": "delete",
		"name":   "gone",
	}
	deleteJSON, _ := json.Marshal(deleteArgs)
	resDelete, isErr := ExecuteSkillManage(string(deleteJSON), SkillManageOptions{GuardEnabled: true})
	if isErr {
		t.Fatalf("delete failed: %s", resDelete)
	}

	skillDir := filepath.Join(tempHome, "skills", "gone")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatal("expected skill directory to be deleted")
	}
}

func TestToolManageWriteAndRemoveFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)
	ClearSkillsPromptCache()

	createArgs := map[string]interface{}{
		"action":  "create",
		"name":    "files",
		"content": VALID_SKILL,
	}
	argsJSON, _ := json.Marshal(createArgs)
	_, _ = ExecuteSkillManage(string(argsJSON), SkillManageOptions{GuardEnabled: true})

	// Write file
	writeArgs := map[string]interface{}{
		"action":       "write_file",
		"name":         "files",
		"file_path":    "references/note.md",
		"file_content": "# Note\n",
	}
	writeJSON, _ := json.Marshal(writeArgs)
	resWrite, isErr := ExecuteSkillManage(string(writeJSON), SkillManageOptions{GuardEnabled: true})
	if isErr {
		t.Fatalf("write_file failed: %s", resWrite)
	}

	refPath := filepath.Join(tempHome, "skills", "files", "references", "note.md")
	if _, err := os.Stat(refPath); err != nil {
		t.Fatalf("expected written file to exist: %v", err)
	}

	// Traversal check in write_file
	traversalArgs := map[string]interface{}{
		"action":       "write_file",
		"name":         "files",
		"file_path":    "../SKILL.md",
		"file_content": "evil",
	}
	traversalJSON, _ := json.Marshal(traversalArgs)
	resTraversal, _ := ExecuteSkillManage(string(traversalJSON), SkillManageOptions{GuardEnabled: true})
	if !strings.Contains(resTraversal, "traversal") && !strings.Contains(resTraversal, "allowed") && !strings.Contains(resTraversal, "escapes") {
		t.Fatalf("expected traversal block, got: %s", resTraversal)
	}

	// Remove file
	removeArgs := map[string]interface{}{
		"action":    "remove_file",
		"name":      "files",
		"file_path": "references/note.md",
	}
	removeJSON, _ := json.Marshal(removeArgs)
	resRemove, isErr := ExecuteSkillManage(string(removeJSON), SkillManageOptions{GuardEnabled: true})
	if isErr {
		t.Fatalf("remove_file failed: %s", resRemove)
	}
	if _, err := os.Stat(refPath); !os.IsNotExist(err) {
		t.Fatal("expected file to be removed")
	}
}

func TestToolManageMarkCreatedAsAgent(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)
	ClearSkillsPromptCache()

	createArgs := map[string]interface{}{
		"action":  "create",
		"name":    "agent-skill",
		"content": VALID_SKILL,
	}
	argsJSON, _ := json.Marshal(createArgs)

	resStr, isErr := ExecuteSkillManage(string(argsJSON), SkillManageOptions{
		GuardEnabled:       true,
		MarkCreatedAsAgent: true,
	})
	if isErr {
		t.Fatalf("create failed: %s", resStr)
	}

	usage := LoadUsage()
	rec, ok := usage["agent-skill"]
	if !ok || rec.CreatedBy == nil || *rec.CreatedBy != "agent" {
		t.Fatalf("expected created_by=agent in usage, got %+v", rec)
	}
}
