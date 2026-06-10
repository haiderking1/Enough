package agent

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (a *Agent) resolvePath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path is required")
	}

	var abs string
	if filepath.IsAbs(p) {
		abs = filepath.Clean(p)
	} else {
		abs = filepath.Clean(filepath.Join(a.workDir, p))
	}

	workAbs, err := filepath.Abs(a.workDir)
	if err != nil {
		return "", err
	}

	if abs != workAbs && !strings.HasPrefix(abs, workAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return abs, nil
}
