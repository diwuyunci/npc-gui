//go:build !windows

package update

import (
	"os/exec"
	"runtime"
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
		err := syscall.Kill(pid, 0)
		if err != nil {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	return false
}

func detachHelper(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

func relaunchTarget(path string) error {
	if runtime.GOOS == "darwin" && strings.HasSuffix(path, ".app") {
		return exec.Command("open", "-n", path).Start()
	}
	cmd := exec.Command(path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
