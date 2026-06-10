package tui

import (
	"github.com/enough/enough/backend/web"
	"github.com/enough/enough/frontend/tui/term"
)

func Run() error {
	defer web.Stop()

	t, err := term.New()
	if err != nil {
		return err
	}

	app := newApp(t)
	return app.run()
}
