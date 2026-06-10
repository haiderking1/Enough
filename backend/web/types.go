package web

import (
	"context"
	"errors"
)

var (
	ErrEmptyInput       = errors.New("query cannot be empty")
	ErrNoSearchProvider = errors.New("web search unavailable")
)

// SearchResult is a lightweight search hit before full page fetch.
type SearchResult struct {
	Title string
	URL   string
}

// Provider finds result URLs for a query.
type Provider interface {
	Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
}

// Hit is one search result with fully extracted page content.
type Hit struct {
	Title   string
	URL     string
	Content string
	Error   string
}

// Options controls web search behavior.
type Options struct {
	MaxPages int
}

const (
	defaultMaxPages = 3
	maxPagesCap     = 5
	maxOutputBytes  = 96_000
	fetchTimeoutSec = 45
)
