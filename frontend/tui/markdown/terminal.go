package markdown

import (
	"os"
	"strings"
)

// Capabilities describes optional terminal features for markdown rendering.
type Capabilities struct {
	Hyperlinks bool
	Images     ImageProtocol
}

var capabilitiesFn = detectCapabilities

func CapabilitiesForTest(c Capabilities) func() {
	prev := capabilitiesFn
	capabilitiesFn = func() Capabilities { return c }
	return func() { capabilitiesFn = prev }
}

func currentCapabilities() Capabilities {
	return capabilitiesFn()
}

func detectCapabilities() Capabilities {
	termProgram := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	term := strings.ToLower(os.Getenv("TERM"))
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))

	inTmuxOrScreen := os.Getenv("TMUX") != "" || strings.HasPrefix(term, "tmux") || strings.HasPrefix(term, "screen")
	if inTmuxOrScreen {
		return Capabilities{Hyperlinks: false, Images: ImageNone}
	}

	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "" || termProgram == "kitty":
		return Capabilities{Hyperlinks: true, Images: ImageKitty}
	case termProgram == "ghostty" || strings.Contains(term, "ghostty") || os.Getenv("GHOSTTY_RESOURCES_DIR") != "":
		return Capabilities{Hyperlinks: true, Images: ImageKitty}
	case os.Getenv("WEZTERM_PANE") != "" || termProgram == "wezterm":
		return Capabilities{Hyperlinks: true, Images: ImageKitty}
	case os.Getenv("ITERM_SESSION_ID") != "" || termProgram == "iterm.app":
		return Capabilities{Hyperlinks: true, Images: ImageITerm2}
	case termProgram == "vscode":
		return Capabilities{Hyperlinks: true, Images: ImageNone}
	case termProgram == "alacritty":
		return Capabilities{Hyperlinks: true, Images: ImageNone}
	case colorTerm == "truecolor" || colorTerm == "24bit":
		return Capabilities{Hyperlinks: true, Images: ImageNone}
	default:
		return Capabilities{Hyperlinks: false, Images: ImageNone}
	}
}

// Hyperlink wraps visible text in an OSC 8 hyperlink sequence.
func Hyperlink(text, url string) string {
	if text == "" || url == "" {
		return text
	}
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

func imageFallback(alt, url string) string {
	alt = strings.TrimSpace(alt)
	url = strings.TrimSpace(url)
	switch {
	case alt != "" && url != "":
		return "[Image: " + alt + " (" + url + ")]"
	case alt != "":
		return "[Image: " + alt + "]"
	case url != "":
		return "[Image: " + url + "]"
	default:
		return "[Image]"
	}
}
