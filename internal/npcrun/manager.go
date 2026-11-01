package npcrun

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"npc-gui/internal/configgen"
	"npc-gui/internal/model"
	"npc-gui/internal/platform"
)

// Emitter receives Wails events.
type Emitter func(event string, payload any)

// Manager supervises one npc process per profile.
type Manager struct {
	mu            sync.Mutex
	runs          map[string]*instance
	dataDir       string
	binaryPath    string
	binaryVersion string
	maxLog        int
	logToDisk     bool
	emit          Emitter
	logger        *slog.Logger
}

type instance struct {
	m          *Manager
	profile    model.Profile
	cmd        *exec.Cmd
	state      *StateMachine
	ring       *Ring
	logCh      chan model.LogEntry
	brokerDone chan struct{}
	done       chan struct{}
	stopFlag   atomic.Bool
	mu         sync.Mutex
	finishOnce sync.Once
	configHash string
	configPath string
	logFile    *rotatingLog
	started    bool
}

type pidRecord struct {
	PID        int    `json:"pid"`
	ProfileID  string `json:"profileId"`
	ConfigPath string `json:"configPath"`
	BinaryPath string `json:"binaryPath"`
	StartedAt  int64  `json:"startedAt"`
}

// NewManager creates a process manager.
func NewManager(dataDir string, maxLog int, emit Emitter, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	return &Manager{
		runs:      map[string]*instance{},
		dataDir:   dataDir,
		maxLog:    maxLog,
		logToDisk: true,
		emit:      emit,
		logger:    logger,
	}
}

// SetBinary updates the npc executable used for new starts.
func (m *Manager) SetBinary(path, version string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.binaryPath = path
	m.binaryVersion = version
}

// SetLogToDisk controls whether captured logs are also written to disk.
func (m *Manager) SetLogToDisk(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logToDisk = enabled
}

// Start launches npc for a profile.
func (m *Manager) Start(ctx context.Context, p model.Profile, configPath, configHash string) model.OperationResult {
	m.mu.Lock()
	if existing, ok := m.runs[p.ID]; ok && existing.isRunning() {
		m.mu.Unlock()
		return model.FailResult("already_running", "该档案已经在运行")
	}
	if strings.TrimSpace(m.binaryPath) == "" {
		m.mu.Unlock()
		return model.FailResult("no_binary", "尚未配置 npc 二进制")
	}
	inst := &instance{
		m:          m,
		profile:    p,
		state:      NewStateMachine(),
		ring:       NewRing(m.maxLog),
		logCh:      make(chan model.LogEntry, 2048),
		brokerDone: make(chan struct{}),
		done:       make(chan struct{}),
		configHash: configHash,
		configPath: configPath,
	}
	m.runs[p.ID] = inst
	binPath := m.binaryPath
	binVersion := m.binaryVersion
	m.mu.Unlock()

	inst.mu.Lock()
	inst.state.PID = 0
	inst.state.State = model.StateStarting
	inst.started = true
	inst.mu.Unlock()
	m.emitState(inst)

	if err := platform.EnsureDir(filepath.Dir(configPath)); err != nil {
		inst.failStart(err)
		return model.FailResult("config_dir", "无法创建配置目录："+err.Error())
	}
	m.mu.Lock()
	logToDisk := m.logToDisk
	m.mu.Unlock()
	if logToDisk {
		logFile, err := newRotatingLog(filepath.Join(m.dataDir, "logs", p.ID), "session-"+time.Now().Format("20060102-150405")+".log", 10<<20, 3)
		if err != nil {
			m.logger.Warn("open log file", "profile", p.ID, "error", err)
		} else {
			inst.logFile = logFile
		}
	}

	args := configgen.BuildArgs(p, configPath, "")
	cmd := exec.Command(binPath, args...)
	cmd.Dir = filepath.Dir(configPath)
	cmd.Env = os.Environ()
	configureProcess(cmd)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		inst.failStart(err)
		return model.FailResult("pipe", "创建 stdout 管道失败："+err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		inst.failStart(err)
		return model.FailResult("pipe", "创建 stderr 管道失败："+err.Error())
	}
	if err := cmd.Start(); err != nil {
		inst.failStart(err)
		return model.FailResult("start", "启动 npc 失败："+err.Error())
	}

	inst.mu.Lock()
	inst.cmd = cmd
	inst.state.Start(cmd.Process.Pid, configHash, binVersion)
	inst.mu.Unlock()
	m.writePIDRecord(inst, binPath)
	m.emitState(inst)

	go inst.broker()
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		inst.readLines(stdout, "stdout")
	}()
	go func() {
		defer readers.Done()
		inst.readLines(stderr, "stderr")
	}()
	go func() {
		err := cmd.Wait()
		readers.Wait()
		close(inst.logCh)
		<-inst.brokerDone
		inst.finish(err)
	}()
	return model.OKResult()
}

func (inst *instance) failStart(err error) {
	inst.finishOnce.Do(func() {
		inst.mu.Lock()
		inst.state.State = model.StateError
		inst.state.LastError = err.Error()
		inst.mu.Unlock()
		if inst.logFile != nil {
			inst.logFile.Close()
		}
		inst.m.emitState(inst)
		close(inst.done)
	})
}

func (inst *instance) isRunning() bool {
	select {
	case <-inst.done:
		return false
	default:
		return inst.started
	}
}

func (inst *instance) readLines(reader io.Reader, stream string) {
	buf := bufio.NewReaderSize(reader, 64*1024)
	var partial strings.Builder
	for {
		line, isPrefix, err := buf.ReadLine()
		if len(line) > 0 {
			if partial.Len() < 64*1024 {
				partial.Write(line)
			}
			if !isPrefix {
				text := partial.String()
				partial.Reset()
				inst.handleLine(text, stream)
			}
		}
		if err != nil {
			if partial.Len() > 0 {
				inst.handleLine(partial.String(), stream)
			}
			return
		}
	}
}

func (inst *instance) handleLine(line, stream string) {
	entry := ParseLine(inst.profile.ID, line, stream, time.Now())
	entry = inst.ring.Add(entry)
	select {
	case inst.logCh <- entry:
	default:
	}

	inst.mu.Lock()
	changed, _ := inst.state.ObserveLine(entry.Level, entry.Text, entry.Time)
	inst.mu.Unlock()
	if changed {
		inst.m.emitState(inst)
	}
}

func (inst *instance) broker() {
	defer close(inst.brokerDone)
	ticker := time.NewTicker(120 * time.Millisecond)
	defer ticker.Stop()
	var batch []model.LogEntry
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if inst.logFile != nil {
			for _, entry := range batch {
				inst.logFile.WriteLine(logLine(entry))
			}
		}
		if inst.m.emit != nil {
			inst.m.emit("runtime:logs", model.LogPage{
				ProfileID: inst.profile.ID,
				Entries:   append([]model.LogEntry(nil), batch...),
				LastSeq:   batch[len(batch)-1].Seq,
			})
		}
		batch = batch[:0]
	}
	for {
		select {
		case entry, ok := <-inst.logCh:
			if !ok {
				flush()
				return
			}
			batch = append(batch, entry)
			if len(batch) >= 200 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (inst *instance) finish(waitErr error) {
	inst.finishOnce.Do(func() {
		userStop := inst.stopFlag.Load()
		inst.mu.Lock()
		code := 0
		message := ""
		if inst.cmd != nil && inst.cmd.ProcessState != nil {
			code = inst.cmd.ProcessState.ExitCode()
		}
		if waitErr != nil && !userStop && !errors.Is(waitErr, exec.ErrWaitDelay) {
			message = "npc 进程退出：" + waitErr.Error()
		}
		info := inst.state.Exit(userStop, code, message)
		info.ProfileID = inst.profile.ID
		inst.mu.Unlock()
		if inst.logFile != nil {
			inst.logFile.Close()
		}
		inst.m.removePIDRecord(inst.profile.ID)
		if inst.m.emit != nil {
			inst.m.emit("runtime:state", info)
		}
		close(inst.done)
	})
}

// Stop terminates one profile process.
func (m *Manager) Stop(p model.Profile) model.OperationResult {
	m.mu.Lock()
	inst, ok := m.runs[p.ID]
	m.mu.Unlock()
	if !ok || !inst.isRunning() {
		return model.OKResult()
	}
	inst.stopFlag.Store(true)
	inst.mu.Lock()
	info := inst.state.Stop()
	cmd := inst.cmd
	inst.mu.Unlock()
	info.ProfileID = p.ID
	if m.emit != nil {
		m.emit("runtime:state", info)
	}
	if cmd != nil {
		_ = terminateProcess(cmd)
	}
	select {
	case <-inst.done:
		return model.OKResult()
	case <-time.After(5 * time.Second):
		_ = killProcess(cmd)
	}
	select {
	case <-inst.done:
		return model.OKResult()
	case <-time.After(3 * time.Second):
		return model.FailResult("stop_timeout", "npc 进程未能在超时内退出，请手动结束 PID")
	}
}

// StopAll terminates every running profile.
func (m *Manager) StopAll(profiles []model.Profile) []model.OperationResult {
	out := make([]model.OperationResult, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, m.Stop(p))
	}
	return out
}

// Close stops every managed process.
func (m *Manager) Close() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.runs))
	for id := range m.runs {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.mu.Lock()
		inst := m.runs[id]
		m.mu.Unlock()
		if inst != nil {
			_ = m.Stop(inst.profile)
		}
	}
}

// Get returns a runtime snapshot for one profile.
func (m *Manager) Get(id string) model.RuntimeInfo {
	m.mu.Lock()
	inst := m.runs[id]
	m.mu.Unlock()
	if inst == nil {
		return model.RuntimeInfo{ProfileID: id, State: model.StateStopped}
	}
	inst.mu.Lock()
	defer inst.mu.Unlock()
	return inst.state.Snapshot(id)
}

// Forget removes a profile from the runtime map after deletion.
func (m *Manager) Forget(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.runs, id)
}

// RunningCount returns the number of processes that are not stopped.
func (m *Manager) RunningCount() int {
	m.mu.Lock()
	ids := make([]string, 0, len(m.runs))
	for id := range m.runs {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	count := 0
	for _, id := range ids {
		info := m.Get(id)
		switch info.State {
		case model.StateStarting, model.StateRunning, model.StateOnline, model.StateReconnecting, model.StateStopping:
			count++
		case model.StateError:
			if info.PID > 0 {
				count++
			}
		}
	}
	return count
}

// List returns runtime snapshots for all known profiles.
func (m *Manager) List() []model.RuntimeInfo {
	m.mu.Lock()
	ids := make([]string, 0, len(m.runs))
	for id := range m.runs {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	out := make([]model.RuntimeInfo, 0, len(ids))
	for _, id := range ids {
		out = append(out, m.Get(id))
	}
	return out
}

// Logs returns buffered log entries for a profile.
func (m *Manager) Logs(id string, after uint64, limit int) model.LogPage {
	m.mu.Lock()
	inst := m.runs[id]
	m.mu.Unlock()
	if inst == nil {
		return model.LogPage{ProfileID: id, Entries: []model.LogEntry{}}
	}
	return inst.ring.After(id, after, limit)
}

// ClearLogs empties the in-memory buffer for a profile.
func (m *Manager) ClearLogs(id string) {
	m.mu.Lock()
	inst := m.runs[id]
	m.mu.Unlock()
	if inst != nil {
		inst.ring.Reset()
	}
}

func (m *Manager) emitState(inst *instance) {
	if m.emit == nil {
		return
	}
	inst.mu.Lock()
	info := inst.state.Snapshot(inst.profile.ID)
	inst.mu.Unlock()
	m.emit("runtime:state", info)
}

func (m *Manager) writePIDRecord(inst *instance, binPath string) {
	rec := pidRecord{
		PID:        inst.cmd.Process.Pid,
		ProfileID:  inst.profile.ID,
		ConfigPath: inst.configPath,
		BinaryPath: binPath,
		StartedAt:  time.Now().UnixMilli(),
	}
	data, _ := json.MarshalIndent(rec, "", "  ")
	_ = platform.AtomicWriteFile(filepath.Join(m.dataDir, "runtime", inst.profile.ID+".json"), data, 0o600)
}

func (m *Manager) removePIDRecord(id string) {
	_ = os.Remove(filepath.Join(m.dataDir, "runtime", id+".json"))
}

var (
	staleMatch     = processMatchesStale
	staleTerminate = terminateStale
	staleKill      = killStale
)

func processMatchesStale(pid int, binaryPath, configPath string) bool {
	cmdLine := strings.ToLower(processCommandLine(pid))
	if cmdLine == "" {
		return false
	}
	if configPath != "" {
		return strings.Contains(cmdLine, strings.ToLower(configPath))
	}
	return binaryPath != "" && strings.Contains(cmdLine, strings.ToLower(binaryPath))
}

func waitProcessGone(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return !processAlive(pid)
}

// CleanupStale stops npc processes left over from a previous GUI run.
// It only terminates a process when its command line still points at the
// profile config recorded in runtime/<id>.json.
func (m *Manager) CleanupStale() (cleaned []string, warnings []string) {
	dir := filepath.Join(m.dataDir, "runtime")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var rec pidRecord
		if err := json.Unmarshal(data, &rec); err != nil {
			_ = os.Remove(path)
			continue
		}
		if !processAlive(rec.PID) {
			_ = os.Remove(path)
			continue
		}
		if !staleMatch(rec.PID, rec.BinaryPath, rec.ConfigPath) {
			warnings = append(warnings, fmt.Sprintf("发现疑似残留进程 PID %d（档案 %s），但无法确认其命令行，未自动结束", rec.PID, rec.ProfileID))
			continue
		}
		_ = staleTerminate(rec.PID)
		if !waitProcessGone(rec.PID, 3*time.Second) {
			_ = staleKill(rec.PID)
			if !waitProcessGone(rec.PID, 2*time.Second) {
				warnings = append(warnings, fmt.Sprintf("残留 npc 进程 PID %d（档案 %s）未能自动结束，请手动处理", rec.PID, rec.ProfileID))
				continue
			}
		}
		_ = os.Remove(path)
		cleaned = append(cleaned, fmt.Sprintf("已停止上次残留的 npc 进程 PID %d（档案 %s）", rec.PID, rec.ProfileID))
	}
	return cleaned, warnings
}
