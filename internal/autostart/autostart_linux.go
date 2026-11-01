//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func desktopPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "autostart", "npc-gui.desktop"), nil
}

func quoteDesktopArg(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return "\"" + s + "\""
}

// Supported reports whether launch at login is implemented.
func Supported() bool { return true }

// IsEnabled reports whether the autostart desktop entry exists.
func IsEnabled() bool {
	path, err := desktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// SetEnabled creates or removes the autostart desktop entry.
func SetEnabled(enabled bool) error {
	path, err := desktopPath()
	if err != nil {
		return err
	}
	if !enabled {
		err = os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	content := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=NPC 客户端\nComment=NPC 桌面客户端\nExec=%s --hidden\nTerminal=false\nX-GNOME-Autostart-enabled=true\n", quoteDesktopArg(exe))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
