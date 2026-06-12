package markdown

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxImageBytes = 8 << 20

type ImageData struct {
	MIME       string
	Base64     string
	Dimensions ImageDimensions
}

func decodeImageDimensions(raw []byte, mime string) ImageDimensions {
	if dims, ok := dimensionsFromBytes(raw, mime); ok {
		return dims
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err == nil && cfg.Width > 0 && cfg.Height > 0 {
		return ImageDimensions{WidthPx: cfg.Width, HeightPx: cfg.Height}
	}
	return ImageDimensions{WidthPx: 800, HeightPx: 600}
}

func dimensionsFromBytes(raw []byte, mime string) (ImageDimensions, bool) {
	switch mime {
	case "image/png":
		return pngDimensions(raw)
	case "image/jpeg", "image/jpg":
		return jpegDimensions(raw)
	case "image/gif":
		return gifDimensions(raw)
	default:
		return ImageDimensions{}, false
	}
}

func pngDimensions(raw []byte) (ImageDimensions, bool) {
	if len(raw) < 24 || raw[0] != 0x89 || raw[1] != 'P' || raw[2] != 'N' || raw[3] != 'G' {
		return ImageDimensions{}, false
	}
	w := int(raw[16])<<24 | int(raw[17])<<16 | int(raw[18])<<8 | int(raw[19])
	h := int(raw[20])<<24 | int(raw[21])<<16 | int(raw[22])<<8 | int(raw[23])
	if w <= 0 || h <= 0 {
		return ImageDimensions{}, false
	}
	return ImageDimensions{WidthPx: w, HeightPx: h}, true
}

func jpegDimensions(raw []byte) (ImageDimensions, bool) {
	if len(raw) < 2 || raw[0] != 0xff || raw[1] != 0xd8 {
		return ImageDimensions{}, false
	}
	offset := 2
	for offset < len(raw)-9 {
		if raw[offset] != 0xff {
			offset++
			continue
		}
		marker := raw[offset+1]
		if marker >= 0xc0 && marker <= 0xc2 {
			h := int(raw[offset+5])<<8 | int(raw[offset+6])
			w := int(raw[offset+7])<<8 | int(raw[offset+8])
			if w > 0 && h > 0 {
				return ImageDimensions{WidthPx: w, HeightPx: h}, true
			}
			return ImageDimensions{}, false
		}
		if offset+3 >= len(raw) {
			break
		}
		length := int(raw[offset+2])<<8 | int(raw[offset+3])
		if length < 2 {
			break
		}
		offset += 2 + length
	}
	return ImageDimensions{}, false
}

func gifDimensions(raw []byte) (ImageDimensions, bool) {
	if len(raw) < 10 {
		return ImageDimensions{}, false
	}
	sig := string(raw[:6])
	if sig != "GIF87a" && sig != "GIF89a" {
		return ImageDimensions{}, false
	}
	w := int(raw[6]) | int(raw[7])<<8
	h := int(raw[8]) | int(raw[9])<<8
	if w <= 0 || h <= 0 {
		return ImageDimensions{}, false
	}
	return ImageDimensions{WidthPx: w, HeightPx: h}, true
}

func encodeBase64(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

func fetchImage(rawURL string) (*ImageData, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("empty image url")
	}
	if strings.HasPrefix(rawURL, "data:") {
		return parseDataURL(rawURL)
	}
	if strings.HasPrefix(rawURL, "file://") {
		return loadImageFile(rawURL)
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return nil, fmt.Errorf("unsupported image url scheme")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("image fetch failed: %s", resp.Status)
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxImageBytes {
		return nil, fmt.Errorf("image too large")
	}

	mime := resp.Header.Get("Content-Type")
	if semi := strings.Index(mime, ";"); semi >= 0 {
		mime = mime[:semi]
	}
	mime = strings.TrimSpace(mime)
	if mime == "" {
		mime = sniffImageMIME(raw)
	}
	if !strings.HasPrefix(mime, "image/") {
		return nil, fmt.Errorf("unsupported content type %q", mime)
	}

	dims := decodeImageDimensions(raw, mime)
	return &ImageData{
		MIME:       mime,
		Base64:     encodeBase64(raw),
		Dimensions: dims,
	}, nil
}

func parseDataURL(rawURL string) (*ImageData, error) {
	comma := strings.Index(rawURL, ",")
	if comma < 0 {
		return nil, fmt.Errorf("invalid data url")
	}
	header := rawURL[5:comma]
	payload := rawURL[comma+1:]
	var raw []byte
	var err error
	if strings.HasSuffix(header, ";base64") {
		raw, err = base64.StdEncoding.DecodeString(payload)
	} else {
		decoded, decErr := url.PathUnescape(payload)
		if decErr != nil {
			return nil, decErr
		}
		raw = []byte(decoded)
	}
	if err != nil {
		return nil, err
	}
	mime := strings.TrimPrefix(header, ";base64")
	mime = strings.TrimSpace(mime)
	if mime == "" {
		mime = sniffImageMIME(raw)
	}
	dims := decodeImageDimensions(raw, mime)
	return &ImageData{MIME: mime, Base64: encodeBase64(raw), Dimensions: dims}, nil
}

func loadImageFile(rawURL string) (*ImageData, error) {
	path := strings.TrimPrefix(rawURL, "file://")
	if path == "" {
		return nil, fmt.Errorf("empty file url")
	}
	if u, err := url.Parse(rawURL); err == nil && u.Path != "" {
		path = u.Path
	}
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxImageBytes {
		return nil, fmt.Errorf("image too large")
	}
	mime := sniffImageMIME(raw)
	dims := decodeImageDimensions(raw, mime)
	return &ImageData{MIME: mime, Base64: encodeBase64(raw), Dimensions: dims}, nil
}

func sniffImageMIME(raw []byte) string {
	if len(raw) >= 4 && raw[0] == 0x89 && raw[1] == 'P' {
		return "image/png"
	}
	if len(raw) >= 2 && raw[0] == 0xff && raw[1] == 0xd8 {
		return "image/jpeg"
	}
	if len(raw) >= 6 && (string(raw[:6]) == "GIF89a" || string(raw[:6]) == "GIF87a") {
		return "image/gif"
	}
	return "application/octet-stream"
}
