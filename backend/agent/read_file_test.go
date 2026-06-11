package agent

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestToolReadFileReportsLineCount(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		content   string
		wantLines int
	}{
		"trailing newline":    {"a\nb\nc\n", 3},
		"no trailing newline": {"a\nb\nc", 3},
		"single line":         {"only", 1},
		"empty":               {"", 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".txt")
			mustWrite(t, path, tc.content)
			a := &Agent{workDir: dir}
			res := a.toolReadFile(`{"path":"` + path + `"}`)
			if res.isErr {
				t.Fatalf("read_file error: %s", res.output)
			}
			wantHeader := "Read " + strconv.Itoa(tc.wantLines) + " lines from "
			if !strings.HasPrefix(res.output, wantHeader) {
				t.Fatalf("expected header %q, got %q", wantHeader, res.output)
			}
		})
	}
}

func TestToolReadFileTruncatedOutputReportsFullLineCount(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.txt")
	content := strings.Repeat("line\n", 15000)
	mustWrite(t, path, content)

	a := &Agent{workDir: dir}
	res := a.toolReadFile(`{"path":"` + path + `"}`)
	if res.isErr {
		t.Fatalf("read_file error: %s", res.output)
	}
	if !strings.HasPrefix(res.output, "Read 15000 lines from ") {
		t.Fatalf("expected full line count in header, got %q", strings.SplitN(res.output, "\n", 2)[0])
	}
	if !strings.Contains(res.output, "... truncated ...") {
		t.Fatalf("expected truncation marker")
	}
}
