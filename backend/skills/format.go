package skills

import (
	"strings"
)

func escapeXml(str string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return r.Replace(str)
}

func FormatSkillsForPrompt(skills []Skill) string {
	var visibleSkills []Skill
	for _, s := range skills {
		if !s.DisableModelInvocation {
			visibleSkills = append(visibleSkills, s)
		}
	}

	if len(visibleSkills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\nThe following skills provide specialized instructions for specific tasks.\n")
	sb.WriteString("Use the read tool to load a skill's file when the task matches its description.\n")
	sb.WriteString("When a skill file references a relative path, resolve it against the skill directory (parent of SKILL.md / dirname of the path) and use that absolute path in tool commands.\n\n")
	sb.WriteString("<available_skills>\n")

	for _, skill := range visibleSkills {
		sb.WriteString("  <skill>\n")
		sb.WriteString("    <name>" + escapeXml(skill.Name) + "</name>\n")
		sb.WriteString("    <description>" + escapeXml(skill.Description) + "</description>\n")
		sb.WriteString("    <location>" + escapeXml(skill.FilePath) + "</location>\n")
		sb.WriteString("  </skill>\n")
	}

	sb.WriteString("</available_skills>")
	return sb.String()
}
