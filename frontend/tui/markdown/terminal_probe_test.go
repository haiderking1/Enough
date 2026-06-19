package markdown

import "testing"

func TestHandleTerminalResponseCellSize(t *testing.T) {
	seq := []byte("\x1b[6;39;18t")
	if !HandleTerminalResponse(seq) {
		t.Fatal("expected cell size report to be consumed")
	}
	if got := GetCellDimensions(); got.WidthPx != 18 || got.HeightPx != 39 {
		t.Fatalf("unexpected cell dims: %+v", got)
	}
	if HandleTerminalResponse([]byte("\x1b[A")) {
		t.Fatal("expected key sequence to pass through")
	}
}

func TestHandleTerminalResponseDeviceAttributes(t *testing.T) {
	for _, seq := range [][]byte{
		[]byte("\x1b[?6c"),
		[]byte("\x1b[?62c"),
	} {
		if !HandleTerminalResponse(seq) {
			t.Fatalf("expected device attributes reply to be consumed: %q", seq)
		}
	}
}

func TestHandleTerminalResponseKittyProbe(t *testing.T) {
	seq := []byte("\x1b_Gi=31;OK\x1b\\")
	if !HandleTerminalResponse(seq) {
		t.Fatal("expected kitty graphics probe reply to be consumed")
	}
}
