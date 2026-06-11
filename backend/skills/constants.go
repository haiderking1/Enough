package skills

import "regexp"

const (
	MaxSkillNameLength        = 64
	MaxSkillDescriptionLength = 1024
	PromptIndexDescriptionMax = 60
	SkillsPromptCacheMax      = 8
	SkillsSnapshotVersion     = 1
	InlineShellMaxOutput      = 4000
	MaxSkillContentChars      = 100000
	MaxSkillFileBytes         = 1048576
)

var ExcludedSkillDirs = map[string]bool{
	".git":           true,
	".github":        true,
	".hub":           true,
	".archive":       true,
	".venv":          true,
	"venv":           true,
	"node_modules":   true,
	"site-packages":  true,
	"__pycache__":    true,
	".tox":           true,
	".nox":           true,
	".pytest_cache":  true,
	".mypy_cache":    true,
	".ruff_cache":    true,
}

var PlatformMap = map[string]string{
	"macos":   "darwin",
	"linux":   "linux",
	"windows": "windows",
}

var InjectionPatterns = []string{
	"ignore previous instructions",
	"ignore all previous",
	"you are now",
	"disregard your",
	"forget your instructions",
	"new instructions:",
	"system prompt:",
	"<system>",
	"]]>",
}

var AllowedSkillSubdirs = map[string]bool{
	"references": true,
	"templates":  true,
	"scripts":    true,
	"assets":     true,
}

var SkillManageNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
var SkillNameValidRe = regexp.MustCompile(`^[a-z0-9-]+$`)
