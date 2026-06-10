package web

import (
	"context"
)

// Search runs a web query (or fetches a URL directly) and returns full readable page text.
func Search(ctx context.Context, input string, opts Options) ([]Hit, error) {
	input = trimInput(input)
	if input == "" {
		return nil, ErrEmptyInput
	}

	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = defaultMaxPages
	}
	if maxPages > maxPagesCap {
		maxPages = maxPagesCap
	}

	if isHTTPURL(input) {
		page, err := FetchPage(ctx, input)
		if err != nil {
			return []Hit{{URL: input, Error: err.Error()}}, nil
		}
		return []Hit{page}, nil
	}

	provider, err := NewSearchProvider(ctx)
	if err != nil {
		return nil, err
	}

	results, err := provider.Search(ctx, input, maxPages)
	if err != nil {
		return nil, err
	}

	var hits []Hit
	for _, r := range results {
		if err := ctx.Err(); err != nil {
			return hits, err
		}
		page, err := FetchPage(ctx, r.URL)
		if err != nil {
			hits = append(hits, Hit{
				Title: r.Title,
				URL:   r.URL,
				Error: err.Error(),
			})
			continue
		}
		if page.Title == "" {
			page.Title = r.Title
		}
		hits = append(hits, page)
	}
	return hits, nil
}

// Format renders hits as plain text for the agent tool.
func Format(hits []Hit) string {
	var out string
	for i, hit := range hits {
		if i > 0 {
			out += "\n\n"
		}
		title := hit.Title
		if title == "" {
			title = hit.URL
		}
		out += "=== " + title + " ===\n"
		out += "URL: " + hit.URL + "\n\n"
		if hit.Error != "" {
			out += "Error: " + hit.Error + "\n"
			continue
		}
		out += hit.Content
	}

	if len(out) > maxOutputBytes {
		out = out[:maxOutputBytes] + "\n\n... truncated ..."
	}
	return out
}
