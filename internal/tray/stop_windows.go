//go:build windows

package tray

import "fyne.io/systray"

func stopTrayIcon() {
	systray.Quit()
}
