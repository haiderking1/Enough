package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// CodexCloudflareHeaders returns headers required for chatgpt.com/backend-api/codex.
func CodexCloudflareHeaders(accessToken string) map[string]string {
	headers := map[string]string{
		"User-Agent": "codex_cli_rs/0.0.0 (Enough)",
		"originator": "codex_cli_rs",
	}
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return headers
	}
	parts := strings.Split(accessToken, ".")
	if len(parts) < 2 {
		return headers
	}
	payloadB64 := parts[1] + strings.Repeat("=", (4-len(parts[1])%4)%4)
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(payloadB64, "="))
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(payloadB64)
		if err != nil {
			return headers
		}
	}
	var claims struct {
		Auth struct {
			ChatGPTAccountID string `json:"chatgpt_account_id"`
		} `json:"https://api.openai.com/auth"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return headers
	}
	if id := strings.TrimSpace(claims.Auth.ChatGPTAccountID); id != "" {
		headers["ChatGPT-Account-ID"] = id
	}
	return headers
}
