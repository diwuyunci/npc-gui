//go:build windows

package update

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func waitForProcessExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if pid <= 0 {
			return true
		}
		out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
		if err != nil || !strings.Contains(string(out), strconv.Itoa(pid)) {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}

func detachHelper(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x00000008}
}

func relaunchTarget(path string) error {
	cmd := exec.Command(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}
