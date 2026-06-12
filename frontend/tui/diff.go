package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func pathFromToolArgs(argsJSON string) string {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	return strings.TrimSpace(args.Path)
}

func resolveToolFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Clean(home)
		}
		return ""
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			path = filepath.Join(cwd, path)
		}
	}
	return filepath.Clean(path)
}

// readFileForDiff returns the current file contents. ok is false only when the
// path cannot be resolved or read for a reason other than missing file.
func readFileForDiff(pathArg string) (content string, ok bool) {
	path := resolveToolFilePath(pathArg)
	if path == "" {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", true
		}
		return "", false
	}
	return string(data), true
}

// finalizeFileToolDiff compares a snapshot taken at tool start with the file on
// disk after the tool finishes. This is the only accurate source for edit_file
// line counts — previewing from old_string/new_string fails when the model passes
// stale context or when multiple edits hit the same file in one turn.
func finalizeFileToolDiff(toolName, argsJSON, before string, toolError bool) (added, removed int) {
	if toolError {
		return 0, 0
	}

	path := pathFromToolArgs(argsJSON)
	after, ok := readFileForDiff(path)
	if !ok && toolName == "write_file" {
		var args struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
			return lineDiff(before, args.Content)
		}
		return 0, 0
	}
	if !ok {
		return 0, 0
	}
	return lineDiff(before, after)
}

func diffWriteFile(argsJSON string) (added, removed int) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return 0, 0
	}

	before, ok := readFileForDiff(args.Path)
	if !ok {
		return countLines(args.Content), 0
	}
	return lineDiff(before, args.Content)
}

func lineDiff(old, new string) (added, removed int) {
	a := splitLines(old)
	b := splitLines(new)
	lcs := lcsLength(a, b)
	return len(b) - lcs, len(a) - lcs
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func lcsLength(a, b []string) int {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return 0
	}

	prev := make([]int, n+1)
	cur := make([]int, n+1)
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				cur[j] = prev[j-1] + 1
			} else if prev[j] >= cur[j-1] {
				cur[j] = prev[j]
			} else {
				cur[j] = cur[j-1]
			}
		}
		prev, cur = cur, prev
		for j := range cur {
			cur[j] = 0
		}
	}
	return prev[n]
}
