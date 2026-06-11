package session

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/enough/enough/backend/enoughhome"
)

const (
	AgentDirName    = ".enough"
	AgentSubdir     = "agent"
	SessionsSubdir  = "sessions"
	CurrentVersion  = 1
)

func HomeAgentDir() (string, error) {
	return filepath.Join(enoughhome.HomeDir(), AgentSubdir), nil
}

func EncodeCWD(cwd string) string {
	s := strings.TrimPrefix(cwd, string(os.PathSeparator))
	if vol := filepath.VolumeName(cwd); vol != "" {
		s = strings.TrimPrefix(cwd, vol)
		s = strings.TrimPrefix(s, string(os.PathSeparator))
	}
	s = strings.NewReplacer(
		string(os.PathSeparator), "-",
		":", "-",
	).Replace(s)
	return "--" + s + "--"
}

func SessionDir(cwd string) (string, error) {
	agentDir, err := HomeAgentDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(agentDir, SessionsSubdir, EncodeCWD(cwd))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}
