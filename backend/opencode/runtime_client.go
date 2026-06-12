package opencode

import "github.com/enough/enough/backend/config"

// NewClientForRuntime builds the correct HTTP client for the active provider.
func NewClientForRuntime(cfg config.Runtime) *Client {
	if cfg.Provider == config.ProviderCodex {
		return NewCodexClient(cfg.Endpoint, cfg.APIKey, cfg.Model)
	}
	return NewClient(cfg.Endpoint, cfg.APIKey, cfg.Model)
}
