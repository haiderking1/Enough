package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/enough/enough/backend/config"
)

var skillInvalidChars = regexp.MustCompile(`[^a-z0-9-]`)
var skillMultiHyphen = regexp.MustCompile(`-{2,}`)

func SkillNameToSlashSlug(name string) string {
	cmd := strings.ToLower(name)
	cmd = strings.ReplaceAll(cmd, " ", "-")
	cmd = strings.ReplaceAll(cmd, "_", "-")
	cmd = skillInvalidChars.ReplaceAllString(cmd, "")
	cmd = skillMultiHyphen.ReplaceAllString(cmd, "-")
	cmd = strings.Trim(cmd, "-")
	return cmd
}

func BuildSkillInvocationMessage(loadedSkill map[string]interface{}, skillDir, userInstruction, sessionId string) string {
	name, _ := loadedSkill["name"].(string)
	if name == "" {
		name = "skill"
	}

	activationNote := fmt.Sprintf("[IMPORTANT: The user has invoked the %q skill. Follow the skill instructions below as your primary guidance for this turn.]", name)
	content, _ := loadedSkill["content"].(string)

	// Preprocess content
	content = PreprocessSkillContent(content, skillDir, sessionId, true)

	var parts []string
	parts = append(parts, activationNote, "", strings.TrimSpace(content))

	if skillDir != "" {
		parts = append(parts, "", fmt.Sprintf("[Skill directory: %s]", skillDir),
			"Resolve any relative paths in this skill (e.g. `scripts/foo.js`, `templates/config.yaml`) against that directory, then run them with the terminal tool using the absolute path.")
	}

	var supporting []string
	if linkedFilesVal, ok := loadedSkill["linked_files"]; ok && linkedFilesVal != nil {
		if linkedMap, ok := linkedFilesVal.(map[string][]string); ok {
			for _, entries := range linkedMap {
				supporting = append(supporting, entries...)
			}
		}
	}

	if len(supporting) == 0 && skillDir != "" {
		if _, err := os.Stat(skillDir); err == nil {
			for _, subdir := range []string{"references", "templates", "scripts", "assets"} {
				subdirPath := filepath.Join(skillDir, subdir)
				if _, err := os.Stat(subdirPath); err == nil {
					_ = filepath.Walk(subdirPath, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							return nil
						}
						if !info.IsDir() {
							rel, err := filepath.Rel(skillDir, path)
							if err == nil {
								supporting = append(supporting, filepath.ToSlash(rel))
							}
						}
						return nil
					})
				}
			}
		}
	}

	if len(supporting) > 0 && skillDir != "" {
		skillViewTarget := filepath.Base(skillDir)
		skillsRoot := SkillsDir()
		if rel, err := filepath.Rel(skillsRoot, skillDir); err == nil {
			skillViewTarget = filepath.ToSlash(rel)
		}

		parts = append(parts, "", "[This skill has supporting files:]")
		for _, sf := range supporting {
			parts = append(parts, fmt.Sprintf("- %s  ->  %s", sf, filepath.Join(skillDir, sf)))
		}
		parts = append(parts, fmt.Sprintf("\nLoad any of these with skill_view(name=%q, file_path=%q), or run scripts directly by absolute path.", skillViewTarget, "<path>"))
	}

	if strings.TrimSpace(userInstruction) != "" {
		parts = append(parts, "", fmt.Sprintf("The user has provided the following instruction alongside the skill invocation: %s", strings.TrimSpace(userInstruction)))
	}

	return strings.Join(parts, "\n")
}

func ExpandSkillSlashCommand(skillName, userArgs, workDir string, cfg config.Runtime, sessionId string) (string, string, error) {
	// Execute executeSkillView logic (preprocess = false)
	viewRes := executeSkillViewInternal(skillName, "", workDir, cfg, sessionId, false)
	if !viewRes.Success {
		return "", "", errors.New(viewRes.Error)
	}

	loadedSkill := map[string]interface{}{
		"name":         viewRes.Name,
		"content":      viewRes.RawContent,
		"linked_files": viewRes.LinkedFiles,
	}

	message := BuildSkillInvocationMessage(loadedSkill, viewRes.SkillDir, userArgs, sessionId)
	cleanBody := PreprocessSkillContent(viewRes.RawContent, viewRes.SkillDir, sessionId, true)
	return message, cleanBody, nil
}
