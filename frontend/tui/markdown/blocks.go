package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	ext "github.com/yuin/goldmark/extension/ast"
)

func (r *renderer) renderBlocks(nodes []ast.Node, source []byte, width int) []renderLine {
	var lines []renderLine
	for i, node := range nodes {
		nextKind := ""
		if i+1 < len(nodes) {
			nextKind = blockKind(nodes[i+1])
		}
		lines = append(lines, r.renderBlock(node, source, width, nextKind)...)
	}
	return lines
}

func blockKind(n ast.Node) string {
	if n == nil {
		return ""
	}
	switch n.(type) {
	case *ast.Paragraph:
		return "paragraph"
	case *ast.Heading:
		return "heading"
	case *ast.FencedCodeBlock:
		return "code"
	case *ast.List:
		return "list"
	case *ext.Table:
		return "table"
	case *ast.Blockquote:
		return "blockquote"
	case *ast.ThematicBreak:
		return "hr"
	default:
		return "other"
	}
}

func (r *renderer) renderBlock(n ast.Node, source []byte, width int, nextKind string) []renderLine {
	switch n := n.(type) {
	case *ast.Heading:
		return r.renderHeading(n, source, nextKind)
	case *ast.Paragraph, *ast.TextBlock:
		return r.renderParagraphLike(n, source, nextKind)
	case *ast.FencedCodeBlock:
		return r.renderCodeBlock(n, source, nextKind)
	case *ast.List:
		return r.renderList(n, source, 0, width)
	case *ext.Table:
		return r.renderTableLines(n, source, width, nextKind)
	case *ast.Blockquote:
		return r.renderBlockquote(n, source, width, nextKind)
	case *ast.ThematicBreak:
		return r.renderHR(width, nextKind)
	case *ast.HTMLBlock:
		raw := strings.TrimSpace(string(n.Lines().Value(source)))
		if raw == "" {
			return nil
		}
		return []renderLine{rl(r.theme.Plain(raw), false)}
	default:
		return nil
	}
}

func (r *renderer) renderHeading(n *ast.Heading, source []byte, nextKind string) []renderLine {
	style := inlineStyle{apply: func(s string) string { return r.theme.Heading(r.theme.Bold(s)) }}
	text := r.renderInlineChildren(n, source, style)
	if n.Level >= 3 {
		prefix := strings.Repeat("#", n.Level) + " "
		text = r.theme.Heading(prefix) + text
	}
	lines := []renderLine{rl(text, false)}
	if nextKind != "" && nextKind != "space" {
		lines = append(lines, rl("", false))
	}
	return lines
}

func (r *renderer) renderParagraphLike(n ast.Node, source []byte, nextKind string) []renderLine {
	if img := soleImage(n, source); img != nil {
		return r.renderImageURL(string(img.Destination), imageAlt(img, source), nextKind)
	}
	text := r.renderInline(n, source)
	if text == "" {
		return nil
	}
	lines := []renderLine{rl(text, false)}
	if nextKind != "" && nextKind != "list" && nextKind != "space" {
		lines = append(lines, rl("", false))
	}
	return lines
}

func (r *renderer) renderCodeBlock(n *ast.FencedCodeBlock, source []byte, nextKind string) []renderLine {
	lang := string(n.Language(source))
	code := strings.TrimRight(string(n.Lines().Value(source)), "\n")

	var lines []renderLine
	lines = append(lines, rl(r.theme.CodeBlockBorder("```"+lang), true))
	if highlighted := r.theme.HighlightCode(lang, code); len(highlighted) > 0 {
		for _, hl := range highlighted {
			lines = append(lines, rl(r.theme.CodeBlockIndent+hl, true))
		}
	} else if code != "" {
		for _, line := range strings.Split(code, "\n") {
			lines = append(lines, rl(r.theme.CodeBlockIndent+r.theme.Plain(line), true))
		}
	}
	lines = append(lines, rl(r.theme.CodeBlockBorder("```"), true))
	if nextKind != "" && nextKind != "space" {
		lines = append(lines, rl("", false))
	}
	return lines
}

func (r *renderer) renderList(n *ast.List, source []byte, depth int, width int) []renderLine {
	var lines []renderLine
	indent := strings.Repeat("    ", depth)
	start := 1
	if n.IsOrdered() && n.Start != 0 {
		start = n.Start
	}

	itemIndex := 0
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		item, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}
		bullet := "- "
		if n.IsOrdered() {
			bullet = itoa(start+itemIndex) + ". "
		}
		bullet += listItemTaskMarker(item)
		itemIndex++

		firstPrefix := indent + r.theme.ListBullet(bullet)
		contPrefix := indent + strings.Repeat(" ", visibleWidth(bullet))
		itemWidth := width - visibleWidth(firstPrefix)
		if itemWidth < 1 {
			itemWidth = 1
		}

		renderedAny := false
		for c := item.FirstChild(); c != nil; c = c.NextSibling() {
			switch c := c.(type) {
			case *ast.List:
				lines = append(lines, r.renderList(c, source, depth+1, width)...)
				renderedAny = true
			default:
				blockLines := r.renderBlock(c, source, itemWidth, "")
				for _, bl := range blockLines {
					wrapped := wrapRenderLines([]renderLine{bl}, itemWidth)
					for _, w := range wrapped {
						if renderedAny {
							lines = append(lines, rl(contPrefix+w, true))
						} else {
							lines = append(lines, rl(firstPrefix+w, true))
							renderedAny = true
						}
					}
				}
			}
		}
		if !renderedAny {
			lines = append(lines, rl(firstPrefix, true))
		}
	}
	return lines
}

func (r *renderer) renderBlockquote(n *ast.Blockquote, source []byte, width int, nextKind string) []renderLine {
	quoteWidth := width - 2
	if quoteWidth < 1 {
		quoteWidth = 1
	}

	var children []ast.Node
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		children = append(children, c)
	}
	inner := r.renderBlocks(children, source, quoteWidth)
	for len(inner) > 0 && inner[len(inner)-1].text == "" {
		inner = inner[:len(inner)-1]
	}

	var lines []renderLine
	for _, line := range wrapRenderLines(inner, quoteWidth) {
		for _, wrapped := range wrapTextWithANSI(line, quoteWidth) {
			lines = append(lines, rl(r.theme.QuoteBorder("│ ")+r.theme.Quote(wrapped), true))
		}
	}
	if nextKind != "" && nextKind != "space" {
		lines = append(lines, rl("", false))
	}
	return lines
}

func (r *renderer) renderHR(width int, nextKind string) []renderLine {
	lineLen := width
	if lineLen > 80 {
		lineLen = 80
	}
	lines := []renderLine{rl(r.theme.HR(strings.Repeat("─", lineLen)), true)}
	if nextKind != "" && nextKind != "space" {
		lines = append(lines, rl("", false))
	}
	return lines
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func listItemTaskMarker(item *ast.ListItem) string {
	for c := item.FirstChild(); c != nil; c = c.NextSibling() {
		tb, ok := c.(*ast.TextBlock)
		if !ok {
			continue
		}
		for ic := tb.FirstChild(); ic != nil; ic = ic.NextSibling() {
			box, ok := ic.(*ext.TaskCheckBox)
			if !ok {
				return ""
			}
			if box.IsChecked {
				return "[x] "
			}
			return "[ ] "
		}
	}
	return ""
}
