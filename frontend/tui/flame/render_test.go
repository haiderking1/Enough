package flame

import "testing"

func TestPrefixEqual(t *testing.T) {
	a := []string{"one", "two", "three"}
	b := []string{"one", "two", "changed"}
	if !prefixEqual(a, a, 2) {
		t.Fatal("expected prefix match for identical slices")
	}
	if prefixEqual(a, b, 3) {
		t.Fatal("expected prefix mismatch at index 2")
	}
}

func TestRenderStablePrefixSkipsScan(t *testing.T) {
	r := &Renderer{
		previousLines: []string{"chat-1", "chat-2", "composer", "footer"},
		previousWidth: 80,
		previousHeight: 24,
	}

	// Simulate composer-only change: thousands of chat lines unchanged.
	prev := make([]string, 0, 1004)
	for i := 0; i < 1000; i++ {
		prev = append(prev, "chat")
	}
	prev = append(prev, "composer v1", "footer")
	r.previousLines = prev

	next := append([]string(nil), prev...)
	next[len(next)-2] = "composer v2"

	diffStart := 0
	if prefixEqual(r.previousLines, next, 1000) {
		diffStart = 1000
	}

	firstChanged := -1
	maxLines := max(len(next), len(r.previousLines))
	for i := diffStart; i < maxLines; i++ {
		if r.previousLines[i] != next[i] {
			firstChanged = i
			break
		}
	}
	if firstChanged != 1000 {
		t.Fatalf("expected first change at 1000, got %d", firstChanged)
	}
}
