package tui

import (
	"bytes"
	"unicode/utf8"
)

type keyAction int

const (
	keyNone keyAction = iota
	keyEnter
	keyBackspace
	keyDelete
	keyLeft
	keyRight
	keyUp
	keyDown
	keyTab
	keyEscape
	keyCtrlC
	keyCtrlD
	keyHome
	keyEnd
	keyRune
	keyPaste
)

type parsedKey struct {
	action keyAction
	r      rune
	paste  string
}

type keyReader struct {
	buf []byte
}

func newKeyReader() *keyReader {
	return &keyReader{}
}

func (kr *keyReader) feed(data []byte) []parsedKey {
	kr.buf = append(kr.buf, data...)
	var keys []parsedKey

	for len(kr.buf) > 0 {
		k, n := kr.parseOne()
		if n == 0 {
			break
		}
		kr.buf = kr.buf[n:]
		if k.action != keyNone {
			keys = append(keys, k)
		}
	}

	return keys
}

func (kr *keyReader) parseOne() (parsedKey, int) {
	b := kr.buf
	if len(b) == 0 {
		return parsedKey{}, 0
	}

	if len(b) >= 6 && bytes.HasPrefix(b, []byte("\x1b[200~")) {
		end := bytes.Index(b, []byte("\x1b[201~"))
		if end == -1 {
			return parsedKey{}, 0
		}
		paste := string(b[6:end])
		return parsedKey{action: keyPaste, paste: paste}, end + 6
	}

	switch b[0] {
	case 3:
		return parsedKey{action: keyCtrlC}, 1
	case 4:
		return parsedKey{action: keyCtrlD}, 1
	case '\r', '\n':
		return parsedKey{action: keyEnter}, 1
	case 127, 8:
		return parsedKey{action: keyBackspace}, 1
	case '\t':
		return parsedKey{action: keyTab}, 1
	case 27:
		if len(b) < 2 {
			return parsedKey{}, 0
		}
		if b[1] == '[' {
			if len(b) < 3 {
				return parsedKey{}, 0
			}
			switch b[2] {
			case 'A':
				return parsedKey{action: keyUp}, 3
			case 'B':
				return parsedKey{action: keyDown}, 3
			case 'C':
				return parsedKey{action: keyRight}, 3
			case 'D':
				return parsedKey{action: keyLeft}, 3
			case 'H':
				return parsedKey{action: keyHome}, 3
			case 'F':
				return parsedKey{action: keyEnd}, 3
			case '3':
				if len(b) >= 4 && b[3] == '~' {
					return parsedKey{action: keyDelete}, 4
				}
			}
			if len(b) >= 4 && b[3] == '~' {
				switch b[2] {
				case '1':
					return parsedKey{action: keyHome}, 4
				case '4':
					return parsedKey{action: keyDelete}, 4
				}
			}
		}
		if b[1] == 'O' && len(b) >= 3 {
			switch b[2] {
			case 'H':
				return parsedKey{action: keyHome}, 3
			case 'F':
				return parsedKey{action: keyEnd}, 3
			}
		}
		return parsedKey{action: keyEscape}, 1
	}

	r, size := utf8.DecodeRune(b)
	if r == utf8.RuneError && size == 1 {
		return parsedKey{}, 1
	}
	return parsedKey{action: keyRune, r: r}, size
}
