package agent

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/enough/enough/backend/opencode"
)

func globTool() opencode.Tool {
	return opencode.Tool{
		Type: "function",
		Function: opencode.ToolFunction{
			Name: "glob",
			Description: "Fast file pattern matching. Returns workspace-relative paths matching a glob " +
				"pattern, sorted alphabetically. Supports ** for recursive matching (e.g. \"**/*.go\", " +
				"\"src/**/*.ts\"). Use this when you know part of a filename or extension.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"pattern": {"type": "string", "description": "Glob pattern, e.g. **/*.go"},
					"path": {"type": "string", "description": "Directory to search in (default workspace root)"}
				},
				"required": ["pattern"]
			}`),
		},
	}
}

const maxGlobResults = 200

func (a *Agent) toolGlob(argsJSON string) toolResult {
	var args struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}
	if strings.TrimSpace(args.Pattern) == "" {
		return toolResult{output: "pattern is required", isErr: true}
	}

	root := args.Path
	if root == "" {
		root = "."
	}
	rootAbs, err := a.resolvePath(root)
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	var matches []string
	err = filepath.WalkDir(rootAbs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) && p != rootAbs {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(rootAbs, p)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if matchGlob(args.Pattern, rel) {
			workRel, wErr := filepath.Rel(a.workDir, p)
			if wErr != nil {
				workRel = rel
			}
			matches = append(matches, filepath.ToSlash(workRel))
		}
		if len(matches) >= maxGlobResults {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return toolResult{output: err.Error(), isErr: true}
	}

	if len(matches) == 0 {
		return toolResult{output: "no matches"}
	}
	sort.Strings(matches)
	out := strings.Join(matches, "\n")
	if len(matches) >= maxGlobResults {
		out += fmt.Sprintf("\n... truncated at %d matches ...", maxGlobResults)
	}
	return toolResult{output: out}
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", ".idea", ".vscode", "dist", "build":
		return true
	}
	return false
}

// matchGlob reports whether a slash-separated relative path matches a glob
// pattern supporting ** (any number of path segments including none).
func matchGlob(pattern, path string) bool {
	// Fast path: a bare "*.ext" pattern matches any file by base name.
	if !strings.Contains(pattern, "/") && !strings.Contains(pattern, "**") {
		if ok, _ := filepath.Match(pattern, filepath.Base(path)); ok {
			return true
		}
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

func matchSegments(pat, name []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			// Collapse consecutive **.
			rest := pat[1:]
			if len(rest) == 0 {
				return true
			}
			for i := 0; i <= len(name); i++ {
				if matchSegments(rest, name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, _ := filepath.Match(pat[0], name[0]); !ok {
			return false
		}
		pat = pat[1:]
		name = name[1:]
	}
	return len(name) == 0
}
