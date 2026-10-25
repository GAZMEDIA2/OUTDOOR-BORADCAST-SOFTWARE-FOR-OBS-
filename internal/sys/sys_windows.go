//go:build windows

package sys

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

// Setup asks Windows for permission ("Do you want to allow this app to make
// changes?") and adds an inbound firewall rule for Van Link.
func Setup() SetupResult {
	return elevate("firewall-add", "Firewall rule added. Van Link can now receive connections.")
}

// Restore removes everything Setup added.
func Restore() SetupResult {
	return elevate("firewall-remove", "Original settings restored.")
}

func elevate(arg, okMsg string) SetupResult {
	exe, err := os.Executable()
	if err != nil {
		return SetupResult{Message: err.Error()}
	}
	ps := "Start-Process -FilePath '" + strings.ReplaceAll(exe, "'", "''") +
		"' -ArgumentList '" + arg + "' -Verb RunAs -Wait -WindowStyle Hidden"
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Run(); err != nil {
		return SetupResult{Message: "The change was not made (permission was not given)."}
	}
	return SetupResult{Done: true, Message: okMsg}
}

// RunElevated is called in the elevated child process.
func RunElevated(arg string) error {
	exe, _ := os.Executable()
	switch arg {
	case "firewall-add":
		exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+RuleName).Run()
		return exec.Command("netsh", "advfirewall", "firewall", "add", "rule", "name="+RuleName,
			"dir=in", "action=allow", "program="+exe, "enable=yes", "profile=any").Run()
	case "firewall-remove":
		return exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", "name="+RuleName).Run()
	}
	return nil
}

var awake = make(chan bool, 1)

func init() {
	go func() {
		runtime.LockOSThread()
		set := syscall.NewLazyDLL("kernel32.dll").NewProc("SetThreadExecutionState")
		const esContinuous, esSystem, esDisplay = 0x80000000, 0x00000001, 0x00000002
		for on := range awake {
			if on {
				set.Call(uintptr(esContinuous | esSystem | esDisplay))
			} else {
				set.Call(uintptr(esContinuous))
			}
		}
	}()
}

// KeepAwake stops the computer sleeping while on air.
func KeepAwake(on bool) {
	select {
	case awake <- on:
	default:
	}
}

// Supported reports whether one-click setup is available here.
func Supported() bool { return true }
