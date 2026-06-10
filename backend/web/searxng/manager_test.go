package searxng

import (
	"os"
	"strings"
	"testing"
)

func TestWriteSettingsPort(t *testing.T) {
	m := &Manager{dataDir: t.TempDir()}
	path, err := m.writeSettings(19999)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "port: 19999") {
		t.Fatalf("expected port 19999 in settings: %s", data)
	}
}
