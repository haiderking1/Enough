package agent

import (
	"strings"

	"github.com/enough/enough/backend/opencode"
)

func ModelContextWindow(provider, model string, configOverride int) int {
	if configOverride > 0 {
		return configOverride
	}
	if provider == "" {
		provider = opencode.ProviderOpenCode
	}
	if w := opencode.ResolveContextWindow(provider, model); w > 0 {
		return w
	}
	modelLower := strings.ToLower(model)
	if strings.Contains(modelLower, "deepseek-chat") {
		return 128000
	}
	if strings.Contains(modelLower, "claude-3-5-sonnet") {
		return 200000
	}
	if strings.Contains(modelLower, "gpt-4o") {
		return 128000
	}
	if strings.Contains(modelLower, "gemini-1.5-pro") {
		return 2000000
	}
	if strings.Contains(modelLower, "gemini-1.5-flash") {
		return 1000000
	}
	return 128000
}
