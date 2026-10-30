package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func startDetached(program string, args []string) (int, error) {
	cmd := exec.Command(program, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	return pid, cmd.Process.Release()
}

const cpUTF8 = 65001

func useUTF8Console() func() {
	output, outputErr := windows.GetConsoleOutputCP()
	input, inputErr := windows.GetConsoleCP()
	if outputErr != nil || inputErr != nil {
		return func() {}
	}
	_ = windows.SetConsoleOutputCP(cpUTF8)
	_ = windows.SetConsoleCP(cpUTF8)
	return func() {
		_ = windows.SetConsoleOutputCP(output)
		_ = windows.SetConsoleCP(input)
	}
}
