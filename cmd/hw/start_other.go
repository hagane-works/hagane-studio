//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func startDetached(program string, args []string) (int, error) {
	cmd := exec.Command(program, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	return pid, cmd.Process.Release()
}

func useUTF8Console() func() { return func() {} }
