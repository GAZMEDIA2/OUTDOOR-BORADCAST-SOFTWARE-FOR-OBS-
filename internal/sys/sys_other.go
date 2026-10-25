//go:build !windows

package sys

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
)

// Setup: macOS and Linux do not block Van Link by default, so there is
// nothing to change.
func Setup() SetupResult {
	return SetupResult{Done: true, Message: "No changes needed on this computer."}
}

// Restore: nothing was changed.
func Restore() SetupResult {
	return SetupResult{Done: true, Message: "Nothing to restore on this computer."}
}

// RunElevated is only used on Windows.
func RunElevated(string) error { return nil }

var (
	mu  sync.Mutex
	cmd *exec.Cmd
)

// KeepAwake stops the computer sleeping while on air (macOS: caffeinate,
// Linux: systemd-inhibit when available).
func KeepAwake(on bool) {
	mu.Lock()
	defer mu.Unlock()
	if on && cmd == nil {
		pid := strconv.Itoa(os.Getpid())
		if runtime.GOOS == "darwin" {
			cmd = exec.Command("caffeinate", "-dims", "-w", pid)
		} else if p, err := exec.LookPath("systemd-inhibit"); err == nil {
			cmd = exec.Command(p, "--what=idle:sleep", "--why=Van Link broadcast", "sleep", "infinity")
		}
		if cmd != nil && cmd.Start() != nil {
			cmd = nil
		}
	}
	if !on && cmd != nil {
		cmd.Process.Kill()
		cmd.Wait()
		cmd = nil
	}
}

// Supported reports whether one-click setup does anything here.
func Supported() bool { return false }
