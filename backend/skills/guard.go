package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var InvisibleChars = []string{
	"\u200b", "\u200c", "\u200d", "\u2060", "\u2062", "\u2063", "\u2064", "\ufeff",
	"\u202a", "\u202b", "\u202c", "\u202d", "\u202e", "\u2066", "\u2067", "\u2068", "\u2069",
}

const (
	MaxFileCount    = 50
	MaxTotalSizeKB  = 1024
	MaxSingleFileKB = 256
)

func unicodeCharName(char string) string {
	names := map[string]string{
		"\u200b": "zero-width space",
		"\u200c": "zero-width non-joiner",
		"\u200d": "zero-width joiner",
		"\u2060": "word joiner",
		"\u2062": "invisible times",
		"\u2063": "invisible separator",
		"\u2064": "invisible plus",
		"\ufeff": "BOM/zero-width no-break space",
		"\u202a": "LTR embedding",
		"\u202b": "RTL embedding",
		"\u202c": "pop directional",
		"\u202d": "LTR override",
		"\u202e": "RTL override",
		"\u2066": "LTR isolate",
		"\u2067": "RTL isolate",
		"\u2068": "first strong isolate",
		"\u2069": "pop directional isolate",
	}
	if name, ok := names[char]; ok {
		return name
	}
	if len(char) > 0 {
		return fmt.Sprintf("U+%04X", []rune(char)[0])
	}
	return "????"
}

func resolveTrustLevel(source string) string {
	normalized := strings.TrimPrefix(source, "skills-sh/")
	normalized = strings.TrimPrefix(normalized, "skills.sh/")
	normalized = strings.TrimPrefix(normalized, "skils-sh/")
	normalized = strings.TrimPrefix(normalized, "skils.sh/")
	normalized = strings.ToLower(normalized)

	if normalized == "agent-created" {
		return "agent-created"
	}
	if normalized == "official" {
		return "builtin"
	}
	trusted := []string{"openai/skills", "anthropics/skills", "huggingface/skills"}
	for _, repo := range trusted {
		if normalized == repo || strings.HasPrefix(normalized, repo+"/") {
			return "trusted"
		}
	}
	return "community"
}

func determineVerdict(findings []SkillGuardFinding) string {
	if len(findings) == 0 {
		return "safe"
	}
	for _, f := range findings {
		if f.Severity == "critical" {
			return "dangerous"
		}
	}
	for _, f := range findings {
		if f.Severity == "high" {
			return "caution"
		}
	}
	return "safe"
}

var scannableExtensions = map[string]bool{
	".md": true, ".txt": true, ".py": true, ".sh": true, ".bash": true,
	".js": true, ".ts": true, ".rb": true, ".yaml": true, ".yml": true,
	".json": true, ".toml": true, ".cfg": true, ".ini": true, ".conf": true,
	".html": true, ".css": true, ".xml": true, ".tex": true, ".r": true,
	".jl": true, ".pl": true, ".php": true,
}

var suspiciousBinaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true, ".bin": true,
	".dat": true, ".com": true, ".msi": true, ".dmg": true, ".app": true,
	".deb": true, ".rpm": true,
}

func ScanSkillFile(filePath, relPath string) []SkillGuardFinding {
	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	if !scannableExtensions[ext] && baseName != "SKILL.md" {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	content := string(data)
	lines := strings.Split(content, "\n")
	var findings []SkillGuardFinding
	seen := make(map[string]bool)

	for _, p := range SkillGuardThreatPatterns {
		for i, line := range lines {
			lineLower := strings.ToLower(line)
			key := fmt.Sprintf("%s:%d", p.PatternID, i+1)
			if seen[key] {
				continue
			}

			matched := false
			// Custom manual verification for non-lookahead patterns or where custom Go matching is needed
			if p.PatternID == "python_os_environ" {
				if p.Regex.MatchString(line) {
					if !strings.Contains(lineLower, "path") {
						matched = true
					}
				}
			} else if p.PatternID == "unpinned_pip_install" {
				if strings.Contains(lineLower, "pip install") || strings.Contains(lineLower, "pip3 install") {
					if !strings.Contains(lineLower, "==") && !strings.Contains(lineLower, "-r") {
						matched = true
					}
				}
			} else if p.PatternID == "unpinned_npm_install" {
				if strings.Contains(lineLower, "npm install") || strings.Contains(lineLower, "npm i ") {
					if !strings.Contains(lineLower, "@") {
						matched = true
					}
				}
			} else {
				matched = p.Regex.MatchString(line)
			}

			if matched {
				seen[key] = true
				matchedText := strings.TrimSpace(line)
				if len(matchedText) > 120 {
					matchedText = matchedText[:117] + "..."
				}
				findings = append(findings, SkillGuardFinding{
					PatternID:   p.PatternID,
					Severity:    p.Severity,
					Category:    p.Category,
					File:        relPath,
					Line:        i + 1,
					Match:       matchedText,
					Description: p.Description,
				})
			}
		}
	}

	for i, line := range lines {
		for _, char := range InvisibleChars {
			if strings.Contains(line, char) {
				findings = append(findings, SkillGuardFinding{
					PatternID:   "invisible_unicode",
					Severity:    "high",
					Category:    "injection",
					File:        relPath,
					Line:        i + 1,
					Match:       fmt.Sprintf("U+%04X (%s)", []rune(char)[0], unicodeCharName(char)),
					Description: fmt.Sprintf("invisible unicode character %s (possible text hiding/injection)", unicodeCharName(char)),
				})
				break
			}
		}
	}

	return findings
}

func checkStructure(skillDir string) []SkillGuardFinding {
	var findings []SkillGuardFinding
	fileCount := 0
	var totalSize int64

	err := filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		fileCount++
		totalSize += info.Size()
		rel, relErr := filepath.Rel(skillDir, path)
		if relErr == nil {
			rel = filepath.ToSlash(rel)
			if info.Size() > MaxSingleFileKB*1024 {
				findings = append(findings, SkillGuardFinding{
					PatternID:   "oversized_file",
					Severity:    "medium",
					Category:    "structural",
					File:        rel,
					Line:        0,
					Match:       fmt.Sprintf("%dKB", info.Size()/1024),
					Description: fmt.Sprintf("file is %dKB (limit: %dKB)", info.Size()/1024, MaxSingleFileKB),
				})
			}
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if suspiciousBinaryExtensions[ext] {
				findings = append(findings, SkillGuardFinding{
					PatternID:   "binary_file",
					Severity:    "critical",
					Category:    "structural",
					File:        rel,
					Line:        0,
					Match:       "binary: " + ext,
					Description: fmt.Sprintf("binary/executable file (%s) should not be in a skill", ext),
				})
			}
		}
		return nil
	})
	if err != nil {
		return findings
	}

	if fileCount > MaxFileCount {
		findings = append(findings, SkillGuardFinding{
			PatternID:   "too_many_files",
			Severity:    "medium",
			Category:    "structural",
			File:        "(directory)",
			Line:        0,
			Match:       fmt.Sprintf("%d files", fileCount),
			Description: fmt.Sprintf("skill has %d files (limit: %d)", fileCount, MaxFileCount),
		})
	}

	if totalSize > MaxTotalSizeKB*1024 {
		findings = append(findings, SkillGuardFinding{
			PatternID:   "oversized_skill",
			Severity:    "high",
			Category:    "structural",
			File:        "(directory)",
			Line:        0,
			Match:       fmt.Sprintf("%dKB total", totalSize/1024),
			Description: fmt.Sprintf("skill is %dKB total (limit: %dKB)", totalSize/1024, MaxTotalSizeKB),
		})
	}

	return findings
}

func ScanSkill(skillPath string, source string) SkillScanResult {
	skillName := filepath.Base(skillPath)
	trustLevel := resolveTrustLevel(source)
	allFindings := checkStructure(skillPath)

	_ = filepath.Walk(skillPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(skillPath, path)
		if relErr == nil {
			allFindings = append(allFindings, ScanSkillFile(path, filepath.ToSlash(rel))...)
		}
		return nil
	})

	verdict := determineVerdict(allFindings)

	// Unique categories
	catMap := make(map[string]bool)
	for _, f := range allFindings {
		catMap[f.Category] = true
	}
	var categories []string
	for cat := range catMap {
		categories = append(categories, cat)
	}
	sort.Strings(categories)

	var summary string
	if len(allFindings) == 0 {
		summary = fmt.Sprintf("%s: clean scan, no threats detected", skillName)
	} else {
		summary = fmt.Sprintf("%s: %s — %d finding(s) in %s", skillName, verdict, len(allFindings), strings.Join(categories, ", "))
	}

	return SkillScanResult{
		SkillName:  skillName,
		Source:     source,
		TrustLevel: trustLevel,
		Verdict:    verdict,
		Findings:   allFindings,
		ScannedAt:  time.Now().Format(time.RFC3339),
		Summary:    summary,
	}
}

func shouldAllowInstall(result SkillScanResult, force bool) (bool, string) {
	policy := map[string][]string{
		"builtin":       {"allow", "allow", "allow"},
		"trusted":       {"allow", "allow", "block"},
		"community":     {"allow", "block", "block"},
		"agent-created": {"allow", "allow", "ask"},
	}

	pList := policy[result.TrustLevel]
	if pList == nil {
		pList = policy["community"]
	}

	var vi int
	switch result.Verdict {
	case "safe":
		vi = 0
	case "caution":
		vi = 1
	default:
		vi = 2
	}

	decision := pList[vi]

	if decision == "allow" {
		return true, fmt.Sprintf("Allowed (%s source, %s verdict)", result.TrustLevel, result.Verdict)
	}

	if force && !(result.Verdict == "dangerous" && (result.TrustLevel == "community" || result.TrustLevel == "trusted")) {
		return true, fmt.Sprintf("Force-installed despite %s verdict (%d findings)", result.Verdict, len(result.Findings))
	}

	if decision == "ask" {
		return false, fmt.Sprintf("Requires confirmation (%s source + %s verdict, %d findings)", result.TrustLevel, result.Verdict, len(result.Findings))
	}

	if result.Verdict == "dangerous" && (result.TrustLevel == "community" || result.TrustLevel == "trusted") {
		return false, fmt.Sprintf("Blocked (%s source + dangerous verdict, %d findings). --force does not override a dangerous verdict.", result.TrustLevel, len(result.Findings))
	}

	return false, fmt.Sprintf("Blocked (%s source + %s verdict, %d findings). Use --force to override.", result.TrustLevel, result.Verdict, len(result.Findings))
}

func FormatScanReport(result SkillScanResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Scan: %s (%s/%s)  Verdict: %s\n", result.SkillName, result.Source, result.TrustLevel, strings.ToUpper(result.Verdict)))

	if len(result.Findings) > 0 {
		sevOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		findings := append([]SkillGuardFinding(nil), result.Findings...)
		sort.Slice(findings, func(i, j int) bool {
			return sevOrder[findings[i].Severity] < sevOrder[findings[j].Severity]
		})

		for _, f := range findings {
			sev := strings.ToUpper(f.Severity)
			if len(sev) < 8 {
				sev += strings.Repeat(" ", 8-len(sev))
			}
			cat := f.Category
			if len(cat) < 14 {
				cat += strings.Repeat(" ", 14-len(cat))
			}
			loc := fmt.Sprintf("%s:%d", f.File, f.Line)
			if len(loc) < 30 {
				loc += strings.Repeat(" ", 30-len(loc))
			}
			matchText := f.Match
			if len(matchText) > 60 {
				matchText = matchText[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("  %s %s %s %q\n", sev, cat, loc, matchText))
		}
		sb.WriteString("\n")
	}

	allowed, reason := shouldAllowInstall(result, false)
	status := "BLOCKED"
	if allowed {
		status = "ALLOWED"
	}
	sb.WriteString(fmt.Sprintf("Decision: %s — %s", status, reason))
	return sb.String()
}

func SecurityScanSkillDir(skillDir string, guardEnabled bool) string {
	if !guardEnabled {
		return ""
	}
	result := ScanSkill(skillDir, "agent-created")
	allowed, reason := shouldAllowInstall(result, false)
	if !allowed {
		report := FormatScanReport(result)
		return fmt.Sprintf("Security scan blocked this skill (%s):\n%s", reason, report)
	}
	return ""
}
