//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func setDaemonSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新会话，脱离控制终端
	}
}
