//go:build !windows

package tray

import "fyne.io/systray"

func startTray(onReady, onExit func()) {
	start, _ := systray.RunWithExternalLoop(onReady, onExit)
	start()
}
