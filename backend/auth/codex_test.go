package auth

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCodexCloudflareHeaders(t *testing.T) {
	claims := map[string]any{
		"https://api.openai.com/auth": map[string]string{
			"chatgpt_account_id": "acct-test-123",
		},
	}
	raw, _ := json.Marshal(claims)
	token := "hdr." + base64.RawURLEncoding.EncodeToString(raw) + ".sig"

	headers := CodexCloudflareHeaders(token)
	if headers["originator"] != "codex_cli_rs" {
		t.Fatalf("originator = %q", headers["originator"])
	}
	if headers["ChatGPT-Account-ID"] != "acct-test-123" {
		t.Fatalf("account id = %q", headers["ChatGPT-Account-ID"])
	}
}

func TestHasCodexAuth(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ENOUGH_HOME", dir)

	if HasCodexAuth() {
		t.Fatal("expected no codex auth initially")
	}

	if err := saveCodexProviderState(providerState{
		Tokens: tokenPair{
			AccessToken:  "access",
			RefreshToken: "refresh",
		},
		BaseURL: codexDefaultBaseURL,
	}); err != nil {
		t.Fatal(err)
	}
	if !HasCodexAuth() {
		t.Fatal("expected codex auth after save")
	}

	path := filepath.Join(dir, "auth.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("auth.json not written: %v", err)
	}
}
