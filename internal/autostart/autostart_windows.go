//go:build windows

package autostart

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKey = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"
const runValue = "NPCGUI"

// Supported reports whether launch at login is implemented.
func Supported() bool { return true }

func openRunKey() (registry.Key, error) {
	return registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE|registry.SET_VALUE)
}

// IsEnabled reports whether the Run entry exists.
func IsEnabled() bool {
	key, err := openRunKey()
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue(runValue)
	return err == nil && strings.TrimSpace(value) != ""
}

// SetEnabled creates or removes the Run entry.
func SetEnabled(enabled bool) error {
	key, err := openRunKey()
	if err != nil {
		return err
	}
	defer key.Close()
	if !enabled {
		err = key.DeleteValue(runValue)
		if err == registry.ErrNotExist {
			return nil
		}
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	return key.SetStringValue(runValue, fmt.Sprintf("\"%s\" --hidden", exe))
}
