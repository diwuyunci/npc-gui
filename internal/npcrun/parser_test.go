package npcrun

import (
	"testing"
	"time"

	"npc-gui/internal/model"
)

func TestParseLine(t *testing.T) {
	now := time.Date(2026, 10, 31, 12, 0, 0, 0, time.Local)
	entry := ParseLine("p1", "2026-10-31T11:59:58+08:00 INF Successful connection with server a:1", "stdout", now)
	if entry.Level != "info" {
		t.Fatalf("level = %q", entry.Level)
	}
	if entry.Text != "Successful connection with server a:1" {
		t.Fatalf("text = %q", entry.Text)
	}
	if entry.Time != time.Date(2026, 10, 31, 3, 59, 58, 0, time.UTC).UnixMilli() && entry.Time == 0 {
		t.Fatalf("missing timestamp")
	}
	if entry.ProfileID != "p1" || entry.Stream != "stdout" {
		t.Fatalf("metadata = %+v", entry)
	}
}

func TestParseLineFallback(t *testing.T) {
	entry := ParseLine("p1", "\x1b[31mplain message\x1b[0m", "stderr", time.Now())
	if entry.Level != "info" || entry.Text != "plain message" {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestStateMachineTransitions(t *testing.T) {
	sm := NewStateMachine()
	info := sm.Start(100, "hash", "0.34.10")
	if info.State != model.StateRunning || info.PID != 100 {
		t.Fatalf("start info = %+v", info)
	}
	changed, info := sm.ObserveLine("info", "Successful connection with server a:1", 1)
	if !changed || info.State != model.StateOnline || info.OnlineSince != 1 {
		t.Fatalf("online info = %+v changed=%v", info, changed)
	}
	changed, info = sm.ObserveLine("error", "The connection server failed, reconnecting", 2)
	if !changed || info.State != model.StateReconnecting {
		t.Fatalf("reconnecting info = %+v changed=%v", info, changed)
	}
	changed, info = sm.ObserveLine("info", "Successful connection with server a:1", 3)
	if !changed || info.State != model.StateOnline || info.Reconnects != 1 {
		t.Fatalf("re-online info = %+v changed=%v", info, changed)
	}
	info = sm.Exit(false, 1, "boom")
	if info.State != model.StateError || info.PID != 0 || info.LastError != "boom" {
		t.Fatalf("exit info = %+v", info)
	}
}

func TestStateMachineFatal(t *testing.T) {
	sm := NewStateMachine()
	sm.Start(1, "", "")
	changed, info := sm.ObserveLine("error", "Config file loading error: missing [common]", 1)
	if !changed || info.State != model.StateError {
		t.Fatalf("fatal info = %+v", info)
	}
}

func TestRing(t *testing.T) {
	r := NewRing(3)
	for i := 0; i < 5; i++ {
		r.Add(model.LogEntry{Text: "line", Time: int64(i)})
	}
	page := r.After("p", 0, 10)
	if len(page.Entries) != 3 || !page.Truncated {
		t.Fatalf("page = %+v", page)
	}
	page = r.After("p", 3, 10)
	if len(page.Entries) != 2 || page.Entries[0].Seq != 4 {
		t.Fatalf("page after 3 = %+v", page)
	}
	r.Reset()
	if len(r.After("p", 0, 10).Entries) != 0 {
		t.Fatalf("reset failed")
	}
}
