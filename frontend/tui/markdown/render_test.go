package markdown

import (
	"strings"
	"testing"
)

func stripANSI(s string) string {
	var out strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if inEscape {
			if b == 'm' {
				inEscape = false
			}
			continue
		}
		if b == '\x1b' {
			inEscape = true
			continue
		}
		out.WriteByte(b)
	}
	return out.String()
}

func TestRenderBold(t *testing.T) {
	out := Render("**hello** world", 40, Theme{})
	plain := stripANSI(out)
	if !strings.Contains(plain, "hello") || strings.Contains(plain, "**") {
		t.Fatalf("expected bold rendered, got %q", plain)
	}
}

func TestRenderTable(t *testing.T) {
	md := `| Factor | Python | Go |
| --- | --- | --- |
| Parsing | great | ok |`
	out := Render(md, 60, Theme{})
	plain := stripANSI(out)
	if !strings.Contains(plain, "┌") || !strings.Contains(plain, "│") {
		t.Fatalf("expected box table, got %q", plain)
	}
	if !strings.Contains(plain, "Python") || !strings.Contains(plain, "Parsing") {
		t.Fatalf("expected table content, got %q", plain)
	}
}

func TestRenderList(t *testing.T) {
	md := `- one
  - nested
- two`
	out := Render(md, 40, Theme{})
	plain := stripANSI(out)
	if !strings.Contains(plain, "- one") || !strings.Contains(plain, "- nested") {
		t.Fatalf("expected nested list, got %q", plain)
	}
}

func TestRenderCodeFence(t *testing.T) {
	out := Render("```go\nfunc main() {}\n```", 40, Theme{})
	plain := stripANSI(out)
	if !strings.Contains(plain, "func main") || !strings.Contains(plain, "```go") {
		t.Fatalf("expected code fence, got %q", plain)
	}
}

func TestRenderTaskList(t *testing.T) {
	md := `- [x] done
- [ ] todo`
	out := Render(md, 40, Theme{})
	plain := stripANSI(out)
	if !strings.Contains(plain, "[x] done") || !strings.Contains(plain, "[ ] todo") {
		t.Fatalf("expected task list markers, got %q", plain)
	}
}

func TestRenderHyperlink(t *testing.T) {
	undo := CapabilitiesForTest(Capabilities{Hyperlinks: true})
	defer undo()

	out := Render("[click me](https://example.com)", 40, Theme{})
	if !strings.Contains(out, "\x1b]8;;https://example.com") {
		t.Fatalf("expected OSC 8 hyperlink, got %q", out)
	}
}

func TestRenderImage(t *testing.T) {
	out := Render("![diagram](https://example.com/a.png)", 40, Theme{})
	plain := stripANSI(out)
	if !strings.Contains(plain, "[Image: diagram") || !strings.Contains(plain, "example.com/a.png") {
		t.Fatalf("expected image placeholder, got %q", plain)
	}
}

func TestWrapPlainText(t *testing.T) {
	lines := wrapTextWithANSI("this is a very long link label for wrapping", 20)
	if len(lines) < 2 {
		t.Fatalf("expected wrap, got %v", lines)
	}
}

func TestWrapHyperlinkAcrossLines(t *testing.T) {
	undo := CapabilitiesForTest(Capabilities{Hyperlinks: true})
	defer undo()

	longText := "this is a very long link label for wrapping"
	theme := Theme{}.withDefaults()
	styled := Hyperlink(theme.Link(longText), "https://example.com/long")
	if len(splitWordsPreserveANSI(styled)) < 2 {
		t.Fatalf("expected splittable hyperlink words")
	}
	lines := wrapTextWithANSI(styled, 20)
	if len(lines) < 2 {
		t.Fatalf("expected wrapped hyperlink, got %d line(s): %q", len(lines), lines)
	}
}

func TestRenderHyperlinkIntegration(t *testing.T) {
	undo := CapabilitiesForTest(Capabilities{Hyperlinks: true})
	defer undo()

	out := Render("[docs](https://example.com/docs)", 80, Theme{})
	if !strings.Contains(out, "\x1b]8;;https://example.com/docs") {
		t.Fatalf("expected OSC 8 in rendered markdown, got %q", out)
	}
}
