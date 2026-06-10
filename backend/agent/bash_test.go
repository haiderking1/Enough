package agent

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/enough/enough/backend/core"
)

func newBashTestAgent(t *testing.T) (*Agent, *eventSink) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("bash not available on windows")
	}
	a := &Agent{workDir: t.TempDir()}
	sink := &eventSink{}
	a.SetEmit(sink.emit)
	return a, sink
}

type eventSink struct {
	mu     sync.Mutex
	chunks []string
}

func (s *eventSink) emit(e core.Event) {
	if e.Kind != core.EventToolDelta {
		return
	}
	ev, ok := e.Data.(core.ToolCallEvent)
	if !ok {
		return
	}
	s.mu.Lock()
	s.chunks = append(s.chunks, ev.Result)
	s.mu.Unlock()
}

func (s *eventSink) joined() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.chunks, "")
}

// Streamed deltas must concatenate exactly to the final result.
func TestBashStreamsLiveOutput(t *testing.T) {
	a, sink := newBashTestAgent(t)

	res := a.toolBash(context.Background(), "call_1", `{"command":"printf 'a\nb\nc\n'"}`)
	if res.isErr {
		t.Fatalf("unexpected error: %s", res.output)
	}
	if res.output != "a\nb\nc\n" {
		t.Fatalf("output = %q", res.output)
	}
	if sink.joined() != res.output {
		t.Fatalf("streamed chunks %q != final %q", sink.joined(), res.output)
	}
	if len(sink.chunks) == 0 {
		t.Fatal("expected at least one streamed delta")
	}
}

// A cancelled context must kill the command promptly rather than waiting for it
// to finish, and report the partial output as interrupted.
func TestBashCancellationKillsCommand(t *testing.T) {
	a, _ := newBashTestAgent(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	res := a.toolBash(ctx, "call_1", `{"command":"echo started; sleep 30"}`)
	elapsed := time.Since(start)

	if elapsed > 5*time.Second {
		t.Fatalf("command was not cancelled promptly: took %s", elapsed)
	}
	if !res.isErr {
		t.Fatal("expected interrupted result to be an error")
	}
	if !strings.Contains(res.output, "[interrupted]") {
		t.Fatalf("expected interrupted marker, got %q", res.output)
	}
	if !strings.Contains(res.output, "started") {
		t.Fatalf("expected partial output before cancel, got %q", res.output)
	}
}

// Output beyond the cap is truncated, and the streamed view stays consistent
// with the final (truncated) result.
func TestBashTruncationConsistent(t *testing.T) {
	a, sink := newBashTestAgent(t)

	res := a.toolBash(context.Background(), "call_1", `{"command":"yes x | head -c 50000"}`)
	if res.isErr {
		t.Fatalf("unexpected error: %s", res.output)
	}
	if !strings.Contains(res.output, "truncated") {
		t.Fatal("expected truncation marker")
	}
	if len(res.output) > maxBashOutput+len(truncMarker)+1 {
		t.Fatalf("output exceeded cap: %d", len(res.output))
	}
	if sink.joined() != res.output {
		t.Fatalf("streamed chunks diverged from final result")
	}
}
