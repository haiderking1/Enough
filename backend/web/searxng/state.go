package searxng

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type runState struct {
	Port int `json:"port"`
	PID  int `json:"pid"`
}

func (m *Manager) statePath() string {
	return filepath.Join(m.dataDir, "run.json")
}

func (m *Manager) writeState(port, pid int) error {
	data, err := json.Marshal(runState{Port: port, PID: pid})
	if err != nil {
		return err
	}
	return os.WriteFile(m.statePath(), data, 0o600)
}

func (m *Manager) readState() (port, pid int, ok bool) {
	data, err := os.ReadFile(m.statePath())
	if err != nil {
		return 0, 0, false
	}
	var st runState
	if err := json.Unmarshal(data, &st); err != nil {
		return 0, 0, false
	}
	return st.Port, st.PID, st.Port > 0 && st.PID > 0
}
