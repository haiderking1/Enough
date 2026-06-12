package config

import (
	"github.com/enough/enough/backend/auth"
	"github.com/enough/enough/backend/secrets"
)

// EnableOpenCodeProvider stores an API key and switches the active provider.
func EnableOpenCodeProvider(key string) error {
	if err := auth.SaveAPIKey(key); err != nil {
		return err
	}
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Provider = ProviderOpenCode
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	return Save(cfg)
}

// EnableCodexProvider switches runtime to OpenAI Codex OAuth.
func EnableCodexProvider() error {
	if !auth.HasCodexAuth() {
		return secrets.ErrNotConnected
	}
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Provider = ProviderCodex
	cfg.Endpoint = auth.CodexDefaultBaseURL()
	if cfg.Model == "" || cfg.Model == DefaultModel {
		cfg.Model = DefaultCodexModel
	}
	return Save(cfg)
}

// ConnectionSettings returns non-secret connection settings.
func ConnectionSettings() (provider, endpoint, model string, err error) {
	cfg, err := Load()
	if err != nil {
		return "", "", "", err
	}
	provider = cfg.Provider
	if provider == "" {
		provider = ProviderOpenCode
	}
	if cfg.Endpoint == "" {
		if provider == ProviderCodex {
			endpoint = auth.CodexDefaultBaseURL()
		} else {
			endpoint = DefaultEndpoint
		}
	} else {
		endpoint = cfg.Endpoint
	}
	model = cfg.Model
	if model == "" {
		if provider == ProviderCodex {
			model = DefaultCodexModel
		} else {
			model = DefaultModel
		}
	}
	return provider, endpoint, model, nil
}

// ApplyProviderModel switches provider, endpoint, and model settings.
func ApplyProviderModel(provider, model, thinkingLevel string) error {
	switch provider {
	case ProviderCodex:
		if !auth.HasCodexAuth() {
			return secrets.ErrNotConnected
		}
	case ProviderOpenCode:
		if !secrets.HasAPIKey() {
			return secrets.ErrNotConnected
		}
	default:
		provider = ProviderOpenCode
	}

	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Provider = provider
	cfg.Model = model
	cfg.ThinkingLevel = thinkingLevel

	switch provider {
	case ProviderCodex:
		cfg.Endpoint = auth.CodexDefaultBaseURL()
	default:
		if cfg.Endpoint == "" || cfg.Endpoint == auth.CodexDefaultBaseURL() {
			cfg.Endpoint = DefaultEndpoint
		}
	}
	return Save(cfg)
}
