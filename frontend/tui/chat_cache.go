package tui

import "strings"

type chatRenderCache struct {
	revision      uint64
	width         int
	hideThinking  bool
	toolsExpanded bool
	lines         []string
}

// chatBlockCache holds per-block rendered output so that a change to one
// message (e.g. the streaming assistant bubble) only re-renders that block
// instead of the whole transcript. Slices are aligned by block index; fps[i]
// fingerprints block i's content and render params.
type chatBlockCache struct {
	fps   []uint64
	lines [][]string
	roles []string
}

type footerRenderCache struct {
	revision uint64
	width    int
	lines    []string
}

func (a *App) bumpChat() {
	a.chatRevision++
}

func (a *App) chatLines(width int) []string {
	c := &a.chatCache
	if c.revision == a.chatRevision &&
		c.width == width &&
		c.hideThinking == a.hideThinking &&
		c.toolsExpanded == a.toolsExpanded &&
		c.lines != nil {
		return c.lines
	}

	c.lines = a.renderChatIncremental(width)
	c.revision = a.chatRevision
	c.width = width
	c.hideThinking = a.hideThinking
	c.toolsExpanded = a.toolsExpanded
	return c.lines
}

// renderChatIncremental rebuilds the chat line buffer, reusing per-block cached
// lines for any block whose fingerprint is unchanged. During streaming only the
// last block's fingerprint changes, so only it is re-rendered and re-clamped.
func (a *App) renderChatIncremental(width int) []string {
	if width <= 0 {
		width = 80
	}

	specs := chatBlockSpecs(a.styles, a.messages, width, a.hideThinking, a.toolsExpanded)
	bc := &a.chatBlocks

	n := len(specs)
	newFps := make([]uint64, n)
	newLines := make([][]string, n)
	newRoles := make([]string, n)

	for i, spec := range specs {
		newFps[i] = spec.fp
		newRoles[i] = spec.role
		if i < len(bc.fps) && bc.fps[i] == spec.fp {
			newLines[i] = bc.lines[i]
			continue
		}
		block := spec.render()
		if block == "" {
			newLines[i] = nil
			continue
		}
		newLines[i] = clampSplitLines(strings.Split(block, "\n"), width)
	}

	bc.fps = newFps
	bc.lines = newLines
	bc.roles = newRoles

	// Assemble blocks with separators, matching joinChatBlocks: a blank line
	// between adjacent blocks, except tool blocks hug their neighbours.
	var out []string
	prevRole := ""
	started := false
	for i := 0; i < n; i++ {
		bl := newLines[i]
		if len(bl) == 0 {
			continue
		}
		if started && newRoles[i] != "tool" && prevRole != "tool" {
			out = append(out, "")
		}
		out = append(out, bl...)
		prevRole = newRoles[i]
		started = true
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func (a *App) footerLines(width int) []string {
	c := &a.footerCache
	if c.revision == a.chatRevision && c.width == width && c.lines != nil {
		return c.lines
	}

	footer := a.renderFooter(width)
	c.lines = clampSplitLines(footer, width)
	c.revision = a.chatRevision
	c.width = width
	return c.lines
}
