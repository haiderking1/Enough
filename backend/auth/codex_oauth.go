package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	codexOAuthClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	codexOAuthIssuer     = "https://auth.openai.com"
	codexOAuthTokenURL   = "https://auth.openai.com/oauth/token"
	codexDefaultBaseURL  = "https://chatgpt.com/backend-api/codex"
	codexDeviceAuthURL   = codexOAuthIssuer + "/codex/device"
	codexRefreshSkewSecs = 120
)

// CodexDefaultBaseURL returns the ChatGPT Codex backend base URL.
func CodexDefaultBaseURL() string { return codexDefaultBaseURL }

// CodexCredentials holds runtime-ready OpenAI Codex OAuth credentials.
type CodexCredentials struct {
	AccessToken  string
	RefreshToken string
	BaseURL      string
}

// DeviceAuthStart is returned when the user must complete browser sign-in.
type DeviceAuthStart struct {
	UserCode     string
	DeviceAuthID string
	VerifyURL    string
	PollInterval time.Duration
}

// HasCodexAuth reports whether Codex OAuth tokens are stored.
func HasCodexAuth() bool {
	state, ok, err := loadCodexProviderState()
	if err != nil || !ok {
		return false
	}
	return state.Tokens.AccessToken != "" && state.Tokens.RefreshToken != ""
}

// StartCodexDeviceAuth begins the OpenAI device-code OAuth flow.
func StartCodexDeviceAuth(ctx context.Context) (DeviceAuthStart, error) {
	body, err := json.Marshal(map[string]string{"client_id": codexOAuthClientID})
	if err != nil {
		return DeviceAuthStart{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		codexOAuthIssuer+"/api/accounts/deviceauth/usercode",
		strings.NewReader(string(body)))
	if err != nil {
		return DeviceAuthStart{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceAuthStart{}, fmt.Errorf("device code request: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return DeviceAuthStart{}, fmt.Errorf("device code request returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var deviceData struct {
		UserCode     string `json:"user_code"`
		DeviceAuthID string `json:"device_auth_id"`
		Interval     any    `json:"interval"`
	}
	if err := json.Unmarshal(raw, &deviceData); err != nil {
		return DeviceAuthStart{}, fmt.Errorf("decode device code: %w", err)
	}
	if deviceData.UserCode == "" || deviceData.DeviceAuthID == "" {
		return DeviceAuthStart{}, fmt.Errorf("device code response missing required fields")
	}

	interval := 5
	switch v := deviceData.Interval.(type) {
	case float64:
		interval = int(v)
	case string:
		if n, err := fmt.Sscanf(v, "%d", &interval); n != 1 || err != nil {
			interval = 5
		}
	}
	if interval < 3 {
		interval = 3
	}

	return DeviceAuthStart{
		UserCode:     deviceData.UserCode,
		DeviceAuthID: deviceData.DeviceAuthID,
		VerifyURL:    codexDeviceAuthURL,
		PollInterval: time.Duration(interval) * time.Second,
	}, nil
}

// PollCodexDeviceAuth waits for the user to finish browser sign-in.
func PollCodexDeviceAuth(ctx context.Context, start DeviceAuthStart) error {
	deadline := time.Now().Add(15 * time.Minute)
	ticker := time.NewTicker(start.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("login timed out after 15 minutes")
			}

			body, err := json.Marshal(map[string]string{
				"device_auth_id": start.DeviceAuthID,
				"user_code":      start.UserCode,
			})
			if err != nil {
				return err
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodPost,
				codexOAuthIssuer+"/api/accounts/deviceauth/token",
				strings.NewReader(string(body)))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("device auth poll: %w", err)
			}

			raw, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			switch resp.StatusCode {
			case http.StatusOK:
				var codeResp struct {
					AuthorizationCode string `json:"authorization_code"`
					CodeVerifier      string `json:"code_verifier"`
				}
				if err := json.Unmarshal(raw, &codeResp); err != nil {
					return fmt.Errorf("decode device token: %w", err)
				}
				if codeResp.AuthorizationCode == "" || codeResp.CodeVerifier == "" {
					return fmt.Errorf("device auth response missing authorization_code or code_verifier")
				}
				return saveCodexTokensFromAuthCode(ctx, codeResp.AuthorizationCode, codeResp.CodeVerifier)
			case http.StatusForbidden, http.StatusNotFound:
				continue
			default:
				return fmt.Errorf("device auth polling returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
			}
		}
	}
}

// CompleteCodexDeviceLogin runs device auth start + poll until tokens are saved.
func CompleteCodexDeviceLogin(ctx context.Context) (DeviceAuthStart, error) {
	start, err := StartCodexDeviceAuth(ctx)
	if err != nil {
		return DeviceAuthStart{}, err
	}
	if err := PollCodexDeviceAuth(ctx, start); err != nil {
		return start, err
	}
	return start, nil
}

func saveCodexTokensFromAuthCode(ctx context.Context, authorizationCode, codeVerifier string) error {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {authorizationCode},
		"redirect_uri":  {codexOAuthIssuer + "/deviceauth/callback"},
		"client_id":     {codexOAuthClientID},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(raw, &tokens); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}
	if tokens.AccessToken == "" {
		return fmt.Errorf("token exchange did not return access_token")
	}

	return saveCodexProviderState(providerState{
		Tokens: tokenPair{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
		},
		BaseURL:     codexDefaultBaseURL,
		LastRefresh: time.Now().UTC().Format(time.RFC3339),
		AuthMode:    "chatgpt",
		Source:      "device-code",
	})
}

// ResolveCodexCredentials returns a fresh access token, refreshing when needed.
func ResolveCodexCredentials(ctx context.Context) (CodexCredentials, error) {
	state, ok, err := loadCodexProviderState()
	if err != nil {
		return CodexCredentials{}, err
	}
	if !ok {
		return CodexCredentials{}, fmt.Errorf("not connected — run: enough auth add openai-codex")
	}

	baseURL := strings.TrimRight(state.BaseURL, "/")
	if baseURL == "" {
		baseURL = codexDefaultBaseURL
	}

	access := state.Tokens.AccessToken
	refresh := state.Tokens.RefreshToken
	if access == "" || refresh == "" {
		return CodexCredentials{}, fmt.Errorf("codex auth incomplete — run: enough auth add openai-codex")
	}

	if codexAccessTokenExpiring(access, codexRefreshSkewSecs) {
		refreshed, err := refreshCodexTokens(ctx, refresh)
		if err != nil {
			return CodexCredentials{}, err
		}
		access = refreshed.AccessToken
		refresh = refreshed.RefreshToken
		state.Tokens.AccessToken = access
		state.Tokens.RefreshToken = refresh
		state.LastRefresh = time.Now().UTC().Format(time.RFC3339)
		if err := saveCodexProviderState(state); err != nil {
			return CodexCredentials{}, err
		}
	}

	return CodexCredentials{
		AccessToken:  access,
		RefreshToken: refresh,
		BaseURL:      baseURL,
	}, nil
}

func refreshCodexTokens(ctx context.Context, refreshToken string) (tokenPair, error) {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {codexOAuthClientID},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, codexOAuthTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenPair{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tokenPair{}, fmt.Errorf("codex token refresh: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return tokenPair{}, fmt.Errorf("codex token refresh failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return tokenPair{}, fmt.Errorf("decode refresh response: %w", err)
	}
	if payload.AccessToken == "" {
		return tokenPair{}, fmt.Errorf("codex refresh response missing access_token")
	}
	out := tokenPair{
		AccessToken:  payload.AccessToken,
		RefreshToken: refreshToken,
	}
	if payload.RefreshToken != "" {
		out.RefreshToken = payload.RefreshToken
	}
	return out, nil
}

func codexAccessTokenExpiring(token string, skewSeconds int) bool {
	exp, ok := jwtExpUnix(token)
	if !ok {
		return true
	}
	return time.Now().Unix() >= exp-int64(skewSeconds)
}

func jwtExpUnix(token string) (int64, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return 0, false
	}
	payloadB64 := parts[1] + strings.Repeat("=", (4-len(parts[1])%4)%4)
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(payloadB64, "="))
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(payloadB64)
		if err != nil {
			return 0, false
		}
	}
	var claims struct {
		Exp float64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil || claims.Exp == 0 {
		return 0, false
	}
	return int64(claims.Exp), true
}
