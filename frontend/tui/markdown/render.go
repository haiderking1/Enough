package markdown

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

type renderer struct {
	width int
	theme Theme
	opts  RenderOptions
}

// RenderOptions configures markdown rendering behavior.
type RenderOptions struct {
	OnImageReady func()
}

// Render parses markdown and returns ANSI-styled terminal text wrapped to width.
func Render(input string, width int, theme Theme, opts ...RenderOptions) string {
	var ro RenderOptions
	if len(opts) > 0 {
		ro = opts[0]
	}
	if input == "" {
		return ""
	}
	if width < 10 {
		width = 10
	}

	theme = theme.withDefaults()
	r := &renderer{width: width, theme: theme, opts: ro}

	source := []byte(strings.ReplaceAll(input, "\t", "   "))
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	doc := md.Parser().Parse(text.NewReader(source))

	var blocks []ast.Node
	for c := doc.FirstChild(); c != nil; c = c.NextSibling() {
		blocks = append(blocks, c)
	}

	lines := wrapRenderLines(r.renderBlocks(blocks, source, width), width)
	return strings.Join(lines, "\n")
}
