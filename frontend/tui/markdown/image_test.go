package markdown

import (
	"strings"
	"testing"
)

const testPNGDataURL = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="

func TestIsImageLine(t *testing.T) {
	line := encodeKitty("AAAA", 2, 2, 42, false)
	if !IsImageLine(line) {
		t.Fatalf("expected kitty image line")
	}
	if IsImageLine("plain text") {
		t.Fatalf("expected non-image line")
	}
}

func TestEncodeKitty(t *testing.T) {
	out := encodeKitty("QUJD", 4, 2, 7, false)
	if !strings.Contains(out, "\x1b_G") || !strings.Contains(out, "c=4") || !strings.Contains(out, "C=1") {
		t.Fatalf("unexpected kitty sequence: %q", out)
	}
}

func TestRenderCachedImageKitty(t *testing.T) {
	undoCaps := CapabilitiesForTest(Capabilities{Hyperlinks: true, Images: ImageKitty})
	defer undoCaps()
	ResetImageCache()

	data, err := parseDataURL(testPNGDataURL)
	if err != nil {
		t.Fatal(err)
	}
	primeImageCache(testPNGDataURL, data)

	out := Render("![dot]("+testPNGDataURL+")", 40, Theme{})
	if !IsImageLine(out) && !strings.Contains(out, "\x1b_G") {
		t.Fatalf("expected kitty image output, got %q", stripANSI(out))
	}
}

func TestRenderImagePlaceholderWithoutSupport(t *testing.T) {
	undoCaps := CapabilitiesForTest(Capabilities{Hyperlinks: false, Images: ImageNone})
	defer undoCaps()

	out := Render("![diagram](https://example.com/a.png)", 40, Theme{})
	plain := stripANSI(out)
	if !strings.Contains(plain, "[Image: diagram") {
		t.Fatalf("expected image placeholder, got %q", plain)
	}
}

func TestPNGDimensions(t *testing.T) {
	raw, err := parseDataURL(testPNGDataURL)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := fetchImage(testPNGDataURL)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Dimensions.WidthPx != 1 || decoded.Dimensions.HeightPx != 1 {
		t.Fatalf("expected 1x1, got %+v", decoded.Dimensions)
	}
	_ = raw
}
