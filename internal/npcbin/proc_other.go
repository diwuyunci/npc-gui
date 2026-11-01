//go:build !windows

package npcbin

import "os/exec"

func hideWindow(cmd *exec.Cmd) {}
