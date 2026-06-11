package obligations

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// DetectVerifyCommand inspects the workspace root and returns the project's
// verification command, or "" when none can be derived (manual mode).
func DetectVerifyCommand(workDir string) string {
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(workDir, name))
		return err == nil
	}

	if exists("go.mod") {
		return "go test ./..."
	}
	if data, err := os.ReadFile(filepath.Join(workDir, "package.json")); err == nil {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(data, &pkg) == nil && pkg.Scripts["test"] != "" {
			return "npm test"
		}
	}
	if exists("Cargo.toml") {
		return "cargo test"
	}
	if exists("pyproject.toml") || exists("pytest.ini") {
		return "pytest"
	}
	return ""
}
