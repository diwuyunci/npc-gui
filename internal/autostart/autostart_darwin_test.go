//go:build darwin

package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestLaunchAgentLifecycle(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if IsEnabled() {
		t.Fatal("autostart unexpectedly enabled")
	}
	if err := SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if !IsEnabled() {
		t.Fatal("autostart not enabled")
	}
	path, err := launchAgentPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "--hidden") || !strings.Contains(content, "com.npcgui.desktop") {
		t.Fatalf("unexpected LaunchAgent: %s", content)
	}
	if err := SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if IsEnabled() {
		t.Fatal("autostart still enabled")
	}
}
