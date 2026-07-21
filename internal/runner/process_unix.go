//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	// A negative pid addresses the process group, including descendants that
	// inherited the command's stdout/stderr and would otherwise keep Run blocked.
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
