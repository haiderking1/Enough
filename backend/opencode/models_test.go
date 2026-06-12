package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchModelsMergesMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"deepseek-v4-flash"},{"id":"hy3-preview"}]}`))
	}))
	defer srv.Close()

	models, err := FetchModels(context.Background(), srv.URL, "")
	if err != nil {
		t.Fatalf("FetchModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("len(models) = %d, want 2", len(models))
	}

	var flash ModelInfo
	for _, m := range models {
		if m.ID == "deepseek-v4-flash" {
			flash = m
			break
		}
	}
	if flash.ContextWindow != 1_000_000 {
		t.Fatalf("context window = %d, want 1000000", flash.ContextWindow)
	}
	if flash.Name != "DeepSeek V4 Flash" {
		t.Fatalf("name = %q", flash.Name)
	}
	if len(flash.ThinkingLevels) != 3 {
		t.Fatalf("thinking levels = %v", flash.ThinkingLevels)
	}
}

func TestRegistryLookupFallback(t *testing.T) {
	r := NewRegistry()
	m, ok := r.Lookup("deepseek-v4-pro")
	if !ok {
		t.Fatal("expected lookup ok")
	}
	if m.ContextWindow != 1_000_000 {
		t.Fatalf("context = %d", m.ContextWindow)
	}
}

func TestFormatContextWindow(t *testing.T) {
	if got := FormatContextWindow(1_000_000); got != "1M" {
		t.Fatalf("got %q", got)
	}
	if got := FormatContextWindow(262144); got != "262.1k" {
		t.Fatalf("got %q", got)
	}
}
