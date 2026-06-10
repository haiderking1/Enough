package web

import (
	"context"
	"strings"

	"github.com/enough/enough/backend/web/searxng"
)

func trimInput(s string) string {
	return strings.TrimSpace(s)
}

// Stop shuts down bundled SearXNG if Enough started it.
func Stop() {
	_ = searxng.Stop()
}

func NewSearchProvider(ctx context.Context) (Provider, error) {
	base, err := searxng.EnsureRunning(ctx)
	if err != nil {
		return nil, err
	}
	return &SearXNGProvider{BaseURL: base}, nil
}
