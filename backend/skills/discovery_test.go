package skills

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestIterSkillIndexFilesExcludesNodeModulesAndGit(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "skills-root")

	// Setup normal skill
	mySkillDir := filepath.Join(root, "github", "my-skill")
	if err := os.MkdirAll(mySkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mySkillDir, "SKILL.md"), []byte("---\nname: my-skill\ndescription: test\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Setup node_modules skill
	nodeSkillDir := filepath.Join(root, "node_modules", "pkg", "nested-skill")
	if err := os.MkdirAll(nodeSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nodeSkillDir, "SKILL.md"), []byte("---\nname: nested\ndescription: hidden\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	found := IterSkillIndexFiles(root, "SKILL.md")
	if len(found) != 1 {
		t.Fatalf("expected 1 skill file, found %d: %v", len(found), found)
	}
	if !strings.Contains(found[0], "my-skill") {
		t.Fatalf("expected my-skill, got %s", found[0])
	}
}

func TestIsExcludedSkillPath(t *testing.T) {
	if !isExcludedSkillPath("/foo/node_modules/bar/SKILL.md") {
		t.Error("expected /foo/node_modules/bar/SKILL.md to be excluded")
	}
	if isExcludedSkillPath("/foo/github/bar/SKILL.md") {
		t.Error("expected /foo/github/bar/SKILL.md not to be excluded")
	}
}

func TestSkillMatchesPlatform(t *testing.T) {
	winFM := map[string]interface{}{"platforms": []interface{}{"windows"}}
	isWin := runtime.GOOS == "windows"
	if skillMatchesPlatform(winFM) != isWin {
		t.Fatalf("expected platform match to be %t", isWin)
	}

	emptyFM := map[string]interface{}{}
	if !skillMatchesPlatform(emptyFM) {
		t.Error("expected empty platforms to always match")
	}
}

func TestLoadSkillsFromDirAssignsCategory(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "cat-skill")
	skillPath := filepath.Join(root, "devops", "deploy")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("---\nname: deploy\ndescription: Deploy things\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	skills, _ := loadSkillsFromDirInternal(root, "test", false, nil, root, root)
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Category != "devops" {
		t.Fatalf("expected category devops, got %q", skills[0].Category)
	}
}

func TestLoadSkillsScansHomeAndLegacy(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("ENOUGH_HOME", filepath.Join(tempDir, "enough-home"))
	t.Setenv("HOME", filepath.Join(tempDir, "user-home"))
	flameHome := filepath.Join(tempDir, "enough-home")
	agentDir := filepath.Join(tempDir, "agent")
	cwd := filepath.Join(tempDir, "project")

	// Global skill
	globalSkillDir := filepath.Join(flameHome, "skills", "global-skill")
	if err := os.MkdirAll(globalSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalSkillDir, "SKILL.md"), []byte("---\nname: global-skill\ndescription: global\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Legacy skill
	legacySkillDir := filepath.Join(agentDir, "skills", "legacy-skill")
	if err := os.MkdirAll(legacySkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacySkillDir, "SKILL.md"), []byte("---\nname: legacy-skill\ndescription: legacy\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Project skill under .flame (should be ignored, enough uses .enough)
	projectSkillDir := filepath.Join(cwd, ".flame", "skills", "project-skill")
	if err := os.MkdirAll(projectSkillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectSkillDir, "SKILL.md"), []byte("---\nname: project-skill\ndescription: project\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := LoadSkills(LoadSkillsOptions{
		Cwd:             cwd,
		AgentDir:        agentDir,
		SkillPaths:      []string{},
		IncludeDefaults: true,
	})

	var names []string
	for _, s := range res.Skills {
		names = append(names, s.Name)
	}
	sort.Strings(names)

	hasGlobal := false
	hasLegacy := false
	hasProject := false
	for _, n := range names {
		if n == "global-skill" {
			hasGlobal = true
		}
		if n == "legacy-skill" {
			hasLegacy = true
		}
		if n == "project-skill" {
			hasProject = true
		}
	}

	if !hasGlobal {
		t.Errorf("expected global-skill in %v", names)
	}
	if !hasLegacy {
		t.Errorf("expected legacy-skill in %v", names)
	}
	if hasProject {
		t.Errorf("did not expect project-skill (nested in .flame) in %v", names)
	}
}

func TestSkillsDirMatchesLoadSkillsPrimaryRoot(t *testing.T) {
	dir := SkillsDir()
	if !strings.Contains(dir, "skills") {
		t.Fatalf("expected SkillsDir() to contain 'skills', got %s", dir)
	}
}
