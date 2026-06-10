package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func diffWriteFile(argsJSON string) (added, removed int) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return 0, 0
	}

	path := args.Path
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			path = filepath.Join(cwd, path)
		}
	}

	old := ""
	if data, err := os.ReadFile(path); err == nil {
		old = string(data)
	}
	return lineDiff(old, args.Content)
}

func diffEditFile(argsJSON string) (added, removed int) {
	var args struct {
		Path       string `json:"path"`
		OldString  string `json:"old_string"`
		NewString  string `json:"new_string"`
		ReplaceAll bool   `json:"replace_all"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return 0, 0
	}

	path := args.Path
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			path = filepath.Join(cwd, path)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return countLines(args.NewString), countLines(args.OldString)
	}

	newContent, _, err := applyEditPreview(string(data), args.OldString, args.NewString, args.ReplaceAll)
	if err != nil {
		return countLines(args.NewString), countLines(args.OldString)
	}
	return lineDiff(string(data), newContent)
}

func applyEditPreview(content, old, new string, replaceAll bool) (string, int, error) {
	if old == "" {
		return "", 0, nil
	}
	count := strings.Count(content, old)
	if count == 0 {
		return "", 0, nil
	}
	if !replaceAll {
		if count > 1 {
			return "", 0, nil
		}
		return strings.Replace(content, old, new, 1), 1, nil
	}
	return strings.Replace(content, old, new, -1), count, nil
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
