package searxng

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

//go:embed settings.yml
var defaultSettings []byte

const (
	repoURL        = "https://github.com/searxng/searxng.git"
	healthTimeout  = 90 * time.Second
	healthInterval = 400 * time.Millisecond
)

var (
	manager     *Manager
	managerOnce sync.Once
)

// Manager runs a local SearXNG instance for Enough.
type Manager struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	baseURL string
	dataDir string
}

// Default returns the shared bundled SearXNG manager.
func Default() *Manager {
	managerOnce.Do(func() {
		dir, err := dataDir()
		if err != nil {
			manager = &Manager{}
			return
		}
		manager = &Manager{dataDir: dir}
	})
	return manager
}

// EnsureRunning installs (if needed), starts SearXNG, and returns its base URL.
func EnsureRunning(ctx context.Context) (string, error) {
	return Default().EnsureRunning(ctx)
}

// Stop shuts down a SearXNG process started by Enough.
func Stop() error {
	return Default().Stop()
}

func (m *Manager) EnsureRunning(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.baseURL != "" && m.healthOK(ctx, m.baseURL) {
		return m.baseURL, nil
	}

	if m.dataDir == "" {
		return "", fmt.Errorf("searxng data dir unavailable")
	}

	if base, ok := m.reuseExisting(ctx); ok {
		m.baseURL = base
		return base, nil
	}

	if err := m.ensureInstalled(ctx); err != nil {
		return "", err
	}

	port, err := freePort()
	if err != nil {
		return "", err
	}

	settingsPath, err := m.writeSettings(port)
	if err != nil {
		return "", err
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	python := filepath.Join(m.dataDir, "venv", "bin", "python")
	srcDir := filepath.Join(m.dataDir, "src")

	cmd := exec.CommandContext(ctx, python, "-m", "searx.webapp")
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(),
		"SEARXNG_SETTINGS_PATH="+settingsPath,
		"SEARXNG_BASE_URL="+baseURL+"/",
	)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start searxng: %w", err)
	}

	m.cmd = cmd
	if err := waitHealthy(ctx, baseURL); err != nil {
		_ = m.stopLocked()
		return "", err
	}

	m.baseURL = baseURL
	_ = m.writeState(port, cmd.Process.Pid)
	return baseURL, nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopLocked()
}

func (m *Manager) stopLocked() error {
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() {
			_ = m.cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = m.cmd.Process.Kill()
		}
	}
	m.cmd = nil
	m.baseURL = ""
	return nil
}

func (m *Manager) ensureInstalled(ctx context.Context) error {
	srcDir := filepath.Join(m.dataDir, "src")
	if _, err := os.Stat(filepath.Join(srcDir, "searx", "webapp.py")); err == nil {
		return m.ensureVenv(ctx)
	}

	if err := os.MkdirAll(m.dataDir, 0o700); err != nil {
		return err
	}

	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("searxng install requires git: %w", err)
	}
	if _, err := exec.LookPath("python3"); err != nil {
		return fmt.Errorf("searxng install requires python3: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", repoURL, srcDir)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clone searxng: %w", err)
	}
	return m.ensureVenv(ctx)
}

func (m *Manager) ensureVenv(ctx context.Context) error {
	venvPython := filepath.Join(m.dataDir, "venv", "bin", "python")
	if _, err := os.Stat(venvPython); err == nil {
		return nil
	}

	py3, err := exec.LookPath("python3")
	if err != nil {
		return fmt.Errorf("python3 not found: %w", err)
	}

	venvDir := filepath.Join(m.dataDir, "venv")
	cmd := exec.CommandContext(ctx, py3, "-m", "venv", venvDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create venv: %w: %s", err, string(out))
	}

	reqs := filepath.Join(m.dataDir, "src", "requirements.txt")
	pip := filepath.Join(venvDir, "bin", "pip")
	cmd = exec.CommandContext(ctx, pip, "install", "-r", reqs)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("install searxng dependencies (this may take a minute): %w", err)
	}
	return nil
}

func (m *Manager) writeSettings(port int) (string, error) {
	text := strings.ReplaceAll(string(defaultSettings), "port: 18752", "port: "+strconv.Itoa(port))
	path := filepath.Join(m.dataDir, "settings.yml")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) reuseExisting(ctx context.Context) (string, bool) {
	port, pid, ok := m.readState()
	if !ok {
		return "", false
	}
	if !processAlive(pid) {
		return "", false
	}
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	if m.healthOK(ctx, base) {
		return base, true
	}
	return "", false
}

func (m *Manager) healthOK(ctx context.Context, base string) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/healthz", nil)
	if err != nil {
		return false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func waitHealthy(ctx context.Context, base string) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, healthTimeout)
		defer cancel()
	}

	ticker := time.NewTicker(healthInterval)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/healthz", nil)
		if err != nil {
			return err
		}
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("searxng did not become ready: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "enough", "searxng"), nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
