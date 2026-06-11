package enoughhome

import (
	"os"
	"path/filepath"
)

// HomeDir returns the path to the Enough home directory (default: ~/.enough).
// If the ENOUGH_HOME environment variable is set, it uses that instead.
func HomeDir() string {
	if eh := os.Getenv("ENOUGH_HOME"); eh != "" {
		return eh
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".enough"
	}
	return filepath.Join(home, ".enough")
}
