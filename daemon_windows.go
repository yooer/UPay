//go:build windows

package main

import (
	"os/exec"
)

func setDaemonSysProcAttr(cmd *exec.Cmd) {
	// Windows-specific background process setting (optional)
	// cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
