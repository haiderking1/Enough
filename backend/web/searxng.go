package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearXNGProvider queries a self-hosted SearXNG instance (JSON format).
type SearXNGProvider struct {
	BaseURL string
	Client  *http.Client
}

func (p *SearXNGProvider) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (p *SearXNGProvider) Search(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = defaultMaxPages
	}

	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	u, err := url.Parse(base + "/search")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("searxng: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Results []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("searxng: parse response: %w", err)
	}

	var out []SearchResult
	for _, r := range parsed.Results {
		if r.URL == "" {
			continue
		}
		out = append(out, SearchResult{Title: r.Title, URL: r.URL})
		if len(out) >= maxResults {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("searxng: no results for %q", query)
	}
	return out, nil
}
