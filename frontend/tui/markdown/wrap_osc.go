package markdown

import "strings"

type osc8Link struct {
	url        string
	terminator string // "\x1b\\" or "\x07"
}

func parseOSC8(seq string) (open *osc8Link, close bool) {
	if !strings.HasPrefix(seq, "\x1b]8;") {
		return nil, false
	}
	if strings.HasSuffix(seq, "\x07") {
		body := seq[4 : len(seq)-1]
		parts := strings.SplitN(body, ";", 2)
		if len(parts) != 2 {
			return nil, false
		}
		if parts[1] == "" {
			return nil, true
		}
		return &osc8Link{url: parts[1], terminator: "\x07"}, false
	}
	if strings.HasSuffix(seq, "\x1b\\") {
		body := seq[4 : len(seq)-2]
		parts := strings.SplitN(body, ";", 2)
		if len(parts) != 2 {
			return nil, false
		}
		if parts[1] == "" {
			return nil, true
		}
		return &osc8Link{url: parts[1], terminator: "\x1b\\"}, false
	}
	return nil, false
}

func formatOSC8Open(link osc8Link) string {
	return "\x1b]8;;" + link.url + link.terminator
}

func formatOSC8Close(link osc8Link) string {
	return "\x1b]8;;" + link.terminator
}

type osc8Tracker struct {
	active *osc8Link
}

func (t *osc8Tracker) consume(text string) {
	for i := 0; i < len(text); {
		if text[i] != '\x1b' {
			i++
			continue
		}
		end := strings.Index(text[i:], "m")
		oscEnd := strings.Index(text[i:], "\x1b\\")
		belEnd := strings.Index(text[i:], "\x07")
		seqEnd := len(text)
		if end >= 0 && (oscEnd < 0 || end < oscEnd) && (belEnd < 0 || end < belEnd) {
			i += end + 1
			continue
		}
		candidateEnd := seqEnd
		if oscEnd >= 0 {
			candidateEnd = i + oscEnd + 2
		}
		if belEnd >= 0 && i+belEnd+1 < candidateEnd {
			candidateEnd = i + belEnd + 1
		}
		seq := text[i:candidateEnd]
		if open, close := parseOSC8(seq); open != nil {
			t.active = open
		} else if close {
			t.active = nil
		}
		i = candidateEnd
	}
}

func (t *osc8Tracker) prefix() string {
	if t.active == nil {
		return ""
	}
	return formatOSC8Open(*t.active)
}

func (t *osc8Tracker) lineEndClose() string {
	if t.active == nil {
		return ""
	}
	return formatOSC8Close(*t.active)
}

func wrapTextWithANSI(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	if text == "" {
		return []string{""}
	}

	inputLines := strings.Split(text, "\n")
	var out []string
	tracker := &osc8Tracker{}

	for li, inputLine := range inputLines {
		prefix := ""
		if li > 0 {
			prefix = tracker.prefix()
		}
		for _, wrapped := range wrapSingleLineOSC(prefix+inputLine, width, tracker) {
			out = append(out, wrapped)
		}
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func wrapSingleLineOSC(line string, width int, tracker *osc8Tracker) []string {
	if line == "" {
		return []string{""}
	}
	if visibleWidth(line) <= width {
		tracker.consume(line)
		return []string{line}
	}

	words := splitWordsPreserveANSI(line)
	if len(words) == 0 {
		tracker.consume(line)
		return []string{line}
	}

	var lines []string
	var cur strings.Builder
	curW := 0
	local := &osc8Tracker{}
	if tracker.active != nil {
		link := *tracker.active
		local.active = &link
	}

	flush := func() {
		if cur.Len() == 0 {
			return
		}
		s := cur.String()
		if close := local.lineEndClose(); close != "" {
			s += close
		}
		lines = append(lines, s)
		cur.Reset()
		curW = 0
	}

	for _, word := range words {
		local.consume(word)
		w := visibleWidth(word)
		if curW == 0 {
			if prefix := local.prefix(); prefix != "" {
				cur.WriteString(prefix)
				curW = visibleWidth(prefix)
			}
			if w > width {
				for i, part := range hardSliceANSI(word, width) {
					partLine := part
					if i > 0 {
						if prefix := local.prefix(); prefix != "" {
							partLine = prefix + partLine
						}
					}
					if close := local.lineEndClose(); close != "" {
						partLine += close
					}
					lines = append(lines, partLine)
				}
				continue
			}
			cur.WriteString(word)
			curW += w
			continue
		}
		if curW+1+w > width {
			flush()
			if prefix := local.prefix(); prefix != "" {
				cur.WriteString(prefix)
				curW = visibleWidth(prefix)
			}
			if w > width {
				for i, part := range hardSliceANSI(word, width) {
					partLine := part
					if i > 0 {
						if prefix := local.prefix(); prefix != "" {
							partLine = prefix + partLine
						}
					}
					if close := local.lineEndClose(); close != "" {
						partLine += close
					}
					lines = append(lines, partLine)
				}
				continue
			}
			cur.WriteString(word)
			curW += w
			continue
		}
		cur.WriteByte(' ')
		cur.WriteString(word)
		curW += 1 + w
	}
	flush()
	tracker.consume(line)
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
