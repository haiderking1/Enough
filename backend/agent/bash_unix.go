//go:build unix

package agent

import (
	"os/exec"
	"syscall"
)

// configureProcGroup puts the command in its own process group and, on context
// cancellation, kills the whole group so child processes spawned by the shell
// (pipelines, subshells) are torn down too — not just the bash parent.
func configureProcGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		// Negative pid targets the process group.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
