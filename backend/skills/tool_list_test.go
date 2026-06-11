package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/enough/enough/backend/config"
)

func TestToolListEmpty(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)

	cfg := config.Runtime{
		Skills: config.SkillsSettings{
			Enabled:  true,
			Disabled: []string{"enough"},
		},
	}
	resStr, isErr := ExecuteSkillsList(`{}`, tempHome, cfg)
	if isErr {
		t.Fatalf("ExecuteSkillsList failed: %s", resStr)
	}

	var res struct {
		Success bool   `json:"success"`
		Count   int    `json:"count"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(resStr), &res); err != nil {
		t.Fatal(err)
	}

	if !res.Success {
		t.Fatal("expected success: true")
	}
	if res.Count != 0 {
		t.Fatalf("expected count 0, got %d", res.Count)
	}
	if !strings.Contains(res.Message, "No skills found") {
		t.Fatalf("expected 'No skills found' in message, got: %q", res.Message)
	}
}

func TestToolListWithMetadataAndFilter(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("ENOUGH_HOME", tempHome)
	t.Setenv("HOME", tempHome)

	// Create a skill under devops category
	deployDir := filepath.Join(tempHome, "skills", "devops", "deploy")
	if err := os.MkdirAll(deployDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deployDir, "SKILL.md"), []byte("---\nname: deploy\ndescription: Deploy services\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create another skill under test category
	testDir := filepath.Join(tempHome, "skills", "testcat", "runtest")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "SKILL.md"), []byte("---\nname: runtest\ndescription: Run tests\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Runtime{
		Skills: config.SkillsSettings{
			Enabled:  true,
			Disabled: []string{"enough"},
		},
	}

	// 1. Check listing all
	resStr, isErr := ExecuteSkillsList(`{}`, tempHome, cfg)
	if isErr {
		t.Fatalf("ExecuteSkillsList failed: %s", resStr)
	}
	var res struct {
		Success    bool     `json:"success"`
		Count      int      `json:"count"`
		Skills     []struct {
			Name     string `json:"name"`
			Category string `json:"category"`
		} `json:"skills"`
		Categories []string `json:"categories"`
	}
	if err := json.Unmarshal([]byte(resStr), &res); err != nil {
		t.Fatal(err)
	}
	if res.Count != 2 {
		t.Fatalf("expected count 2, got %d", res.Count)
	}
	if res.Categories[0] != "devops" || res.Categories[1] != "testcat" {
		t.Fatalf("expected categories [devops, testcat], got: %v", res.Categories)
	}

	// 2. Check filtering by category
	resStrFiltered, isErr := ExecuteSkillsList(`{"category": "devops"}`, tempHome, cfg)
	if isErr {
		t.Fatalf("ExecuteSkillsList filtered failed: %s", resStrFiltered)
	}
	var resFiltered struct {
		Count  int `json:"count"`
		Skills []struct {
			Name string `json:"name"`
		} `json:"skills"`
	}
	if err := json.Unmarshal([]byte(resStrFiltered), &resFiltered); err != nil {
		t.Fatal(err)
	}
	if resFiltered.Count != 1 {
		t.Fatalf("expected count 1, got %d", resFiltered.Count)
	}
	if resFiltered.Skills[0].Name != "deploy" {
		t.Fatalf("expected skill 'deploy', got: %q", resFiltered.Skills[0].Name)
	}
}
