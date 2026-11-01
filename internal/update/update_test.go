package update

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"npc-gui/internal/ghrelease"
)

func TestAppAssetName(t *testing.T) {
	if got := appAssetName("0.2.0", "darwin", "arm64"); got != "npc-gui_0.2.0_darwin_arm64.app.zip" {
		t.Fatalf("darwin asset = %q", got)
	}
	if got := appAssetName("0.2.0", "windows", "amd64"); got != "npc-gui_0.2.0_windows_amd64.exe" {
		t.Fatalf("windows asset = %q", got)
	}
	if got := appAssetName("0.2.0", "linux", "amd64"); got != "npc-gui_0.2.0_linux_amd64.tar.gz" {
		t.Fatalf("linux asset = %q", got)
	}
}

func TestSelectAsset(t *testing.T) {
	want := appAssetName("0.2.0", runtime.GOOS, runtime.GOARCH)
	rel := &ghrelease.Release{Assets: []ghrelease.Asset{
		{Name: "npc-gui_0.2.0_windows_386.exe"},
		{Name: want},
	}}
	got := selectAsset(rel, "0.2.0")
	if got == nil || got.Name != want {
		t.Fatalf("selectAsset = %+v, want %q", got, want)
	}
}

func TestApplyBinaryHelper(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "npc-gui")
	staged := filepath.Join(dir, "staged")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := helperConfig{Target: target, Staged: staged, Kind: "binary", NoRelaunch: true}
	if err := applyHelper(cfg, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "new" {
		t.Fatalf("target = %q err=%v", data, err)
	}
	old, err := os.ReadFile(target + ".old")
	if err != nil || string(old) != "old" {
		t.Fatalf("backup = %q err=%v", old, err)
	}
}

func TestApplyAppHelper(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "NPC 客户端.app")
	staged := filepath.Join(dir, "staged", "NPC 客户端.app")
	if err := os.MkdirAll(filepath.Join(target, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(staged, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "Contents", "MacOS", "npc-gui"), []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staged, "Contents", "MacOS", "npc-gui"), []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := helperConfig{Target: target, Staged: filepath.Join(staged, "Contents", "MacOS", "npc-gui"), StagedApp: staged, Kind: "app", NoRelaunch: true}
	if err := applyHelper(cfg, func(string, ...any) {}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(target, "Contents", "MacOS", "npc-gui"))
	if err != nil || string(data) != "new" {
		t.Fatalf("target = %q err=%v", data, err)
	}
}

func TestParseHelperArgs(t *testing.T) {
	cfg, err := parseHelperArgs([]string{"--apply-update", "--target", "/x", "--staged", "/y", "--old-pid", "12", "--no-relaunch"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Target != "/x" || cfg.Staged != "/y" || cfg.OldPID != 12 || !cfg.NoRelaunch {
		t.Fatalf("cfg = %+v", cfg)
	}
	if _, err := parseHelperArgs([]string{"--target", "/x"}); err == nil {
		t.Fatal("expected missing staged error")
	}
}
