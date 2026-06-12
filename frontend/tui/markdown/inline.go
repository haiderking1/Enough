package markdown

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	ext "github.com/yuin/goldmark/extension/ast"
)

type inlineStyle struct {
	apply func(string) string
}

func (r *renderer) defaultInlineStyle() inlineStyle {
	return inlineStyle{apply: r.theme.Plain}
}

func (r *renderer) renderInlineNode(n ast.Node, source []byte, style inlineStyle) string {
	switch n := n.(type) {
	case *ast.Text:
		text := style.apply(string(n.Segment.Value(source)))
		if n.HardLineBreak() {
			text += "\n"
		} else if n.SoftLineBreak() {
			text += " "
		}
		return text
	case *ast.String:
		return style.apply(string(n.Value))
	case *ast.CodeSpan:
		return r.theme.Code(string(n.Text(source)))
	case *ast.Emphasis:
		inner := r.renderInlineChildren(n, source, style)
		if n.Level >= 2 {
			return r.theme.Bold(inner)
		}
		return r.theme.Italic(inner)
	case *ext.Strikethrough:
		inner := r.renderInlineChildren(n, source, style)
		return r.theme.Strikethrough(inner)
	case *ext.TaskCheckBox:
		return ""
	case *ast.Image:
		alt := r.renderInlineChildren(n, source, style)
		dest := string(n.Destination)
		label := r.theme.Image(imageFallback(alt, dest))
		if currentCapabilities().Hyperlinks && dest != "" {
			return Hyperlink(label, dest)
		}
		return label
	case *ast.Link:
		text := r.renderInlineChildren(n, source, style)
		styled := r.theme.Link(text)
		href := string(n.Destination)
		if href == "" {
			return styled
		}
		if currentCapabilities().Hyperlinks {
			return Hyperlink(styled, href)
		}
		hrefForCompare := href
		if strings.HasPrefix(href, "mailto:") {
			hrefForCompare = strings.TrimPrefix(href, "mailto:")
		}
		if text == href || text == hrefForCompare {
			return styled
		}
		return styled + r.theme.LinkURL(" ("+href+")")
	case *ast.AutoLink:
		label := string(n.Label(source))
		return r.theme.Link(label)
	case *ast.RawHTML:
		return style.apply(string(n.Segments.Value(source)))
	default:
		return r.renderInlineChildren(n, source, style)
	}
}

func (r *renderer) renderInlineChildren(n ast.Node, source []byte, style inlineStyle) string {
	var out string
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		out += r.renderInlineNode(child, source, style)
	}
	return out
}

func (r *renderer) renderInline(n ast.Node, source []byte) string {
	return r.renderInlineChildren(n, source, r.defaultInlineStyle())
}

func (r *renderer) renderBlockText(n ast.Node, source []byte) string {
	switch n := n.(type) {
	case *ast.Paragraph, *ast.TextBlock:
		return r.renderInline(n, source)
	case *ast.Text:
		return string(n.Segment.Value(source))
	default:
		var parts []string
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t := r.renderBlockText(c, source); t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, " ")
	}
}
