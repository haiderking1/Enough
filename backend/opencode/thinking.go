package opencode

// ThinkingLevel controls model reasoning effort (OpenCode + Codex + Responses API).
type ThinkingLevel string

const (
	ThinkingOff    ThinkingLevel = "off"
	ThinkingMinimal ThinkingLevel = "minimal"
	ThinkingLow    ThinkingLevel = "low"
	ThinkingMedium ThinkingLevel = "medium"
	ThinkingHigh   ThinkingLevel = "high"
	ThinkingXHigh  ThinkingLevel = "xhigh"
)

type ThinkingParams struct {
	Type string `json:"type"`
}

// deepseekV4FlashLevels matches Flame's opencode-go deepseek-v4-flash thinkingLevelMap.
var deepseekV4FlashLevels = []ThinkingLevel{ThinkingOff, ThinkingHigh, ThinkingXHigh}

// defaultReasoningLevels is the standard OpenAI/Codex reasoning effort ladder.
var defaultReasoningLevels = []ThinkingLevel{
	ThinkingOff, ThinkingMinimal, ThinkingLow, ThinkingMedium, ThinkingHigh, ThinkingXHigh,
}

func SupportsThinking(model string) bool {
	return len(SupportedThinkingLevels(model)) > 1
}

func SupportedThinkingLevels(model string) []ThinkingLevel {
	if m, ok := LookupCatalogModel(model); ok && len(m.ThinkingLevels) > 1 {
		return append([]ThinkingLevel(nil), m.ThinkingLevels...)
	}
	if SupportsThinkingLevels(model) {
		return append([]ThinkingLevel(nil), deepseekV4FlashLevels...)
	}
	if m, ok := LookupCatalogModel(model); ok && m.Reasoning {
		return append([]ThinkingLevel(nil), defaultReasoningLevels...)
	}
	return []ThinkingLevel{ThinkingOff}
}

func CycleThinkingLevel(current ThinkingLevel, model string) ThinkingLevel {
	levels := SupportedThinkingLevels(model)
	if len(levels) <= 1 {
		return ThinkingOff
	}
	idx := 0
	for i, l := range levels {
		if l == current {
			idx = i
			break
		}
	}
	return levels[(idx+1)%len(levels)]
}

func StepThinkingLevel(current ThinkingLevel, model string, delta int) ThinkingLevel {
	levels := SupportedThinkingLevels(model)
	if len(levels) <= 1 {
		return ThinkingOff
	}
	idx := 0
	for i, l := range levels {
		if l == current {
			idx = i
			break
		}
	}
	n := len(levels)
	idx = ((idx+delta)%n + n) % n
	return levels[idx]
}

func ApplyThinkingToRequest(req *ChatRequest, level ThinkingLevel, model string) {
	if !SupportsThinking(model) {
		return
	}
	if level == ThinkingOff || level == "" {
		req.Thinking = &ThinkingParams{Type: "disabled"}
		req.ReasoningEffort = ""
		return
	}

	effort := ReasoningEffortForAPI(level, model)
	if SupportsThinkingLevels(model) {
		req.Thinking = &ThinkingParams{Type: "enabled"}
		req.ReasoningEffort = effort
		return
	}

	req.ReasoningEffort = effort
}

// ReasoningEffortForAPI maps UI thinking levels to provider wire values.
func ReasoningEffortForAPI(level ThinkingLevel, model string) string {
	if level == ThinkingOff || level == "" {
		return ""
	}
	if SupportsThinkingLevels(model) {
		switch level {
		case ThinkingXHigh:
			return "max"
		case ThinkingHigh:
			return "high"
		default:
			return string(level)
		}
	}
	return string(level)
}

// NormalizeMessages ensures assistant messages include reasoning_content when required.
func NormalizeMessages(msgs []Message, model string) []Message {
	if !SupportsThinking(model) {
		return msgs
	}
	out := make([]Message, len(msgs))
	copy(out, msgs)
	for i := range out {
		if out[i].Role != "assistant" {
			continue
		}
		if out[i].ReasoningContent == nil {
			empty := ""
			out[i].ReasoningContent = &empty
		}
	}
	return out
}

func ParseThinkingLevel(s string) ThinkingLevel {
	switch ThinkingLevel(s) {
	case ThinkingMinimal, ThinkingLow, ThinkingMedium, ThinkingHigh, ThinkingXHigh:
		return ThinkingLevel(s)
	default:
		return ThinkingOff
	}
}

func FormatThinkingLabel(level ThinkingLevel) string {
	if level == "" || level == ThinkingOff {
		return "off"
	}
	return string(level)
}
