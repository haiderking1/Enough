//go:build !unix

package agent

import "os/exec"

// configureProcGroup is a no-op on non-unix platforms; exec.CommandContext's
// default cancellation (killing the process) still applies.
func configureProcGroup(cmd *exec.Cmd) {}
