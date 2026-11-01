//go:build !windows

package npcrun

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"npc-gui/internal/model"
)

func TestManagerFakeProcess(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-npc.sh")
	body := "#!/bin/sh\necho '2026-10-31T12:00:00+08:00 INF Loading configuration successfully'\necho '2026-10-31T12:00:01+08:00 INF Successful connection with server test:1'\nsleep 20\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "profiles", "p1", "npc.conf")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte("[common]\nserver_addr=test:1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var states []string
	m := NewManager(dir, 100, func(event string, payload any) {
		if event == "runtime:state" {
			info := payload.(model.RuntimeInfo)
			mu.Lock()
			states = append(states, info.State)
			mu.Unlock()
		}
	}, nil)
	m.SetBinary(script, "0.0.1")
	profile := model.NewProfile()
	profile.ID = "p1"
	profile.Name = "fake"
	profile.Server = "test:1"
	profile.VKey = "k"

	if res := m.Start(context.Background(), profile, configPath, "hash"); !res.OK {
		t.Fatalf("Start() = %+v", res)
	}
	deadline := time.Now().Add(4 * time.Second)
	var online bool
	for time.Now().Before(deadline) {
		info := m.Get("p1")
		if info.State == model.StateOnline {
			online = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !online {
		t.Fatalf("process never became online, state=%+v", m.Get("p1"))
	}
	page := m.Logs("p1", 0, 100)
	if len(page.Entries) < 2 {
		t.Fatalf("logs = %+v", page)
	}
	found := false
	for _, entry := range page.Entries {
		if strings.Contains(entry.Text, "Successful connection") {
			found = true
		}
	}
	if !found {
		t.Fatalf("online log not captured: %+v", page.Entries)
	}

	if res := m.Stop(profile); !res.OK {
		t.Fatalf("Stop() = %+v", res)
	}
	info := m.Get("p1")
	if info.State != model.StateStopped {
		t.Fatalf("state after stop = %+v", info)
	}
	if info.PID != 0 {
		t.Fatalf("pid after stop = %d", info.PID)
	}
	_ = m.Stop(profile)
	m.Close()
}
func writeStaleRecord(t *testing.T, dir string, rec pidRecord) string {
	t.Helper()
	runtimeDir := filepath.Join(dir, "runtime")
	if err := os.MkdirAll(runtimeDir, 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(runtimeDir, rec.ProfileID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCleanupStaleKillsOrphan(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = cmd.Wait() }()
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	recordPath := writeStaleRecord(t, dir, pidRecord{
		PID:        cmd.Process.Pid,
		ProfileID:  "orphan",
		ConfigPath: filepath.Join(dir, "profiles", "orphan", "npc.conf"),
		BinaryPath: "/bin/sleep",
	})
	oldMatch := staleMatch
	staleMatch = func(pid int, binaryPath, configPath string) bool {
		return binaryPath == "/bin/sleep"
	}
	defer func() { staleMatch = oldMatch }()

	m := NewManager(dir, 100, nil, nil)
	cleaned, warnings := m.CleanupStale()
	if len(cleaned) != 1 {
		t.Fatalf("cleaned = %v, warnings = %v", cleaned, warnings)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if processAlive(cmd.Process.Pid) {
		t.Fatalf("orphan process still alive")
	}
	if _, err := os.Stat(recordPath); !os.IsNotExist(err) {
		t.Fatalf("runtime record was not removed: %v", err)
	}
}

func TestCleanupStaleSkipsUnverifiedProcess(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go func() { _ = cmd.Wait() }()
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	_ = writeStaleRecord(t, dir, pidRecord{
		PID:        cmd.Process.Pid,
		ProfileID:  "not-ours",
		ConfigPath: filepath.Join(dir, "profiles", "not-ours", "npc.conf"),
		BinaryPath: "/bin/sleep",
	})
	oldMatch := staleMatch
	staleMatch = func(pid int, binaryPath, configPath string) bool { return false }
	defer func() { staleMatch = oldMatch }()

	m := NewManager(dir, 100, nil, nil)
	cleaned, warnings := m.CleanupStale()
	if len(cleaned) != 0 || len(warnings) != 1 {
		t.Fatalf("cleaned = %v, warnings = %v", cleaned, warnings)
	}
	if !processAlive(cmd.Process.Pid) {
		t.Fatalf("unverified process should not be killed")
	}
}
