package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/enough/enough/backend/enoughhome"
)

const providerOpenAICodex = "openai-codex"

type authStore struct {
	Providers map[string]providerState `json:"providers"`
}

type providerState struct {
	Tokens      tokenPair `json:"tokens"`
	BaseURL     string    `json:"base_url,omitempty"`
	LastRefresh string    `json:"last_refresh,omitempty"`
	Source      string    `json:"source,omitempty"`
	AuthMode    string    `json:"auth_mode,omitempty"`
}

type tokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

var authStoreMu sync.Mutex

func authStorePath() (string, error) {
	return filepath.Join(enoughhome.HomeDir(), "auth.json"), nil
}

func loadAuthStore() (authStore, error) {
	path, err := authStorePath()
	if err != nil {
		return authStore{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return authStore{Providers: map[string]providerState{}}, nil
	}
	if err != nil {
		return authStore{}, err
	}
	var store authStore
	if err := json.Unmarshal(data, &store); err != nil {
		return authStore{}, fmt.Errorf("decode auth.json: %w", err)
	}
	if store.Providers == nil {
		store.Providers = map[string]providerState{}
	}
	return store, nil
}

func saveAuthStore(store authStore) error {
	path, err := authStorePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if store.Providers == nil {
		store.Providers = map[string]providerState{}
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func loadCodexProviderState() (providerState, bool, error) {
	store, err := loadAuthStore()
	if err != nil {
		return providerState{}, false, err
	}
	state, ok := store.Providers[providerOpenAICodex]
	if !ok || state.Tokens.AccessToken == "" {
		return providerState{}, false, nil
	}
	return state, true, nil
}

func saveCodexProviderState(state providerState) error {
	authStoreMu.Lock()
	defer authStoreMu.Unlock()

	store, err := loadAuthStore()
	if err != nil {
		return err
	}
	store.Providers[providerOpenAICodex] = state
	return saveAuthStore(store)
}
