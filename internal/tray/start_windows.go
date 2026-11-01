//go:build windows

package tray

import (
	"runtime"

	"fyne.io/systray"
)

// startTray creates the notification icon and runs its message loop on the
// same locked OS thread. This is required on Windows because the tray window
// receives its click messages on the thread that created it.
func startTray(onReady, onExit func()) {
	go func() {
		runtime.LockOSThread()
		systray.Run(onReady, onExit)
	}()
}
