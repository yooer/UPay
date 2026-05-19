//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"syscall"
)

func setDaemonSysProcAttr(cmd *exec.Cmd) {
	// Windows下在此可以设置 HideWindow: true 等，但我们保持默认
}

func checkProcessExists(pid int) bool {
	// PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	handle, err := syscall.OpenProcess(0x1000, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	err = syscall.GetExitCodeProcess(handle, &exitCode)
	if err != nil {
		return false
	}
	return exitCode == 259 // 259 为 STILL_ACTIVE
}

func killProcess(pid int) error {
	// Windows 下通过 taskkill 递归终止进程树 (/T)
	cmd := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
	return cmd.Run()
}
