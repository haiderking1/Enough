package tui

import "time"

const minRenderInterval = 16 * time.Millisecond

func (a *App) requestRender() {
	a.scheduleRender(false)
}

func (a *App) requestRenderNow() {
	a.scheduleRender(true)
}

func (a *App) scheduleRender(immediate bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	fire := func() {
		a.mu.Lock()
		a.renderTimer = nil
		a.renderPending = false
		a.lastRenderAt = time.Now()
		a.mu.Unlock()

		select {
		case a.renderCh <- struct{}{}:
		default:
		}
	}

	if immediate {
		if a.renderTimer != nil {
			if !a.renderTimer.Stop() {
				select {
				case <-a.renderTimer.C:
				default:
				}
			}
			a.renderTimer = nil
		}
		a.renderPending = false
		a.renderTimer = time.AfterFunc(0, fire)
		return
	}

	// Flame-style coalescing: one pending render covers bursts of requestRender calls.
	if a.renderPending {
		return
	}
	a.renderPending = true

	delay := time.Duration(0)
	if since := time.Since(a.lastRenderAt); since < minRenderInterval {
		delay = minRenderInterval - since
	}

	a.renderTimer = time.AfterFunc(delay, fire)
}

func (a *App) stopRenderTimer() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.renderTimer != nil {
		a.renderTimer.Stop()
		a.renderTimer = nil
	}
	a.renderPending = false
}
