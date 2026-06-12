package markdown

import "strings"

func appendEscapeSequence(s string, i int, cur *strings.Builder) int {
	cur.WriteByte(s[i])
	if i+1 >= len(s) {
		return i + 1
	}
	if s[i+1] == ']' {
		j := i + 1
		for j < len(s) {
			cur.WriteByte(s[j])
			if s[j] == '\x07' {
				return j + 1
			}
			if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
				cur.WriteByte(s[j+1])
				return j + 2
			}
			j++
		}
		return len(s)
	}
	j := i + 1
	for j < len(s) {
		cur.WriteByte(s[j])
		if s[j] == 'm' {
			return j + 1
		}
		j++
	}
	return len(s)
}

func hardSliceANSI(s string, width int) []string {
	if width < 1 || s == "" {
		return []string{s}
	}
	var out []string
	var cur strings.Builder
	curW := 0

	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
			curW = 0
		}
	}

	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			start := cur.Len()
			i = appendEscapeSequence(s, i, &cur)
			curW += visibleWidth(cur.String()[start:])
			continue
		}
		if curW+1 > width {
			flush()
		}
		cur.WriteByte(s[i])
		curW++
		i++
	}
	flush()
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

func splitWordsPreserveANSI(text string) []string {
	var words []string
	var cur strings.Builder

	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}

	for i := 0; i < len(text); {
		if text[i] == '\x1b' {
			i = appendEscapeSequence(text, i, &cur)
			continue
		}
		if text[i] == ' ' || text[i] == '\t' {
			flush()
			i++
			continue
		}
		cur.WriteByte(text[i])
		i++
	}
	flush()
	return words
}
