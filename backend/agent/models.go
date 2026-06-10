package agent

import "strings"

func ModelContextWindow(model string, configOverride int) int {
	if configOverride > 0 {
		return configOverride
	}
	modelLower := strings.ToLower(model)
	if strings.Contains(modelLower, "deepseek-v4-flash") || strings.Contains(modelLower, "deepseek-v4-pro") {
		return 1_000_000
	}
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
	// default fallback
	return 128000
}
