package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/enough/enough/backend/secrets"
)

// SaveAPIKey stores the OpenCode API key for the current user.
func SaveAPIKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("api key cannot be empty")
	}
	return secrets.SetAPIKey(key)
}

// Connected reports whether any credentials are stored.
func Connected() bool {
	if secrets.HasAPIKey() {
		return true
	}
	return HasCodexAuth()
}

// AddOpenAICodex runs browser OAuth and saves Codex credentials to auth.json.
func AddOpenAICodex(ctx context.Context) (DeviceAuthStart, error) {
	return CompleteCodexDeviceLogin(ctx)
}
