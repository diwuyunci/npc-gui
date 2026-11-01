package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"npc-gui/internal/model"
)

func testProfile() model.Profile {
	p := model.NewProfile()
	p.ID = "abc123"
	p.Name = "测试"
	p.Server = "127.0.0.1:8024"
	p.VKey = "secret"
	return p
}

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.SaveProfile(testProfile())
	if err != nil {
		t.Fatal(err)
	}
	if p.ID == "" {
		t.Fatal("empty id")
	}
	if _, err := s.SaveSettings(model.Settings{
		AutoCheckUpdates: true,
		Theme:            "dark",
		Locale:           "zh-CN",
		DefaultLogLevel:  "info",
		MaxLogLines:      5000,
		ConfirmOnQuit:    true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBinary(&model.BinaryInfo{Path: "/tmp/npc", Version: "0.34.10"}); err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "profiles.json"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("profiles.json mode = %v, want 0600", info.Mode().Perm())
		}
	}

	reloaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.Profiles(); len(got) != 1 || got[0].Name != "测试" {
		t.Fatalf("reloaded profiles = %+v", got)
	}
	if reloaded.Settings().Theme != "dark" {
		t.Fatalf("settings not persisted")
	}
	if reloaded.Binary() == nil || reloaded.Binary().Version != "0.34.10" {
		t.Fatalf("binary not persisted")
	}
}

func TestStoreDeleteAndAdd(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	items, err := s.AddProfiles([]model.Profile{testProfile(), testProfile()})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID == items[1].ID {
		t.Fatalf("unexpected ids: %+v", items)
	}
	if err := s.DeleteProfile(items[0].ID); err != nil {
		t.Fatal(err)
	}
	if got := s.Profiles(); len(got) != 1 {
		t.Fatalf("profiles after delete = %d", len(got))
	}
}

func TestStoreCorruptProfiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "profiles.json"), []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Warnings()) == 0 {
		t.Fatalf("expected a corruption warning")
	}
	entries, _ := os.ReadDir(dir)
	found := false
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".corrupt-"+filepath.Ext(entry.Name()) {
			found = true
		}
		if len(entry.Name()) > len("profiles.json.corrupt-") && entry.Name()[:len("profiles.json.corrupt-")] == "profiles.json.corrupt-" {
			found = true
		}
	}
	if !found {
		t.Fatalf("corrupt file was not quarantined: %v", entries)
	}
}

func TestStoreDefaults(t *testing.T) {
	s, _ := New(t.TempDir())
	settings := s.Settings()
	if settings.MaxLogLines != 5000 || settings.Theme != "dark" || settings.DefaultLogLevel != "info" {
		t.Fatalf("unexpected defaults: %+v", settings)
	}
	data, err := json.Marshal(settings)
	if err != nil || len(data) == 0 {
		t.Fatalf("settings not marshalable: %v", err)
	}
}
