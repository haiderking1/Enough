package markdown

import (
	"github.com/enough/enough/frontend/tui/highlight"
	"github.com/enough/enough/frontend/tui/term"
)

// Theme maps markdown elements to ANSI-styled terminal output.
type Theme struct {
	Plain           func(string) string
	Bold            func(string) string
	Italic          func(string) string
	Strikethrough   func(string) string
	Code            func(string) string
	Link            func(string) string
	LinkURL         func(string) string
	Heading         func(string) string
	Quote           func(string) string
	QuoteBorder     func(string) string
	HR              func(string) string
	ListBullet      func(string) string
	CodeBlockBorder func(string) string
	CodeBlockIndent string
	Image           func(string) string
	HighlightCode   func(lang, code string) []string
}

func (t Theme) withDefaults() Theme {
	p := highlight.GruvboxDark()
	if t.Plain == nil {
		t.Plain = func(s string) string { return p.Paint(p.Fg, s) }
	}
	if t.Bold == nil {
		t.Bold = p.Bold
	}
	if t.Italic == nil {
		t.Italic = p.Italic
	}
	if t.Strikethrough == nil {
		t.Strikethrough = func(s string) string { return "\033[9m" + s + "\033[0m" }
	}
	if t.Code == nil {
		t.Code = p.InlineCode
	}
	if t.Link == nil {
		t.Link = func(s string) string { return p.Paint(p.Special, s) }
	}
	if t.LinkURL == nil {
		t.LinkURL = func(s string) string { return p.Paint(p.Comment, s) }
	}
	if t.Heading == nil {
		t.Heading = p.Bold
	}
	if t.Quote == nil {
		t.Quote = p.Italic
	}
	if t.QuoteBorder == nil {
		t.QuoteBorder = func(s string) string { return p.Paint(p.Comment, s) }
	}
	if t.HR == nil {
		t.HR = func(s string) string { return p.Paint(p.Comment, s) }
	}
	if t.ListBullet == nil {
		t.ListBullet = func(s string) string { return s }
	}
	if t.CodeBlockBorder == nil {
		t.CodeBlockBorder = func(s string) string { return p.Paint(p.Special, s) }
	}
	if t.Image == nil {
		t.Image = func(s string) string { return p.Paint(p.Comment, s) }
	}
	if t.CodeBlockIndent == "" {
		t.CodeBlockIndent = "  "
	}
	if t.HighlightCode == nil {
		t.HighlightCode = func(lang, code string) []string {
			out := highlight.HighlightCode(lang, code, p)
			if out == "" {
				return nil
			}
			lines := splitLines(out)
			if len(lines) == 0 {
				return nil
			}
			return lines
		}
	}
	return t
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func visibleWidth(s string) int {
	return term.VisibleWidth(s)
}
