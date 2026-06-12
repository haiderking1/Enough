package agent

import "testing"

func TestUserMessageSignalsProfileCorrection(t *testing.T) {
	yes := []string{
		"bro my name is haider lol with lowercase h not my name is h",
		"wym hey h lol",
		"my name is haider, not h",
		"stop calling me h",
		"you got my name wrong",
	}
	for _, msg := range yes {
		if !userMessageSignalsProfileCorrection(msg) {
			t.Fatalf("expected correction signal for %q", msg)
		}
	}

	no := []string{
		"hmmmmmmmmmmmmmmmmmm",
		"what's the weather",
		"fix the diff bug in the tui",
	}
	for _, msg := range no {
		if userMessageSignalsProfileCorrection(msg) {
			t.Fatalf("unexpected correction signal for %q", msg)
		}
	}
}
