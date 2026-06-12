package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/enough/enough/backend/auth"
)

var codexModelsURL = "https://chatgpt.com/backend-api/codex/models?client_version=1.0.0"

// codexContextFallback mirrors Hermes' verified Codex OAuth limits (Apr 2026).
var codexContextFallback = map[string]int{
	"gpt-5.5":             272_000,
	"gpt-5.4":             272_000,
	"gpt-5.4-mini":        272_000,
	"gpt-5.3-codex":       272_000,
	"gpt-5.3-codex-spark": 128_000,
	"gpt-5-codex":         272_000,
}

type codexModelsResponse struct {
	Models []codexModelEntry `json:"models"`
}

type codexModelEntry struct {
	Slug          string `json:"slug"`
	Title         string `json:"title"`
	ContextWindow int    `json:"context_window"`
	Visibility    string `json:"visibility"`
	Priority      int    `json:"priority"`
}

// FetchCodexModels loads the live Codex model catalog with context windows.
func FetchCodexModels(ctx context.Context, accessToken string) ([]ModelInfo, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return CodexModels(), fmt.Errorf("codex: missing access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, codexModelsURL, nil)
	if err != nil {
		return CodexModels(), err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	for k, v := range auth.CodexCloudflareHeaders(accessToken) {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return CodexModels(), err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return CodexModels(), err
	}
	if resp.StatusCode >= 400 {
		return CodexModels(), fmt.Errorf("codex models %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload codexModelsResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return CodexModels(), fmt.Errorf("decode codex models: %w", err)
	}

	type ranked struct {
		rank int
		m    ModelInfo
	}
	var sortable []ranked
	for _, entry := range payload.Models {
		slug := strings.TrimSpace(entry.Slug)
		if slug == "" {
			continue
		}
		vis := strings.ToLower(strings.TrimSpace(entry.Visibility))
		if vis == "hide" || vis == "hidden" {
			continue
		}

		name := strings.TrimSpace(entry.Title)
		if name == "" {
			if known, ok := codexKnownModels[slug]; ok {
				name = known.Name
			} else {
				name = slug
			}
		}

		ctxWindow := entry.ContextWindow
		if ctxWindow <= 0 {
			ctxWindow = codexContextFallbackFor(slug)
		}

		m := normalizeModel(ModelInfo{
			ID:            slug,
			Name:          name,
			ContextWindow: ctxWindow,
			Reasoning:     true,
			ThinkingLevels: append([]ThinkingLevel(nil), defaultReasoningLevels...),
		})
		rank := entry.Priority
		if rank <= 0 {
			rank = 10_000
		}
		sortable = append(sortable, ranked{rank: rank, m: m})
	}

	if len(sortable) == 0 {
		return CodexModels(), fmt.Errorf("codex models: empty list")
	}

	sort.Slice(sortable, func(i, j int) bool {
		if sortable[i].rank != sortable[j].rank {
			return sortable[i].rank < sortable[j].rank
		}
		return strings.ToLower(sortable[i].m.Name) < strings.ToLower(sortable[j].m.Name)
	})
	out := make([]ModelInfo, len(sortable))
	for i, item := range sortable {
		out[i] = item.m
	}
	return out, nil
}
