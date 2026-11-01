//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const launchAgentLabel = "com.npcgui.desktop"

func launchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist"), nil
}

func currentAppPath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	const marker = ".app/Contents/MacOS/"
	if idx := strings.Index(exe, marker); idx >= 0 {
		return exe[:idx+len(".app")]
	}
	return exe
}

func xmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	).Replace(s)
}

// Supported reports whether launch at login is implemented.
func Supported() bool { return true }

// IsEnabled reports whether the LaunchAgent entry exists.
func IsEnabled() bool {
	path, err := launchAgentPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// SetEnabled creates or removes the LaunchAgent entry.
func SetEnabled(enabled bool) error {
	path, err := launchAgentPath()
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
	appPath := currentAppPath()
	if appPath == "" {
		return fmt.Errorf("无法确定应用路径")
	}
	args := []string{appPath, "--hidden"}
	if strings.HasSuffix(appPath, ".app") {
		args = []string{"/usr/bin/open", "-a", appPath, "--args", "--hidden"}
	}
	var argXML strings.Builder
	for _, arg := range args {
		fmt.Fprintf(&argXML, "    <string>%s</string>\n", xmlEscape(arg))
	}
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
%s  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>ProcessType</key>
  <string>Interactive</string>
</dict>
</plist>
`, launchAgentLabel, argXML.String())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
