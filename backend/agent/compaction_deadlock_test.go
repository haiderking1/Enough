package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/enough/enough/backend/config"
	"github.com/enough/enough/backend/core"
	"github.com/enough/enough/backend/opencode"
	"github.com/enough/enough/backend/secrets"
	"github.com/enough/enough/backend/session"
)

func TestCompactDoesNotDeadlock(t *testing.T) {
	dir, err := os.MkdirTemp("", "enough-compact-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	credPath := filepath.Join(dir, "credentials")
	os.Setenv("ENOUGH_CREDENTIALS_FILE", credPath)
	defer os.Unsetenv("ENOUGH_CREDENTIALS_FILE")

	if err := secrets.SetAPIKey("test-key"); err != nil {
		t.Fatal(err)
	}

	sm, err := session.ContinueRecent(dir)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadRuntime()
	if err != nil {
		t.Fatal(err)
	}

	ag := New(cfg, dir, sm)
	var kinds []string
	ag.SetEmit(func(e core.Event) { kinds = append(kinds, e.Kind) })

	done := make(chan struct{})
	go func() {
		_, _ = ag.Compact(context.Background(), "")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Compact deadlocked")
	}

	if len(kinds) < 2 {
		t.Fatalf("expected compaction start/end events, got %v", kinds)
	}
	if kinds[0] != core.EventCompactionStart || kinds[len(kinds)-1] != core.EventCompactionEnd {
		t.Fatalf("unexpected event order: %v", kinds)
	}
}

func TestCompactRequiresTwoMessages(t *testing.T) {
	dir, err := os.MkdirTemp("", "enough-compact-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", oldHome)

	credPath := filepath.Join(dir, "credentials")
	os.Setenv("ENOUGH_CREDENTIALS_FILE", credPath)
	defer os.Unsetenv("ENOUGH_CREDENTIALS_FILE")

	if err := secrets.SetAPIKey("test-key"); err != nil {
		t.Fatal(err)
	}

	sm, err := session.ContinueRecent(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = sm.AppendMessage(opencode.Message{Role: "user", Content: opencode.StringContent("solo")})

	cfg, err := config.LoadRuntime()
	if err != nil {
		t.Fatal(err)
	}

	ag := New(cfg, dir, sm)
	_, err = ag.Compact(context.Background(), "")
	if err == nil || err.Error() != "Nothing to compact (no messages yet)" {
		t.Fatalf("expected no messages yet error, got %v", err)
	}
}
