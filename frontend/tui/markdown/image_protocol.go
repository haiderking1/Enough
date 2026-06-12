package markdown

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
)

type ImageProtocol string

const (
	ImageKitty   ImageProtocol = "kitty"
	ImageITerm2  ImageProtocol = "iterm2"
	ImageNone    ImageProtocol = ""
)

const (
	kittyPrefix  = "\x1b_G"
	iterm2Prefix = "\x1b]1337;File="
)

// CellDimensions holds terminal cell size in pixels.
type CellDimensions struct {
	WidthPx  int
	HeightPx int
}

var cellDimensions = CellDimensions{WidthPx: 9, HeightPx: 18}

// SetCellDimensions updates the assumed terminal cell size for image scaling.
func SetCellDimensions(d CellDimensions) {
	if d.WidthPx > 0 && d.HeightPx > 0 {
		cellDimensions = d
	}
}

// IsImageLine reports whether a rendered line contains inline terminal image data.
func IsImageLine(line string) bool {
	if strings.HasPrefix(line, kittyPrefix) || strings.HasPrefix(line, iterm2Prefix) {
		return true
	}
	return strings.Contains(line, kittyPrefix) || strings.Contains(line, iterm2Prefix)
}

type ImageDimensions struct {
	WidthPx  int
	HeightPx int
}

type imageCellSize struct {
	Columns int
	Rows    int
}

func allocateImageID() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 1
	}
	id := binary.BigEndian.Uint32(b[:])
	if id == 0 {
		return 1
	}
	return id
}

func encodeKitty(base64Data string, columns, rows int, imageID uint32, moveCursor bool) string {
	const chunkSize = 4096
	params := []string{"a=T", "f=100", "q=2"}
	if !moveCursor {
		params = append(params, "C=1")
	}
	if columns > 0 {
		params = append(params, fmt.Sprintf("c=%d", columns))
	}
	if rows > 0 {
		params = append(params, fmt.Sprintf("r=%d", rows))
	}
	if imageID > 0 {
		params = append(params, fmt.Sprintf("i=%d", imageID))
	}

	if len(base64Data) <= chunkSize {
		return kittyPrefix + strings.Join(params, ",") + ";" + base64Data + "\x1b\\"
	}

	var chunks []string
	offset := 0
	first := true
	for offset < len(base64Data) {
		end := offset + chunkSize
		if end > len(base64Data) {
			end = len(base64Data)
		}
		chunk := base64Data[offset:end]
		last := end >= len(base64Data)
		switch {
		case first:
			chunks = append(chunks, kittyPrefix+strings.Join(params, ",")+",m=1;"+chunk+"\x1b\\")
			first = false
		case last:
			chunks = append(chunks, kittyPrefix+"m=0;"+chunk+"\x1b\\")
		default:
			chunks = append(chunks, kittyPrefix+"m=1;"+chunk+"\x1b\\")
		}
		offset = end
	}
	return strings.Join(chunks, "")
}

func encodeITerm2(base64Data string, widthCells int, name string) string {
	params := []string{"inline=1", fmt.Sprintf("width=%d", widthCells), "height=auto", "preserveAspectRatio=1"}
	if name != "" {
		params = append(params, "name="+base64EncodeString(name))
	}
	return iterm2Prefix + strings.Join(params, ";") + ":" + base64Data + "\x07"
}

func calculateImageCellSize(dim ImageDimensions, maxWidthCells int, maxHeightCells *int) imageCellSize {
	maxWidth := max(1, maxWidthCells)
	cellW := max(1, cellDimensions.WidthPx)
	cellH := max(1, cellDimensions.HeightPx)
	imageW := max(1, dim.WidthPx)
	imageH := max(1, dim.HeightPx)

	widthScale := float64(maxWidth*cellW) / float64(imageW)
	heightScale := widthScale
	if maxHeightCells != nil {
		maxHeight := max(1, *maxHeightCells)
		heightScale = float64(maxHeight*cellH) / float64(imageH)
	}
	scale := min(widthScale, heightScale)

	columns := int(float64(imageW)*scale/float64(cellW) + 0.999999)
	rows := int(float64(imageH)*scale/float64(cellH) + 0.999999)
	if columns < 1 {
		columns = 1
	}
	if rows < 1 {
		rows = 1
	}
	if columns > maxWidth {
		columns = maxWidth
	}
	if maxHeightCells != nil && rows > *maxHeightCells {
		rows = *maxHeightCells
	}
	return imageCellSize{Columns: columns, Rows: rows}
}

func renderImageSequence(base64Data string, mime string, dims ImageDimensions, maxWidthCells int, alt string) []string {
	caps := currentCapabilities()
	if caps.Images == ImageNone {
		return nil
	}

	maxHeight := max(1, (maxWidthCells*cellDimensions.WidthPx)/max(1, cellDimensions.HeightPx))
	size := calculateImageCellSize(dims, maxWidthCells, &maxHeight)

	switch caps.Images {
	case ImageKitty:
		seq := encodeKitty(base64Data, size.Columns, size.Rows, allocateImageID(), false)
		lines := []string{seq}
		for i := 1; i < size.Rows; i++ {
			lines = append(lines, "")
		}
		return lines
	case ImageITerm2:
		seq := encodeITerm2(base64Data, size.Columns, alt)
		lines := make([]string, size.Rows)
		for i := 0; i < size.Rows-1; i++ {
			lines[i] = ""
		}
		moveUp := ""
		if size.Rows > 1 {
			moveUp = fmt.Sprintf("\x1b[%dA", size.Rows-1)
		}
		lines[size.Rows-1] = moveUp + seq
		return lines
	default:
		return nil
	}
}

func imageFallbackLabel(mime string, dims *ImageDimensions, alt, url string) string {
	if mime != "" || dims != nil {
		parts := []string{}
		if alt != "" {
			parts = append(parts, alt)
		}
		if mime != "" {
			parts = append(parts, "["+mime+"]")
		}
		if dims != nil {
			parts = append(parts, fmt.Sprintf("%dx%d", dims.WidthPx, dims.HeightPx))
		}
		if len(parts) > 0 {
			return "[Image: " + strings.Join(parts, " ") + "]"
		}
	}
	return imageFallback(alt, url)
}

func base64EncodeString(s string) string {
	return encodeBase64([]byte(s))
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
