package skills

import (
	"path/filepath"
	"strings"
)

func hasTraversalComponent(filePath string) bool {
	parts := strings.FieldsFunc(filePath, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	return false
}

func isPathWithinDir(filePath, baseDir string) bool {
	resolvedBase, errBase := filepath.Abs(baseDir)
	resolvedTarget, errTarget := filepath.Abs(filePath)
	if errBase != nil || errTarget != nil {
		return false
	}
	if resolvedTarget == resolvedBase {
		return true
	}
	// Add separator suffix to base to prevent matching partial directory names like /foo/bar-nested with base /foo/bar
	sep := string(filepath.Separator)
	prefix := resolvedBase
	if !strings.HasSuffix(prefix, sep) {
		prefix += sep
	}
	return strings.HasPrefix(resolvedTarget, prefix)
}

func validateWithinDir(targetPath, baseDir string) string {
	resolvedBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "Failed to resolve skill directory."
	}
	// Join relative path
	resolvedTarget := filepath.Join(resolvedBase, targetPath)
	if !isPathWithinDir(resolvedTarget, resolvedBase) {
		return "Path escapes the skill directory."
	}
	return ""
}
