package markdown

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// Foot identifies itself in tertiary DA as "FOOT" (hex 464f4f54).
// See https://codeberg.org/dnkl/foot#programmatically-checking-if-running-in-foot
const footTertiaryDA = "464f4f54"

// Kitty graphics query response token when the terminal supports the protocol.
// See https://sw.kovidgoyal.net/kitty/graphics-protocol/
const kittyGraphicsOK = "_Gi=31;OK"

// imageProbeQuery asks for Foot tertiary DA, Kitty graphics support, then DA1.
// Only opt-in via ENOUGH_PROBE_TERMINAL_IMAGES — replies leak on many terminals.
const imageProbeQuery = "\033P!|?\033\\\033_Gi=31,s=1,v=1,a=q,t=d,f=24;AAAA\033\\\033[c"

var (
	capsLocked         bool
	cellSizeReportRe     = regexp.MustCompile(`\x1b\[6;(\d+);(\d+)t`)
	deviceAttributesRe = regexp.MustCompile(`\x1b\[\?[0-9;]*c`)
)

// InitTerminalCapabilities locks env-based terminal feature detection. It does
// not send interactive escape probes by default — those replies show up as
// garbage on the shell prompt in Warp, Alacritty, and others.
func InitTerminalCapabilities(fd int) {
	if capsLocked || !term.IsTerminal(fd) {
		return
	}
	capsLocked = true

	base := detectCapabilities()
	if base.Images == ImageNone && probeTerminalImagesEnabled() {
		if proto := probeImageProtocol(fd); proto != ImageNone {
			base.Images = proto
			base.TrueColor = true
			base.Hyperlinks = true
		}
	}

	lockCapabilities(base)
}

func probeTerminalImagesEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ENOUGH_PROBE_TERMINAL_IMAGES"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func lockCapabilities(c Capabilities) {
	locked := c
	capabilitiesFn = func() Capabilities { return locked }
}

func probeImageProtocol(fd int) ImageProtocol {
	if _, err := os.Stdout.WriteString(imageProbeQuery); err != nil {
		return ImageNone
	}

	tty := os.NewFile(uintptr(fd), "tty")
	if tty == nil {
		return ImageNone
	}
	defer tty.Close()

	deadline := time.Now().Add(250 * time.Millisecond)
	buf := make([]byte, 256)
	var out []byte
	var detected ImageProtocol
	for time.Now().Before(deadline) {
		_ = tty.SetReadDeadline(deadline)
		n, err := tty.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
			switch {
			case bytes.Contains(out, []byte(footTertiaryDA)):
				detected = ImageSixel
			case bytes.Contains(out, []byte("P>|foot(")):
				detected = ImageSixel
			case bytes.Contains(out, []byte(kittyGraphicsOK)):
				detected = ImageKitty
			}
		}
		if err != nil {
			break
		}
	}
	drainTerminalResponses(tty, 200*time.Millisecond)
	return detected
}

func drainTerminalResponses(tty *os.File, totalTimeout time.Duration) {
	if tty == nil {
		return
	}
	deadline := time.Now().Add(totalTimeout)
	idleUntil := time.Now().Add(60 * time.Millisecond)
	buf := make([]byte, 256)
	for time.Now().Before(deadline) {
		wait := time.Until(idleUntil)
		if wait <= 0 {
			return
		}
		if wait > 50*time.Millisecond {
			wait = 50 * time.Millisecond
		}
		_ = tty.SetReadDeadline(time.Now().Add(wait))
		n, err := tty.Read(buf)
		if n > 0 {
			idleUntil = time.Now().Add(60 * time.Millisecond)
			continue
		}
		var netErr interface{ Timeout() bool }
		if errors.As(err, &netErr) && netErr.Timeout() {
			if time.Now().After(idleUntil) {
				return
			}
			continue
		}
		return
	}
}

// QueryCellDimensions is intentionally a no-op. CSI 16 t replies leak to the
// shell on exit; cell size comes from TIOCGWINSZ (see term.CellPixels) or the
// built-in default in image_protocol.go.
func QueryCellDimensions(w interface{ Write(string) }) {}

// HandleTerminalResponse consumes terminal query replies that must not reach
// the key handler (e.g. cell-size reports). Returns true when seq was handled.
func HandleTerminalResponse(seq []byte) bool {
	if dims := parseCellSizeReport(seq); dims != nil {
		SetCellDimensions(*dims)
		return true
	}
	if deviceAttributesRe.Match(seq) {
		return true
	}
	if bytes.Contains(seq, []byte(kittyGraphicsOK)) {
		return true
	}
	if isTerminalControlResponse(seq) {
		return true
	}
	return false
}

func isTerminalControlResponse(seq []byte) bool {
	if len(seq) < 3 || seq[0] != 0x1b {
		return false
	}
	switch seq[1] {
	case '_', 'P':
		return bytes.HasSuffix(seq, []byte("\x1b\\"))
	default:
		return false
	}
}

func parseCellSizeReport(data []byte) *CellDimensions {
	m := cellSizeReportRe.FindSubmatch(data)
	if len(m) != 3 || len(m[0]) != len(data) {
		return nil
	}
	heightPx, err1 := strconv.Atoi(string(m[1]))
	widthPx, err2 := strconv.Atoi(string(m[2]))
	if err1 != nil || err2 != nil || heightPx <= 0 || widthPx <= 0 {
		return nil
	}
	return &CellDimensions{WidthPx: widthPx, HeightPx: heightPx}
}
