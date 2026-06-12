package markdown

type renderLine struct {
	text   string
	noWrap bool
}

func wrapRenderLines(lines []renderLine, width int) []string {
	var out []string
	for _, line := range lines {
		if line.text == "" {
			out = append(out, "")
			continue
		}
		if line.noWrap || IsImageLine(line.text) {
			out = append(out, line.text)
			continue
		}
		out = append(out, wrapTextWithANSI(line.text, width)...)
	}
	return out
}

func rl(text string, noWrap bool) renderLine {
	return renderLine{text: text, noWrap: noWrap}
}

func joinLines(lines []renderLine) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.text
	}
	return out
}
