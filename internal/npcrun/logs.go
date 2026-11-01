package npcrun

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"npc-gui/internal/model"
	"npc-gui/internal/platform"
)

// Ring is a bounded log buffer with sequence numbers.
type Ring struct {
	mu        sync.RWMutex
	entries   []model.LogEntry
	max       int
	seq       uint64
	truncated bool
}

// NewRing creates a ring holding at most max entries.
func NewRing(max int) *Ring {
	if max <= 0 {
		max = 5000
	}
	return &Ring{max: max}
}

// Add appends one log entry and returns it with a sequence number.
func (r *Ring) Add(entry model.LogEntry) model.LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	entry.Seq = r.seq
	if r.max <= 0 {
		return entry
	}
	if len(r.entries) >= r.max {
		copy(r.entries, r.entries[1:])
		r.entries[len(r.entries)-1] = entry
		r.truncated = true
	} else {
		r.entries = append(r.entries, entry)
	}
	return entry
}

// After returns entries newer than seq, up to limit.
func (r *Ring) After(profileID string, seq uint64, limit int) model.LogPage {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 || limit > r.max {
		limit = r.max
	}
	start := 0
	for i := range r.entries {
		if r.entries[i].Seq > seq {
			start = i
			break
		}
		start = len(r.entries)
	}
	entries := append([]model.LogEntry(nil), r.entries[start:]...)
	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}
	var last uint64
	if len(r.entries) > 0 {
		last = r.entries[len(r.entries)-1].Seq
	}
	return model.LogPage{ProfileID: profileID, Entries: entries, LastSeq: last, Truncated: r.truncated}
}

// Reset clears the ring for a new session.
func (r *Ring) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = nil
	r.seq = 0
	r.truncated = false
}

// rotatingLog writes captured lines to a size-limited file.
type rotatingLog struct {
	mu       sync.Mutex
	dir      string
	name     string
	maxBytes int64
	maxFiles int
	file     *os.File
	size     int64
}

func newRotatingLog(dir, name string, maxBytes int64, maxFiles int) (*rotatingLog, error) {
	if err := platform.EnsureDir(dir); err != nil {
		return nil, err
	}
	r := &rotatingLog{dir: dir, name: name, maxBytes: maxBytes, maxFiles: maxFiles}
	if err := r.open(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *rotatingLog) open() error {
	path := filepath.Join(r.dir, r.name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	info, _ := f.Stat()
	r.file = f
	if info != nil {
		r.size = info.Size()
	}
	return nil
}

func (r *rotatingLog) WriteLine(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return
	}
	if r.size+int64(len(line)) > r.maxBytes {
		r.rotateLocked()
	}
	if r.file == nil {
		return
	}
	n, _ := r.file.WriteString(line)
	r.size += int64(n)
}

func (r *rotatingLog) rotateLocked() {
	if r.file != nil {
		_ = r.file.Sync()
		_ = r.file.Close()
		r.file = nil
	}
	base := filepath.Join(r.dir, r.name)
	for i := r.maxFiles - 1; i >= 1; i-- {
		old := fmt.Sprintf("%s.%d", base, i)
		next := fmt.Sprintf("%s.%d", base, i+1)
		if i == r.maxFiles-1 {
			_ = os.Remove(old)
		}
		_ = os.Rename(old, next)
	}
	_ = os.Rename(base, base+".1")
	r.size = 0
	_ = r.open()
}

func (r *rotatingLog) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file != nil {
		_ = r.file.Sync()
		_ = r.file.Close()
		r.file = nil
	}
}

func logLine(entry model.LogEntry) string {
	ts := time.UnixMilli(entry.Time).Format("2006-01-02 15:04:05.000")
	return fmt.Sprintf("%s %-5s %s\n", ts, entry.Level, entry.Text)
}
