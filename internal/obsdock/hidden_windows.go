//go:build windows

package obsdock

import (
	"os/exec"
	"syscall"
)

func hiddenCommand(name string, args ...string) *exec.Cmd {
	c := exec.Command(name, args...)
	c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return c
}
