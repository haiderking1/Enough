package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormatTruncates(t *testing.T) {
	hits := []Hit{{Title: "T", URL: "https://example.com", Content: strings.Repeat("x", maxOutputBytes+10)}}
	out := Format(hits)
	if len(out) <= maxOutputBytes+32 {
		// includes truncation marker
	}
	if !strings.Contains(out, "truncated") {
		t.Fatalf("expected truncation marker")
	}
}

func TestValidateFetchURLBlocksLocalhost(t *testing.T) {
	t.Setenv("ENOUGH_WEB_ALLOW_PRIVATE", "0")
	if _, err := validateFetchURL("http://localhost/"); err == nil {
		t.Fatal("expected localhost to be blocked")
	}
}

func TestSearchDirectURL(t *testing.T) {
	t.Setenv("ENOUGH_WEB_ALLOW_PRIVATE", "1")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>Docs</title></head><body><article><h1>Hello</h1><p>` +
			strings.Repeat("Readable article body. ", 40) + `</p></article></body></html>`))
	}))
	defer srv.Close()

	hits, err := Search(context.Background(), srv.URL, Options{})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("got %d hits", len(hits))
	}
	if hits[0].Error != "" {
		t.Fatalf("fetch error: %s", hits[0].Error)
	}
	if !strings.Contains(hits[0].Content, "Readable article body") {
		t.Fatalf("missing content: %q", hits[0].Content)
	}
}
