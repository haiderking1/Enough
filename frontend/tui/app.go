package tui

import (
	"strings"
	"sync"

	"github.com/enough/enough/backend/auth"
	"github.com/enough/enough/backend/core"
	"github.com/enough/enough/frontend/tui/flame"
	"github.com/enough/enough/frontend/tui/term"
)

type App struct {
	term     *term.Terminal
	renderer *flame.Renderer
	keys     *keyReader
	styles   Styles

	width  int
	height int

	mode composerMode

	editor      Editor
	messages    []chatMsg
	slashCursor int

	running  bool
	agentCh  <-chan core.Event

	greeted bool
	quit    bool

	mu       sync.Mutex
	renderCh chan struct{}
}

func newApp(t *term.Terminal) *App {
	return &App{
		term:     t,
		renderer: flame.NewRenderer(t),
		keys:     newKeyReader(),
		styles:   NewStyles(),
		editor:   NewEditor(512),
		renderCh: make(chan struct{}, 1),
	}
}

func (a *App) requestRender() {
	select {
	case a.renderCh <- struct{}{}:
	default:
	}
}

func (a *App) run() error {
	a.width = a.term.Columns()
	a.height = a.term.Rows()

	inputCh := make(chan []byte, 32)
	if err := a.term.Start(func(b []byte) {
		select {
		case inputCh <- b:
		default:
		}
	}, func() {
		a.mu.Lock()
		a.width = a.term.Columns()
		a.height = a.term.Rows()
		a.mu.Unlock()
		a.requestRender()
	}); err != nil {
		return err
	}
	defer func() {
		a.renderer.Stop()
		a.term.Stop()
	}()

	if !auth.Connected() {
		a.appendMessage("system", "not connected — type / to connect")
	}
	a.greeted = true
	a.renderer.Render(a.buildLines())

	for !a.quit {
		a.mu.Lock()
		agentCh := a.agentCh
		a.mu.Unlock()

		select {
		case data := <-inputCh:
			a.handleInput(data)

		case e, ok := <-agentCh:
			if !ok {
				a.mu.Lock()
				a.running = false
				a.agentCh = nil
				a.mu.Unlock()
				a.requestRender()
			} else if chat, ok := eventToChatMsg(e); ok {
				a.appendMessage(chat.role, chat.text)
			}

		case <-a.renderCh:
			a.mu.Lock()
			lines := a.buildLines()
			a.mu.Unlock()
			a.renderer.Render(lines)
		}
	}

	return nil
}

func (a *App) handleInput(data []byte) {
	for _, k := range a.keys.feed(data) {
		if a.handleKey(k) {
			return
		}
	}
}

func (a *App) handleKey(k parsedKey) bool {
	switch k.action {
	case keyCtrlC, keyCtrlD:
		a.quit = true
		return true

	case keyEscape:
		if a.slashActive() {
			a.dismissSlashMenu()
			a.requestRender()
			return false
		}
		if a.mode == modeConnect {
			a.cancelConnect()
			a.requestRender()
			return false
		}
		a.quit = true
		return true
	}

	a.mu.Lock()
	running := a.running
	a.mu.Unlock()

	if !running && a.slashActive() {
		switch k.action {
		case keyUp:
			if a.slashCursor > 0 {
				a.slashCursor--
			}
			a.requestRender()
			return false
		case keyDown:
			a.slashCursor++
			a.clampSlashCursor()
			a.requestRender()
			return false
		case keyRune:
			if k.r == 'k' || k.r == 'K' {
				if a.slashCursor > 0 {
					a.slashCursor--
				}
				a.requestRender()
				return false
			}
			if k.r == 'j' || k.r == 'J' {
				a.slashCursor++
				a.clampSlashCursor()
				a.requestRender()
				return false
			}
		case keyTab:
			a.autocompleteSlash()
			a.requestRender()
			return false
		case keyEnter:
			cmds := a.filteredSlashCommands()
			if len(cmds) > 0 {
				a.clampSlashCursor()
				name := cmds[a.slashCursor].name
				a.editor.SetValue("")
				a.slashCursor = 0
				a.runSlashCommand(name)
				a.requestRender()
			}
			return false
		}
	}

	if k.action == keyEnter && !running {
		a.handleSubmit()
		a.requestRender()
		return false
	}

	if running {
		return false
	}

	prevFilter := a.slashFilter()
	a.applyEditorKey(k)
	if a.slashFilter() != prevFilter {
		a.slashCursor = 0
	}
	a.clampSlashCursor()
	a.requestRender()
	return false
}

func (a *App) applyEditorKey(k parsedKey) {
	switch k.action {
	case keyRune:
		a.editor.Insert(k.r)
	case keyBackspace:
		a.editor.Backspace()
	case keyDelete:
		a.editor.Delete()
	case keyLeft:
		a.editor.MoveLeft()
	case keyRight:
		a.editor.MoveRight()
	case keyHome:
		a.editor.Home()
	case keyEnd:
		a.editor.End()
	case keyPaste:
		a.editor.InsertPaste(k.paste)
	}
}

func (a *App) buildLines() []string {
	w := a.width
	if w <= 0 {
		w = 80
	}

	var lines []string

	chat := renderChat(a.styles, a.messages, w)
	if chat != "" {
		lines = append(lines, strings.Split(chat, "\n")...)
	}

	if menu := a.renderSlashMenu(w); menu != "" {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.Split(menu, "\n")...)
	}

	composer := a.styles.InputBox.
		Width(w - 2).
		Render(a.renderTaskInput())
	lines = append(lines, strings.Split(composer, "\n")...)

	// Pad top so composer stays at bottom when content is short (Flame-style).
	h := a.height
	if h <= 0 {
		h = 24
	}
	for len(lines) < h {
		lines = append([]string{""}, lines...)
	}

	return lines
}

func (a *App) renderTaskInput() string {
	value := a.editor.Value()

	if a.mode == modeConnect {
		prompt := a.styles.InputPrompt.Render("key ")
		if value == "" {
			return prompt + a.styles.InputCaret.Render("▎") + a.styles.InputHint.Render(connectPlaceholder)
		}
		return a.renderTypedLine(prompt, value)
	}

	prompt := a.styles.InputPrompt.Render("❯ ")

	if a.running {
		return prompt + a.styles.InputHint.Render("running...")
	}

	if value == "" {
		hint := taskPlaceholder
		if !auth.Connected() {
			hint = "type / for commands..."
		}
		return prompt + a.styles.InputCaret.Render("▎") + a.styles.InputHint.Render(hint)
	}

	return a.renderTypedLine(prompt, value)
}

func (a *App) renderTypedLine(prompt, value string) string {
	pos := a.editor.Cursor()
	runes := []rune(value)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}

	before := a.styles.Text.Render(string(runes[:pos]))

	if pos == len(runes) {
		return prompt + before + a.styles.InputCaret.Render("▎")
	}

	cur := a.styles.InputCaret.Render(string(runes[pos]))
	after := a.styles.Text.Render(string(runes[pos+1:]))

	return prompt + before + cur + after
}

func (a *App) appendMessage(role, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	a.messages = append(a.messages, chatMsg{role: role, text: text})
	a.requestRender()
}

func (a *App) runSlashCommand(name string) {
	a.handleSlash("/" + name)
}

func (a *App) handleSubmit() {
	raw := strings.TrimSpace(a.editor.Value())
	a.editor.SetValue("")

	if a.mode == modeConnect {
		a.saveAPIKey(raw)
		return
	}

	if strings.HasPrefix(raw, "/") {
		a.handleSlash(raw)
		return
	}

	if !auth.Connected() {
		a.appendMessage("error", "not connected — type / and pick connect")
		return
	}

	if raw == "" {
		return
	}

	a.appendMessage("user", raw)
	a.startAgent(raw)
}

func (a *App) startAgent(task string) {
	a.running = true
	a.agentCh = startAgent(task)
	a.requestRender()
}
