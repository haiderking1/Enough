package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/opencode"
)

func TestToolReadFileReportsLineCount(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		content   string
		wantLines int
	}{
		"trailing newline":    {"a\nb\nc\n", 3},
		"no trailing newline": {"a\nb\nc", 3},
		"single line":         {"only", 1},
		"empty":               {"", 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".txt")
			mustWrite(t, path, tc.content)
			a := &Agent{workDir: dir}
			res := a.toolReadFile(`{"path":"` + path + `"}`)
			if res.isErr {
				t.Fatalf("read_file error: %s", res.output)
			}
			wantHeader := "Read " + strconv.Itoa(tc.wantLines) + " lines from "
			if !strings.HasPrefix(res.output, wantHeader) {
				t.Fatalf("expected header %q, got %q", wantHeader, res.output)
			}
		})
	}
}

func TestWorkerToolsIncludeCodingTools(t *testing.T) {
	names := toolNames(workerTools(0))
	for _, want := range []string{"read_file", "write_file", "edit_file", "bash", "web_search", "agent_swarm"} {
		if !names[want] {
			t.Fatalf("worker at depth 0 missing %q", want)
		}
	}
}

func TestWorkerToolsCapNestingAtMaxDepth(t *testing.T) {
	names := toolNames(workerTools(maxSwarmDepth))
	if names["agent_swarm"] {
		t.Fatal("worker at max depth should not get agent_swarm")
	}
	for _, want := range []string{"read_file", "bash", "write_file"} {
		if !names[want] {
			t.Fatalf("worker at max depth missing %q", want)
		}
	}
}

func TestAgentSwarmToolRegistration(t *testing.T) {
	if !hasTool(nativeTools(), "agent_swarm") {
		t.Fatal("main agent is missing agent_swarm")
	}
	if hasTool(nativeTools(), "agent") {
		t.Fatal("main agent should not expose legacy agent tool")
	}
}

func TestParsePlannedSwarmTasks(t *testing.T) {
	raw := `[{"id":"a","prompt":"do A"},{"id":"b","prompt":"do B","depends_on":["a"]}]`
	tasks := parsePlannedSwarmTasks(raw)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}
	if tasks[0].ID != "a" || tasks[1].DependsOn[0] != "a" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
}

func TestSwarmTaskIDDefaults(t *testing.T) {
	if got := swarmTaskID(swarmTask{}, 0); got != "agent-1" {
		t.Fatalf("expected agent-1, got %q", got)
	}
	if got := swarmTaskID(swarmTask{ID: "scout"}, 0); got != "scout" {
		t.Fatalf("expected scout, got %q", got)
	}
}

func TestAggregateSwarmOutput(t *testing.T) {
	out := aggregateSwarmOutput([]swarmWorkerResult{
		{ID: "a", Status: "ok", Output: "done", Turns: 2},
		{ID: "b", Status: "error", Error: "boom", Turns: 1},
	}, 2, "")
	if !strings.Contains(out, "1 ok") || !strings.Contains(out, "1 error") {
		t.Fatalf("unexpected header: %q", out)
	}
	if !strings.Contains(out, "## a [ok]") || !strings.Contains(out, "## b [error]") {
		t.Fatalf("missing worker sections: %q", out)
	}
}

func TestToolGlobAndGrep(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "main.go"), "package main\nfunc Logger() {}\n")
	mustWrite(t, filepath.Join(dir, "sub", "util.go"), "package sub\nvar loggerName = 1\n")
	mustWrite(t, filepath.Join(dir, "readme.md"), "docs\n")

	a := &Agent{workDir: dir}

	glob := a.toolGlob(`{"pattern":"**/*.go"}`)
	if glob.isErr {
		t.Fatalf("glob error: %s", glob.output)
	}
	if !strings.Contains(glob.output, "main.go") || !strings.Contains(glob.output, filepath.ToSlash("sub/util.go")) {
		t.Fatalf("glob did not find go files: %q", glob.output)
	}

	grep := a.toolGrep(`{"pattern":"(?i)logger","include":"*.go"}`)
	if grep.isErr {
		t.Fatalf("grep error: %s", grep.output)
	}
	if !strings.Contains(grep.output, "main.go:2") || !strings.Contains(grep.output, "util.go:2") {
		t.Fatalf("grep missed matches: %q", grep.output)
	}
}

func TestResolveWorkerOutput(t *testing.T) {
	cases := map[string]struct {
		finalText string
		swarmOut  string
		want      string
	}{
		"final text wins":          {"summary here", "PAYLOAD:deep", "summary here"},
		"empty falls back to swarm": {"   ", "PAYLOAD:deep", "PAYLOAD:deep"},
		"both empty":                {"", "", ""},
		"final trimmed":             {"  done  ", "PAYLOAD:deep", "done"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := resolveWorkerOutput(tc.finalText, tc.swarmOut); got != tc.want {
				t.Fatalf("resolveWorkerOutput(%q, %q) = %q, want %q", tc.finalText, tc.swarmOut, got, tc.want)
			}
		})
	}
}

// TestNestedSwarmPropagatesDeepPayload drives a real (mocked-transport) worker
// loop through several levels of nested agent_swarm. Each intermediate worker
// spawns one child and then finishes with EMPTY text, simulating a model that
// forgets to echo its child's result. The innermost worker returns a known
// payload; the test asserts that payload survives the trip all the way back to
// the outermost aggregated output.
func TestLinkedSwarmContextRespectsParentAbort(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	ctx, cancel := linkedSwarmContext(parent)
	defer cancel()

	parentCancel()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("swarm context did not cancel when parent aborted")
	}
}

func TestNestedSwarmPropagatesDeepPayloadStress(t *testing.T) {
	for i := 0; i < 20; i++ {
		t.Run(fmt.Sprintf("run-%d", i), func(t *testing.T) {
			testNestedSwarmPropagatesDeepPayload(t)
		})
	}
}

func TestNestedSwarmPropagatesDeepPayload(t *testing.T) {
	testNestedSwarmPropagatesDeepPayload(t)
}

func testNestedSwarmPropagatesDeepPayload(t *testing.T) {
	t.Helper()
	const payload = "PAYLOAD:level-deep"
	depthRe := regexp.MustCompile(`DEPTH=(\d+)`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req opencode.ChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		last := req.Messages[len(req.Messages)-1]

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		writeChunk := func(v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		}

		// A tool result just came back: finish with empty text. This exercises
		// the resolveWorkerOutput fallback to the nested swarm output.
		if last.Role == "tool" {
			writeChunk(streamChunkJSON("", nil))
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		depth := 0
		if m := depthRe.FindStringSubmatch(opencode.ContentString(last)); len(m) == 2 {
			depth, _ = strconv.Atoi(m[1])
		}

		if depth <= 0 {
			// Innermost worker: hand back the payload directly.
			writeChunk(streamChunkJSON(payload, nil))
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		// Spawn exactly one child one level shallower via a nested swarm.
		args := fmt.Sprintf(`{"tasks":[{"id":"child","prompt":"DEPTH=%d"}]}`, depth-1)
		writeChunk(streamChunkJSON("", []toolCallJSON{{
			Index:    0,
			ID:       "call-swarm",
			Type:     "function",
			Function: toolFnJSON{Name: "agent_swarm", Arguments: args},
		}}))
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	a := &Agent{
		cfg:     config.Runtime{Model: "test-model"},
		client:  opencode.NewClient(srv.URL, "test-key", "test-model"),
		workDir: t.TempDir(),
	}

	// depth=3 is the deepest a worker chain can nest under maxSwarmDepth=3:
	// workers at swarmDepth 0,1,2 each spawn one child, the swarmDepth-3 worker
	// returns the payload. This matches the user-observed "5 levels" (main +
	// four worker levels).
	args := `{"tasks":[{"id":"root","prompt":"DEPTH=3"}]}`
	res := a.toolAgentSwarm(context.Background(), "outer", args, 0)

	if res.isErr {
		t.Fatalf("outer swarm errored: %s", res.output)
	}
	if !strings.Contains(res.output, payload) {
		t.Fatalf("deep payload %q did not propagate to outer output:\n%s", payload, res.output)
	}
}

// TestNestedSwarmPropagatesWhenModelReturnsStub verifies that even when the
// model would reply with a useless summary after a nested swarm (instead of
// echoing the payload), the short-circuit path still returns the child output.
func TestNestedSwarmPropagatesWhenModelReturnsStub(t *testing.T) {
	const payload = "PAYLOAD:stub-test"
	depthRe := regexp.MustCompile(`DEPTH=(\d+)`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req opencode.ChatRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		last := req.Messages[len(req.Messages)-1]

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		writeChunk := func(v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "data: %s\n\n", b)
			if flusher != nil {
				flusher.Flush()
			}
		}

		if last.Role == "tool" {
			// Bad model: summarizes instead of echoing nested output.
			writeChunk(streamChunkJSON("Task complete.", nil))
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		depth := 0
		if m := depthRe.FindStringSubmatch(opencode.ContentString(last)); len(m) == 2 {
			depth, _ = strconv.Atoi(m[1])
		}
		if depth <= 0 {
			writeChunk(streamChunkJSON(payload, nil))
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}

		args := fmt.Sprintf(`{"tasks":[{"id":"child","prompt":"DEPTH=%d"}]}`, depth-1)
		writeChunk(streamChunkJSON("", []toolCallJSON{{
			Index:    0,
			ID:       "call-swarm",
			Type:     "function",
			Function: toolFnJSON{Name: "agent_swarm", Arguments: args},
		}}))
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	a := &Agent{
		cfg:     config.Runtime{Model: "test-model"},
		client:  opencode.NewClient(srv.URL, "test-key", "test-model"),
		workDir: t.TempDir(),
	}

	args := `{"tasks":[{"id":"root","prompt":"DEPTH=2"}]}`
	res := a.toolAgentSwarm(context.Background(), "outer", args, 0)
	if res.isErr {
		t.Fatalf("outer swarm errored: %s", res.output)
	}
	if !strings.Contains(res.output, payload) {
		t.Fatalf("payload lost when model returns stub summary:\n%s", res.output)
	}
}

// TestSwarmProgressCallbackIsRaceFree fans a single swarm call out to many
// parallel workers. The per-worker progress callback (onEach) used to append to
// a shared slice without synchronization, corrupting memory under load and
// surfacing as nondeterministic hangs at higher nesting depth. Run with -race
// to guard against a regression.
func TestSwarmProgressCallbackIsRaceFree(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, _ := w.(http.Flusher)
		b, _ := json.Marshal(streamChunkJSON("done", nil))
		fmt.Fprintf(w, "data: %s\n\n", b)
		if flusher != nil {
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	a := &Agent{
		cfg:     config.Runtime{Model: "test-model"},
		client:  opencode.NewClient(srv.URL, "test-key", "test-model"),
		workDir: t.TempDir(),
	}

	var tasks []string
	for i := 0; i < 24; i++ {
		tasks = append(tasks, fmt.Sprintf(`{"id":"t%d","prompt":"do %d"}`, i, i))
	}
	args := fmt.Sprintf(`{"tasks":[%s],"max_concurrency":12}`, strings.Join(tasks, ","))

	res := a.toolAgentSwarm(context.Background(), "outer", args, 0)
	if res.isErr {
		t.Fatalf("swarm errored: %s", res.output)
	}
	if !strings.Contains(res.output, "24 ok") {
		t.Fatalf("expected 24 ok workers, got:\n%s", res.output)
	}
}

// streamChunk/toolCall JSON shims mirror the subset of the OpenAI-style SSE
// schema that ChatStream parses, so the test server can emit valid deltas.
type streamDeltaJSON struct {
	Role      string         `json:"role,omitempty"`
	Content   string         `json:"content,omitempty"`
	ToolCalls []toolCallJSON `json:"tool_calls,omitempty"`
}

type toolCallJSON struct {
	Index    int        `json:"index"`
	ID       string     `json:"id,omitempty"`
	Type     string     `json:"type,omitempty"`
	Function toolFnJSON `json:"function"`
}

type toolFnJSON struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

func streamChunkJSON(content string, calls []toolCallJSON) map[string]any {
	return map[string]any{
		"choices": []map[string]any{{
			"delta": streamDeltaJSON{Role: "assistant", Content: content, ToolCalls: calls},
		}},
	}
}

func toolNames(tools []opencode.Tool) map[string]bool {
	m := make(map[string]bool, len(tools))
	for _, tl := range tools {
		m[tl.Function.Name] = true
	}
	return m
}

func hasTool(tools []opencode.Tool, name string) bool {
	return toolNames(tools)[name]
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
