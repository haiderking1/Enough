package tui

import (
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestSpawnBulletFixedWidth(t *testing.T) {
	want := runewidth.StringWidth(spawnIdleGlyph)
	for frame := range npmDotSpinner {
		if w := spawnBulletWidth(true, frame*npmSpinnerHoldTicks); w != want {
			t.Fatalf("frame %d width = %d, want %d (glyph %q)", frame, w, want, npmDotSpinner[frame])
		}
	}
	if w := spawnBulletWidth(false, 0); w != want {
		t.Fatalf("idle width = %d, want %d", w, want)
	}
}

func TestSpawnBulletAnimatesWhileRunning(t *testing.T) {
	a := spawnBulletPlain(true, 0)
	b := spawnBulletPlain(true, npmSpinnerHoldTicks)
	if a == b {
		t.Fatalf("frames should differ after one hold period, both %q", a)
	}
	idle := spawnBulletPlain(false, 0)
	if idle != spawnIdleGlyph {
		t.Fatalf("idle glyph = %q, want %q", idle, spawnIdleGlyph)
	}
}

func TestNpmSpinnerCycle(t *testing.T) {
	if len(npmDotSpinner) < 8 {
		t.Fatalf("want full npm dot cycle, got %d frames", len(npmDotSpinner))
	}
}
