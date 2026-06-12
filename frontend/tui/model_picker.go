package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/secrets"
)

func (a *App) startModelFetch() {
	endpoint := config.DefaultEndpoint
	if cfg, err := config.Load(); err == nil && cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}
	apiKey, _ := secrets.GetAPIKey()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = a.modelRegistry.Refresh(ctx, endpoint, apiKey)
		a.requestRender()
	}()
}

func (a *App) openModelPicker(filter string) {
	a.modelPickerFilter = strings.ToLower(strings.TrimSpace(filter))
	a.modelPickerCursor = 0
	a.modelPickerStatus = ""
	a.modelPickerThinking = opencode.ThinkingOff

	cfg, err := config.Load()
	if err == nil {
		a.modelPickerThinking = opencode.ParseThinkingLevel(cfg.ThinkingLevel)
		for i, m := range a.filteredModelPickerItems() {
			if m.ID == cfg.Model {
				a.modelPickerCursor = i
				break
			}
		}
	}

	a.mode = modeModelPicker
	a.editor.SetValue("")
	a.requestRender()
}

func (a *App) dismissModelPicker() {
	a.mode = modeTask
	a.modelPickerFilter = ""
	a.modelPickerCursor = 0
	a.modelPickerStatus = ""
	a.requestRender()
}

func (a *App) filteredModelPickerItems() []opencode.ModelInfo {
	items := a.modelRegistry.Models()
	if len(items) == 0 {
		items = opencode.FallbackModels()
	}
	filter := a.modelPickerFilter
	if filter == "" {
		return items
	}
	out := make([]opencode.ModelInfo, 0, len(items))
	for _, m := range items {
		if strings.Contains(strings.ToLower(m.ID), filter) ||
			strings.Contains(strings.ToLower(m.Name), filter) {
			out = append(out, m)
		}
	}
	return out
}

func (a *App) clampModelPickerCursor() {
	items := a.filteredModelPickerItems()
	if len(items) == 0 {
		a.modelPickerCursor = 0
		return
	}
	if a.modelPickerCursor >= len(items) {
		a.modelPickerCursor = len(items) - 1
	}
	if a.modelPickerCursor < 0 {
		a.modelPickerCursor = 0
	}
}

func (a *App) modelPickerCurrent() (opencode.ModelInfo, bool) {
	items := a.filteredModelPickerItems()
	a.clampModelPickerCursor()
	if len(items) == 0 {
		return opencode.ModelInfo{}, false
	}
	return items[a.modelPickerCursor], true
}

func (a *App) syncModelPickerThinking() {
	m, ok := a.modelPickerCurrent()
	if !ok {
		return
	}
	levels := opencode.SupportedThinkingLevels(m.ID)
	if len(levels) <= 1 {
		a.modelPickerThinking = opencode.ThinkingOff
		return
	}
	for _, l := range levels {
		if l == a.modelPickerThinking {
			return
		}
	}
	a.modelPickerThinking = levels[0]
}

func (a *App) cycleModelPickerThinking() {
	m, ok := a.modelPickerCurrent()
	if !ok || !opencode.SupportsThinking(m.ID) {
		return
	}
	a.modelPickerThinking = opencode.CycleThinkingLevel(a.modelPickerThinking, m.ID)
	a.requestRender()
}

func (a *App) applyModelSelection() {
	m, ok := a.modelPickerCurrent()
	if !ok {
		a.modelPickerStatus = "No models available"
		return
	}

	cfg, err := config.Load()
	if err != nil {
		a.modelPickerStatus = err.Error()
		return
	}

	cfg.Model = m.ID
	if opencode.SupportsThinking(m.ID) {
		cfg.ThinkingLevel = string(a.modelPickerThinking)
	} else {
		cfg.ThinkingLevel = ""
	}

	if err := config.Save(cfg); err != nil {
		a.modelPickerStatus = err.Error()
		return
	}

	a.thinkingLevel = opencode.ParseThinkingLevel(cfg.ThinkingLevel)
	a.dismissModelPicker()

	msg := fmt.Sprintf("Model: %s (%s ctx)", m.Name, opencode.FormatContextWindow(m.ContextWindow))
	if opencode.SupportsThinking(m.ID) {
		msg += fmt.Sprintf(" · thinking %s", cfg.ThinkingLevel)
	} else if m.Reasoning {
		msg += " · reasoning"
	}
	a.appendMessage("system", msg)

	if runCfg, err := config.LoadRuntime(); err == nil {
		a.mu.Lock()
		if a.agent != nil {
			a.agent.UpdateConfig(runCfg)
		}
		a.mu.Unlock()
	}
	a.requestRender()
}

func (a *App) handleModelPickerKey(k parsedKey) bool {
	switch k.action {
	case keyUp:
		if a.modelPickerCursor > 0 {
			a.modelPickerCursor--
		}
		a.syncModelPickerThinking()
		a.requestRender()
		return true
	case keyDown:
		a.modelPickerCursor++
		a.clampModelPickerCursor()
		a.syncModelPickerThinking()
		a.requestRender()
		return true
	case keyRune:
		if k.r == 'k' || k.r == 'K' {
			if a.modelPickerCursor > 0 {
				a.modelPickerCursor--
			}
			a.syncModelPickerThinking()
			a.requestRender()
			return true
		}
		if k.r == 'j' || k.r == 'J' {
			a.modelPickerCursor++
			a.clampModelPickerCursor()
			a.syncModelPickerThinking()
			a.requestRender()
			return true
		}
	case keyTab:
		a.cycleModelPickerThinking()
		return true
	case keyEnter:
		a.applyModelSelection()
		return true
	}
	return false
}

func (a *App) renderModelPicker(width int) string {
	if a.mode != modeModelPicker {
		return ""
	}

	a.clampModelPickerCursor()

	cfg, _ := config.Load()
	currentModel := cfg.Model

	items := a.filteredModelPickerItems()
	var lines []string
	title := "  Select model"
	if a.modelPickerFilter != "" {
		title += fmt.Sprintf(" (filter: %s)", a.modelPickerFilter)
	}
	lines = append(lines, a.styles.SlashSelected.Render(title))

	if fetchErr := a.modelRegistry.Err(); fetchErr != nil && len(items) == 0 {
		lines = append(lines, a.styles.SlashDim.Render("  could not fetch models — using catalog"))
	}

	if len(items) == 0 {
		lines = append(lines, a.styles.SlashDim.Render("  no matching models"))
	} else {
		for i, m := range items {
			marker := "  "
			if m.ID == currentModel {
				marker = "* "
			}
			if i == a.modelPickerCursor {
				if m.ID == currentModel {
					marker = "›*"
				} else {
					marker = "› "
				}
			}

			thinking := opencode.ThinkingOff
			if opencode.SupportsThinking(m.ID) && i == a.modelPickerCursor {
				thinking = a.modelPickerThinking
			} else if m.ID == currentModel {
				thinking = opencode.ParseThinkingLevel(cfg.ThinkingLevel)
			}

			meta := opencode.FormatContextWindow(m.ContextWindow) + " ctx"
			if badge := opencode.FormatThinkingBadge(m, thinking); badge != "" {
				meta += " · " + badge
			}

			line := fmt.Sprintf("%s%-22s  %s", marker, m.Name, meta)
			style := a.styles.SlashDim
			if i == a.modelPickerCursor {
				style = a.styles.SlashSelected
			}
			lines = append(lines, style.Render(line))
		}
	}

	var hint string
	switch {
	case a.modelPickerStatus != "":
		hint = a.styles.AssistError.Render("  " + a.modelPickerStatus)
	default:
		hint = a.styles.SlashDim.Render("  ↑↓ pick   tab thinking   enter apply   esc close")
	}
	body := strings.Join(lines, "\n") + "\n" + hint

	return a.styles.SlashMenu.
		Width(width - 2).
		Render(body)
}
