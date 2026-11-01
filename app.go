package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/options"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"npc-gui/internal/autostart"
	"npc-gui/internal/buildinfo"
	"npc-gui/internal/configgen"
	"npc-gui/internal/model"
	"npc-gui/internal/npcbin"
	"npc-gui/internal/npcrun"
	"npc-gui/internal/platform"
	"npc-gui/internal/store"
	"npc-gui/internal/tray"
	"npc-gui/internal/update"
)

// App is the Wails-bound facade. Only exported methods are exposed to the frontend.
type App struct {
	ctx context.Context

	mu          sync.RWMutex
	dataDir     string
	store       *store.Store
	manager     *npcrun.Manager
	npcBin      *npcbin.Manager
	updater     *update.Manager
	logger      *slog.Logger
	logFile     *os.File
	binary      *model.BinaryInfo
	candidates  []model.BinaryInfo
	warnings    []string
	initErr     string
	quitting    bool
	autoStart   bool
	detecting   bool
	startHidden bool
	npcCancel   context.CancelFunc
	tray        *tray.Tray
	notifier    *notifier
}

// NewApp creates the application facade.
func NewApp() *App {
	return &App{notifier: &notifier{}}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	dir, err := platform.DataDir()
	if err != nil {
		a.setInitError("无法确定数据目录：" + err.Error())
		return
	}
	a.dataDir = dir
	if err := platform.EnsureSubdirs(dir); err != nil {
		a.setInitError("无法创建数据目录：" + err.Error())
		return
	}
	a.setupLogger()
	st, err := store.New(dir)
	if err != nil {
		a.setInitError("加载本地配置失败：" + err.Error())
		return
	}
	a.store = st
	a.warnings = append(a.warnings, st.Warnings()...)
	settings := st.Settings()
	a.manager = npcrun.NewManager(dir, settings.MaxLogLines, a.emit, a.logger)
	a.manager.SetLogToDisk(settings.LogToDisk)
	a.npcBin = npcbin.NewManager(dir)
	a.updater = update.New(dir, a.emit, a.logger)
	a.updater.SetMirror(settings.DownloadMirror)
	if b := st.Binary(); b != nil {
		a.binary = b
		if _, err := os.Stat(b.Path); err == nil {
			a.manager.SetBinary(b.Path, b.Version)
		}
	}
	cleaned, staleWarnings := a.manager.CleanupStale()
	a.warnings = append(a.warnings, staleWarnings...)
	for _, message := range cleaned {
		a.logger.Info("stale npc process cleaned", "message", message)
	}
	a.syncAutostart(settings)
	if settings.TrayEnabled {
		a.startTray()
	}
	a.mu.RLock()
	startHidden := a.startHidden
	a.mu.RUnlock()
	if startHidden {
		if !settings.TrayEnabled {
			wailsRuntime.WindowShow(ctx)
		} else {
			go func() {
				time.Sleep(4 * time.Second)
				if !a.trayActive() {
					a.ShowWindow()
				}
			}()
		}
	}
	go a.bootstrapAsync()
}

func (a *App) setInitError(msg string) {
	a.mu.Lock()
	a.initErr = msg
	a.mu.Unlock()
	a.emit("app:notice", map[string]string{"level": "error", "message": msg})
}

func (a *App) setupLogger() {
	path := filepath.Join(a.dataDir, "logs", "app.log")
	if err := platform.EnsureDir(filepath.Dir(path)); err != nil {
		a.logger = slog.Default()
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		a.logger = slog.Default()
		return
	}
	a.logFile = f
	a.logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(a.logger)
}

func (a *App) baseContext() context.Context {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

func (a *App) emit(event string, payload any) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx == nil {
		return
	}
	a.notifyEvent(event, payload)
	if event == "runtime:state" || event == "update:state" {
		a.updateTray()
	}
	wailsRuntime.EventsEmit(ctx, event, payload)
}

func (a *App) bootstrapAsync() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	a.detectBinary(ctx)
	a.startAutoProfiles()
	a.checkUpdateIfStale(ctx)
}

func (a *App) currentSettings() model.Settings {
	if a.store == nil {
		return model.DefaultSettings()
	}
	return a.store.Settings()
}

func (a *App) currentBinary() *model.BinaryInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.binary == nil {
		return nil
	}
	cp := *a.binary
	return &cp
}

func (a *App) detectBinary(ctx context.Context) BinaryResult {
	a.mu.Lock()
	if a.detecting {
		a.mu.Unlock()
		return BinaryResult{Result: model.FailResult("detecting", "正在检测 npc 二进制")}
	}
	a.detecting = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.detecting = false
		a.mu.Unlock()
	}()

	if a.npcBin == nil {
		return BinaryResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	settings := a.currentSettings()
	candidates := a.npcBin.Detect(ctx, settings.NpcBinaryPath)
	var chosen *model.BinaryInfo
	a.mu.RLock()
	previous := a.binary
	a.mu.RUnlock()
	if previous != nil {
		for i := range candidates {
			if candidates[i].Path == previous.Path {
				c := candidates[i]
				chosen = &c
				break
			}
		}
	}
	if chosen == nil && len(candidates) > 0 {
		c := candidates[0]
		chosen = &c
	}
	a.mu.Lock()
	a.candidates = candidates
	if chosen != nil {
		a.binary = chosen
	}
	a.mu.Unlock()
	if chosen != nil {
		if a.manager != nil {
			a.manager.SetBinary(chosen.Path, chosen.Version)
		}
		if a.store != nil {
			_ = a.store.SetBinary(chosen)
			if settings.NpcBinaryPath != chosen.Path {
				settings.NpcBinaryPath = chosen.Path
				_, _ = a.store.SaveSettings(settings)
			}
		}
	}
	a.emit("binary:info", a.binaryState())
	if chosen == nil {
		a.emit("app:notice", map[string]string{"level": "warn", "message": "尚未找到 npc，请在设置中下载或选择本地文件"})
		return BinaryResult{Candidates: candidates, Result: model.FailResult("no_binary", "尚未找到 npc")}
	}
	return BinaryResult{Binary: chosen, Candidates: candidates, Result: model.OKResult()}
}

func (a *App) binaryState() BinaryResult {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var b *model.BinaryInfo
	if a.binary != nil {
		cp := *a.binary
		b = &cp
	}
	candidates := append([]model.BinaryInfo(nil), a.candidates...)
	return BinaryResult{Binary: b, Candidates: candidates, Result: model.OKResult()}
}

func (a *App) binaryCandidates() []model.BinaryInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]model.BinaryInfo(nil), a.candidates...)
}

func (a *App) startAutoProfiles() {
	a.mu.Lock()
	if a.autoStart || a.quitting {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()
	if a.currentBinary() == nil || a.store == nil {
		return
	}
	a.mu.Lock()
	a.autoStart = true
	a.mu.Unlock()
	profiles := a.store.Profiles()
	go func() {
		for _, p := range profiles {
			if !p.AutoStart {
				continue
			}
			if a.isQuitting() {
				return
			}
			if res := a.startProfile(p.ID); !res.OK {
				a.emit("app:notice", map[string]string{"level": "error", "message": p.DisplayName() + " 自动启动失败：" + res.Message})
			}
			time.Sleep(250 * time.Millisecond)
		}
	}()
}

func (a *App) checkUpdateIfStale(ctx context.Context) {
	settings := a.currentSettings()
	if !settings.AutoCheckUpdates || a.updater == nil || !a.updater.Enabled() {
		return
	}
	state := a.updater.State()
	if state.CheckedAt > 0 && time.Since(time.UnixMilli(state.CheckedAt)) < 24*time.Hour {
		return
	}
	checkCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if updated, err := a.updater.Check(checkCtx, false); err == nil && updated.CheckedAt > 0 {
		if a.store != nil {
			settings.LastUpdateCheck = updated.CheckedAt
			_, _ = a.store.SaveSettings(settings)
		}
	}
}

func (a *App) isQuitting() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.quitting
}

// GetBootstrap returns the complete initial state.
func (a *App) GetBootstrap() model.Bootstrap {
	settings := a.currentSettings()
	profiles := []model.Profile{}
	if a.store != nil {
		profiles = a.store.Profiles()
	}
	runtimes := make([]model.RuntimeInfo, 0, len(profiles))
	known := map[string]model.RuntimeInfo{}
	if a.manager != nil {
		for _, info := range a.manager.List() {
			known[info.ProfileID] = info
		}
	}
	for _, p := range profiles {
		info, ok := known[p.ID]
		if !ok && a.manager != nil {
			info = a.manager.Get(p.ID)
		}
		if !ok {
			info = model.RuntimeInfo{ProfileID: p.ID, State: model.StateStopped}
		}
		if info.State != model.StateStopped && info.ConfigHash != "" {
			if content, err := configgen.Generate(p); err == nil {
				info.PendingRestart = configgen.ConfigHash(content) != info.ConfigHash
			}
		}
		runtimes = append(runtimes, info)
	}
	binary := a.currentBinary()
	a.mu.RLock()
	candidates := append([]model.BinaryInfo(nil), a.candidates...)
	warnings := append([]string(nil), a.warnings...)
	initErr := a.initErr
	a.mu.RUnlock()
	if initErr != "" {
		warnings = append(warnings, initErr)
	}
	upd := model.UpdateStateInfo{State: model.UpdateIdle, CurrentVersion: buildinfo.Version}
	if a.updater != nil {
		upd = a.updater.State()
	}
	if candidates == nil {
		candidates = []model.BinaryInfo{}
	}
	firstRun := len(profiles) == 0 || binary == nil
	return model.Bootstrap{
		App: model.AppInfo{
			ProductName:   model.AppName,
			Version:       buildinfo.Version,
			Commit:        buildinfo.Commit,
			BuildDate:     buildinfo.BuildDate,
			Repo:          buildinfo.Repo,
			OS:            runtime.GOOS,
			Arch:          runtime.GOARCH,
			UpdateEnabled: a.updater != nil && a.updater.Enabled(),
			DataDir:       a.dataDir,
		},
		Profiles:         profiles,
		Runtimes:         runtimes,
		Settings:         settings,
		Binary:           binary,
		BinaryCandidates: candidates,
		Update:           upd,
		Warnings:         warnings,
		FirstRun:         firstRun,
	}
}

// SaveProfile validates, persists and regenerates a profile.
func (a *App) SaveProfile(p model.Profile) ProfileSaveResult {
	if a.store == nil {
		return ProfileSaveResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Server = strings.TrimSpace(p.Server)
	p.VKey = strings.TrimSpace(p.VKey)
	p.ConnType = strings.ToLower(strings.TrimSpace(p.ConnType))
	if p.LogLevel == "" {
		p.LogLevel = a.currentSettings().DefaultLogLevel
	}
	if res := configgen.Validate(p); !res.OK {
		return ProfileSaveResult{Result: res}
	}
	saved, err := a.store.SaveProfile(p)
	if err != nil {
		return ProfileSaveResult{Result: model.FailResult("save_failed", "保存档案失败："+err.Error())}
	}
	content, err := configgen.Generate(saved)
	if err != nil {
		return ProfileSaveResult{Profile: saved, Result: model.FailResult("config_failed", "生成配置失败："+err.Error())}
	}
	configPath := a.store.ConfigPath(saved.ID)
	if err := configgen.WriteConfig(configPath, content); err != nil {
		return ProfileSaveResult{Profile: saved, Result: model.FailResult("config_write", "写入配置失败："+err.Error())}
	}
	pending := false
	if a.manager != nil {
		info := a.manager.Get(saved.ID)
		if info.State != model.StateStopped && info.ConfigHash != "" {
			pending = info.ConfigHash != configgen.ConfigHash(content)
		}
	}
	a.emit("profiles:changed", nil)
	return ProfileSaveResult{Profile: saved, PendingRestart: pending, Result: model.OKResult()}
}

// DeleteProfile removes a stopped profile and its generated files.
func (a *App) DeleteProfile(id string) model.OperationResult {
	if a.store == nil {
		return model.FailResult("not_ready", "应用尚未初始化")
	}
	if _, ok := a.store.GetProfile(id); !ok {
		return model.FailResult("not_found", "档案不存在")
	}
	if a.manager != nil {
		info := a.manager.Get(id)
		switch info.State {
		case model.StateStarting, model.StateRunning, model.StateOnline, model.StateReconnecting, model.StateStopping:
			return model.FailResult("running", "请先停止该档案再删除")
		}
		if info.PID > 0 {
			if res := a.manager.Stop(model.Profile{ID: id}); !res.OK {
				return res
			}
		}
	}
	if err := a.store.DeleteProfile(id); err != nil {
		return model.FailResult("delete_failed", "删除档案失败："+err.Error())
	}
	if a.manager != nil {
		a.manager.Forget(id)
	}
	_ = os.RemoveAll(a.store.ProfileDir(id))
	a.emit("profiles:changed", nil)
	return model.OKResult()
}

// DuplicateProfile creates a copy of an existing profile.
func (a *App) DuplicateProfile(id string) ProfileSaveResult {
	if a.store == nil {
		return ProfileSaveResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	p, ok := a.store.GetProfile(id)
	if !ok {
		return ProfileSaveResult{Result: model.FailResult("not_found", "档案不存在")}
	}
	p.ID = ""
	p.Name = p.DisplayName() + " 副本"
	p.CreatedAt = 0
	p.UpdatedAt = 0
	p.AutoStart = false
	return a.SaveProfile(p)
}

// ImportCommand parses a pasted NPS launch command.
func (a *App) ImportCommand(raw string) model.ImportPreview {
	return configgen.ParseCommand(raw)
}

// CommitImport persists selected profiles from an import preview.
func (a *App) CommitImport(profiles []model.Profile) ProfilesSaveResult {
	if a.store == nil {
		return ProfilesSaveResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	if len(profiles) == 0 {
		return ProfilesSaveResult{Profiles: []model.Profile{}, Result: model.FailResult("empty", "没有选择任何档案")}
	}
	valid := make([]model.Profile, 0, len(profiles))
	warnings := []string{}
	for i := range profiles {
		p := profiles[i]
		p.Name = strings.TrimSpace(p.Name)
		p.Server = strings.TrimSpace(p.Server)
		p.VKey = strings.TrimSpace(p.VKey)
		p.ConnType = strings.ToLower(strings.TrimSpace(p.ConnType))
		if p.LogLevel == "" {
			p.LogLevel = a.currentSettings().DefaultLogLevel
		}
		res := configgen.Validate(p)
		if !res.OK {
			return ProfilesSaveResult{Profiles: []model.Profile{}, Result: model.FailResult("validation", fmt.Sprintf("%s 校验失败：%s", p.DisplayName(), res.Message))}
		}
		warnings = append(warnings, res.Warnings...)
		valid = append(valid, p)
	}
	saved, err := a.store.AddProfiles(valid)
	if err != nil {
		return ProfilesSaveResult{Profiles: []model.Profile{}, Result: model.FailResult("save_failed", "保存档案失败："+err.Error())}
	}
	for _, p := range saved {
		content, err := configgen.Generate(p)
		if err != nil {
			continue
		}
		_ = configgen.WriteConfig(a.store.ConfigPath(p.ID), content)
	}
	a.emit("profiles:changed", nil)
	return ProfilesSaveResult{Profiles: saved, Result: model.OKResult(warnings...)}
}

// ImportProfiles opens a JSON file and adds its profiles.
func (a *App) ImportProfiles() ProfilesSaveResult {
	if a.store == nil {
		return ProfilesSaveResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	path, err := wailsRuntime.OpenFileDialog(a.baseContext(), wailsRuntime.OpenDialogOptions{
		Title:            "导入连接档案",
		DefaultDirectory: a.dataDir,
		Filters:          []wailsRuntime.FileFilter{{DisplayName: "JSON 文件", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return ProfilesSaveResult{Result: model.FailResult("cancel", "已取消导入")}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ProfilesSaveResult{Result: model.FailResult("read", "读取文件失败："+err.Error())}
	}
	var envelope ProfileExport
	if err := json.Unmarshal(data, &envelope); err != nil {
		return ProfilesSaveResult{Result: model.FailResult("parse", "解析 JSON 失败："+err.Error())}
	}
	if envelope.SchemaVersion > 1 {
		return ProfilesSaveResult{Result: model.FailResult("schema", "文件版本高于当前应用，请升级应用")}
	}
	if len(envelope.Profiles) == 0 {
		return ProfilesSaveResult{Result: model.FailResult("empty", "文件中没有连接档案")}
	}
	warnings := []string{}
	valid := make([]model.Profile, 0, len(envelope.Profiles))
	for i := range envelope.Profiles {
		p := envelope.Profiles[i]
		if strings.TrimSpace(p.VKey) == "" {
			p.VKey = "__missing__"
			warnings = append(warnings, p.DisplayName()+" 缺少密钥，导入后需要补填")
		}
		res := configgen.Validate(p)
		if !res.OK {
			return ProfilesSaveResult{Result: model.FailResult("validation", fmt.Sprintf("%s 校验失败：%s", p.DisplayName(), res.Message))}
		}
		p.VKey = strings.TrimSpace(envelope.Profiles[i].VKey)
		p.ID = ""
		valid = append(valid, p)
	}
	saved, err := a.store.AddProfiles(valid)
	if err != nil {
		return ProfilesSaveResult{Result: model.FailResult("save_failed", "保存档案失败："+err.Error())}
	}
	a.emit("profiles:changed", nil)
	return ProfilesSaveResult{Profiles: saved, Result: model.OKResult(warnings...)}
}

// ExportProfiles writes selected profiles to a JSON file.
func (a *App) ExportProfiles(ids []string, includeSecrets bool) StringResult {
	if a.store == nil {
		return StringResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	all := a.store.Profiles()
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	out := make([]model.Profile, 0, len(ids))
	for _, p := range all {
		if len(want) > 0 && !want[p.ID] {
			continue
		}
		if !includeSecrets {
			p.VKey = ""
			p.BasicPassword = ""
			p.WebPassword = ""
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return StringResult{Result: model.FailResult("empty", "没有可导出的档案")}
	}
	filename := "npc-gui-profiles-" + time.Now().Format("20060102") + ".json"
	path, err := wailsRuntime.SaveFileDialog(a.baseContext(), wailsRuntime.SaveDialogOptions{
		Title:            "导出连接档案",
		DefaultDirectory: a.dataDir,
		DefaultFilename:  filename,
		Filters:          []wailsRuntime.FileFilter{{DisplayName: "JSON 文件", Pattern: "*.json"}},
	})
	if err != nil || path == "" {
		return StringResult{Result: model.FailResult("cancel", "已取消导出")}
	}
	envelope := ProfileExport{
		SchemaVersion:  1,
		ExportedAt:     time.Now().UnixMilli(),
		IncludeSecrets: includeSecrets,
		Profiles:       out,
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return StringResult{Result: model.FailResult("marshal", "生成 JSON 失败："+err.Error())}
	}
	if err := platform.AtomicWriteFile(path, data, 0o600); err != nil {
		return StringResult{Result: model.FailResult("write", "写入文件失败："+err.Error())}
	}
	warnings := []string{}
	if includeSecrets {
		warnings = append(warnings, "导出文件包含 vkey 与密码，请谨慎保管")
	}
	return StringResult{Value: path, Result: model.OKResult(warnings...)}
}

func (a *App) startProfile(id string) model.OperationResult {
	if a.store == nil || a.manager == nil {
		return model.FailResult("not_ready", "应用尚未初始化")
	}
	p, ok := a.store.GetProfile(id)
	if !ok {
		return model.FailResult("not_found", "档案不存在")
	}
	res := configgen.Validate(p)
	if !res.OK {
		res.ProfileID = id
		return res
	}
	binary := a.currentBinary()
	if binary == nil {
		r := model.FailResult("no_binary", "尚未配置 npc 二进制")
		r.ProfileID = id
		return r
	}
	content, err := configgen.Generate(p)
	if err != nil {
		r := model.FailResult("config", "生成配置失败："+err.Error())
		r.ProfileID = id
		return r
	}
	configPath := a.store.ConfigPath(id)
	if err := configgen.WriteConfig(configPath, content); err != nil {
		r := model.FailResult("config_write", "写入配置失败："+err.Error())
		r.ProfileID = id
		return r
	}
	a.manager.SetBinary(binary.Path, binary.Version)
	res = a.manager.Start(a.baseContext(), p, configPath, configgen.ConfigHash(content))
	res.ProfileID = id
	return res
}

// StartProfile starts one profile process.
func (a *App) StartProfile(id string) model.OperationResult {
	return a.startProfile(id)
}

// StopProfile stops one profile process.
func (a *App) StopProfile(id string) model.OperationResult {
	if a.store == nil || a.manager == nil {
		return model.FailResult("not_ready", "应用尚未初始化")
	}
	p, ok := a.store.GetProfile(id)
	if !ok {
		return model.FailResult("not_found", "档案不存在")
	}
	res := a.manager.Stop(p)
	res.ProfileID = id
	return res
}

// RestartProfile stops then starts one profile.
func (a *App) RestartProfile(id string) model.OperationResult {
	if res := a.StopProfile(id); !res.OK {
		return res
	}
	return a.StartProfile(id)
}

// StartAll starts every stopped profile.
func (a *App) StartAll() []model.OperationResult {
	if a.store == nil || a.manager == nil {
		return []model.OperationResult{model.FailResult("not_ready", "应用尚未初始化")}
	}
	out := []model.OperationResult{}
	for _, p := range a.store.Profiles() {
		info := a.manager.Get(p.ID)
		switch info.State {
		case model.StateStarting, model.StateRunning, model.StateOnline, model.StateReconnecting, model.StateStopping:
			continue
		}
		out = append(out, a.startProfile(p.ID))
	}
	return out
}

// StopAll stops every profile.
func (a *App) StopAll() []model.OperationResult {
	if a.store == nil || a.manager == nil {
		return []model.OperationResult{model.FailResult("not_ready", "应用尚未初始化")}
	}
	out := []model.OperationResult{}
	for _, p := range a.store.Profiles() {
		out = append(out, a.StopProfile(p.ID))
	}
	return out
}

// GetLogs returns buffered logs for one profile.
func (a *App) GetLogs(id string, afterSeq uint64, limit int) model.LogPage {
	if a.manager == nil {
		return model.LogPage{ProfileID: id, Entries: []model.LogEntry{}}
	}
	return a.manager.Logs(id, afterSeq, limit)
}

// GetAllRecentLogs returns recent logs across profiles.
func (a *App) GetAllRecentLogs(limit int) []model.LogEntry {
	if a.manager == nil || a.store == nil {
		return []model.LogEntry{}
	}
	if limit <= 0 || limit > 5000 {
		limit = 300
	}
	out := []model.LogEntry{}
	for _, p := range a.store.Profiles() {
		page := a.manager.Logs(p.ID, 0, limit)
		out = append(out, page.Entries...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Time == out[j].Time {
			return out[i].Seq < out[j].Seq
		}
		return out[i].Time < out[j].Time
	})
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// ClearLogs clears the in-memory buffer for one profile.
func (a *App) ClearLogs(id string) model.OperationResult {
	if a.manager == nil {
		return model.FailResult("not_ready", "应用尚未初始化")
	}
	a.manager.ClearLogs(id)
	return model.OKResult()
}

// ExportLogs exports one profile's logs to a text file.
func (a *App) ExportLogs(id string) StringResult {
	if a.manager == nil {
		return StringResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	page := a.manager.Logs(id, 0, 100000)
	if len(page.Entries) == 0 {
		return StringResult{Result: model.FailResult("empty", "当前没有可导出的日志")}
	}
	name := "npc-log-" + id + "-" + time.Now().Format("20060102-150405") + ".log"
	path, err := wailsRuntime.SaveFileDialog(a.baseContext(), wailsRuntime.SaveDialogOptions{
		Title:            "导出日志",
		DefaultDirectory: a.dataDir,
		DefaultFilename:  name,
		Filters:          []wailsRuntime.FileFilter{{DisplayName: "日志文件", Pattern: "*.log;*.txt"}},
	})
	if err != nil || path == "" {
		return StringResult{Result: model.FailResult("cancel", "已取消导出")}
	}
	var b strings.Builder
	for _, e := range page.Entries {
		b.WriteString(time.UnixMilli(e.Time).Format("2006-01-02 15:04:05.000"))
		b.WriteString(" ")
		b.WriteString(strings.ToUpper(e.Level))
		b.WriteString(" ")
		b.WriteString(e.Text)
		b.WriteString("\n")
	}
	if err := platform.AtomicWriteFile(path, []byte(b.String()), 0o600); err != nil {
		return StringResult{Result: model.FailResult("write", "写入日志失败："+err.Error())}
	}
	return StringResult{Value: path, Result: model.OKResult()}
}

// PreviewConfig returns the generated npc.conf, optionally unmasked.
func (a *App) PreviewConfig(id string, reveal bool) model.ConfigPreview {
	if a.store == nil {
		return model.ConfigPreview{}
	}
	p, ok := a.store.GetProfile(id)
	if !ok {
		return model.ConfigPreview{}
	}
	content, err := configgen.Generate(p)
	if err != nil {
		return model.ConfigPreview{Content: "生成配置失败：" + err.Error()}
	}
	if !reveal {
		content = configgen.MaskSecrets(content)
	}
	npcPath := ""
	if b := a.currentBinary(); b != nil {
		npcPath = b.Path
	}
	command := configgen.FormatCommand(npcPath, configgen.BuildArgs(p, a.store.ConfigPath(id), a.currentSettings().DefaultLogLevel))
	return model.ConfigPreview{Path: a.store.ConfigPath(id), Content: content, Command: command}
}

// ExportConfig writes the real generated config to a user-selected path.
func (a *App) ExportConfig(id string) StringResult {
	if a.store == nil {
		return StringResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	p, ok := a.store.GetProfile(id)
	if !ok {
		return StringResult{Result: model.FailResult("not_found", "档案不存在")}
	}
	content, err := configgen.Generate(p)
	if err != nil {
		return StringResult{Result: model.FailResult("config", "生成配置失败："+err.Error())}
	}
	path, err := wailsRuntime.SaveFileDialog(a.baseContext(), wailsRuntime.SaveDialogOptions{
		Title:            "导出 npc 配置",
		DefaultDirectory: a.dataDir,
		DefaultFilename:  safeFileName(p.DisplayName()) + ".npc.conf",
		Filters:          []wailsRuntime.FileFilter{{DisplayName: "配置文件", Pattern: "*.conf"}},
	})
	if err != nil || path == "" {
		return StringResult{Result: model.FailResult("cancel", "已取消导出")}
	}
	if err := platform.AtomicWriteFile(path, []byte(content), 0o600); err != nil {
		return StringResult{Result: model.FailResult("write", "写入配置失败："+err.Error())}
	}
	return StringResult{Value: path, Result: model.OKResult("配置文件包含 vkey，请谨慎保管")}
}

func safeFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "npc-profile"
	}
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			b.WriteRune('-')
		default:
			if r < 32 {
				b.WriteRune('-')
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// GetSettings returns application settings.
func (a *App) GetSettings() model.Settings {
	return a.currentSettings()
}

// SaveSettings validates and persists application settings.
func (a *App) SaveSettings(s model.Settings) SettingsSaveResult {
	if a.store == nil {
		return SettingsSaveResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	previous := a.currentSettings()
	if s.LaunchAtLogin != previous.LaunchAtLogin {
		if err := autostart.SetEnabled(s.LaunchAtLogin); err != nil {
			return SettingsSaveResult{Result: model.FailResult("autostart_failed", "设置开机自启失败："+err.Error())}
		}
	}
	if s.MaxLogLines < 500 {
		s.MaxLogLines = 500
	}
	if s.MaxLogLines > 50000 {
		s.MaxLogLines = 50000
	}
	if s.Theme != "dark" && s.Theme != "light" {
		s.Theme = "dark"
	}
	if s.DefaultLogLevel == "" {
		s.DefaultLogLevel = "info"
	}
	if s.Locale == "" {
		s.Locale = "zh-CN"
	}
	saved, err := a.store.SaveSettings(s)
	if err != nil {
		return SettingsSaveResult{Result: model.FailResult("save_failed", "保存设置失败："+err.Error())}
	}
	if a.updater != nil {
		a.updater.SetMirror(saved.DownloadMirror)
	}
	if a.manager != nil {
		a.manager.SetLogToDisk(saved.LogToDisk)
	}
	if saved.TrayEnabled != previous.TrayEnabled {
		a.emit("app:notice", map[string]string{"level": "info", "message": "系统托盘设置将在应用下次启动时生效"})
	}
	a.updateTray()
	return SettingsSaveResult{Settings: saved, Result: model.OKResult()}
}

// DetectNpcBinaries probes known locations for npc.
func (a *App) DetectNpcBinaries() []model.BinaryInfo {
	if a.npcBin == nil {
		return []model.BinaryInfo{}
	}
	ctx, cancel := context.WithTimeout(a.baseContext(), 20*time.Second)
	defer cancel()
	res := a.detectBinary(ctx)
	return res.Candidates
}

// ValidateNpcBinary checks a selected executable.
func (a *App) ValidateNpcBinary(path string) BinaryResult {
	ctx, cancel := context.WithTimeout(a.baseContext(), 10*time.Second)
	defer cancel()
	info, err := npcbin.Probe(ctx, path, "自定义")
	if err != nil {
		return BinaryResult{Result: model.FailResult("invalid_binary", err.Error())}
	}
	return BinaryResult{Binary: &info, Candidates: a.binaryCandidates(), Result: model.OKResult()}
}

// SetNpcBinary validates and activates a local npc executable.
func (a *App) SetNpcBinary(path string) BinaryResult {
	ctx, cancel := context.WithTimeout(a.baseContext(), 10*time.Second)
	defer cancel()
	info, err := npcbin.Probe(ctx, path, "自定义")
	if err != nil {
		return BinaryResult{Result: model.FailResult("invalid_binary", err.Error())}
	}
	a.activateBinary(info)
	return BinaryResult{Binary: &info, Candidates: a.binaryCandidates(), Result: model.OKResult()}
}

func (a *App) activateBinary(info model.BinaryInfo) {
	a.mu.Lock()
	a.binary = &info
	a.mu.Unlock()
	if a.manager != nil {
		a.manager.SetBinary(info.Path, info.Version)
	}
	if a.store != nil {
		_ = a.store.SetBinary(&info)
		s := a.currentSettings()
		s.NpcBinaryPath = info.Path
		_, _ = a.store.SaveSettings(s)
	}
	a.emit("binary:info", a.binaryState())
	a.startAutoProfiles()
}

// PickNpcBinary opens a file dialog and validates the selected npc.
func (a *App) PickNpcBinary() BinaryResult {
	path, err := wailsRuntime.OpenFileDialog(a.baseContext(), wailsRuntime.OpenDialogOptions{
		Title:            "选择 npc 可执行文件",
		DefaultDirectory: a.dataDir,
	})
	if err != nil || path == "" {
		return BinaryResult{Result: model.FailResult("cancel", "已取消选择")}
	}
	return a.SetNpcBinary(path)
}

// DownloadNpc downloads the latest upstream npc release.
func (a *App) DownloadNpc() BinaryResult {
	if a.npcBin == nil {
		return BinaryResult{Result: model.FailResult("not_ready", "应用尚未初始化")}
	}
	ctx, cancel := context.WithCancel(a.baseContext())
	a.mu.Lock()
	if a.npcCancel != nil {
		a.npcCancel()
	}
	a.npcCancel = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.npcCancel = nil
		a.mu.Unlock()
		cancel()
	}()
	settings := a.currentSettings()
	a.npcBin.Mirror = settings.DownloadMirror
	a.emit("binary:progress", model.UpdateProgress{Stage: "准备下载", Percent: 0})
	info, err := a.npcBin.DownloadLatest(ctx, func(stage string, received, total int64) {
		percent := 0.0
		if total > 0 {
			percent = float64(received) / float64(total) * 100
		}
		a.emit("binary:progress", model.UpdateProgress{Stage: stage, Received: received, Total: total, Percent: percent})
	})
	if err != nil {
		return BinaryResult{Result: model.FailResult("download_failed", err.Error())}
	}
	a.activateBinary(info)
	return BinaryResult{Binary: &info, Candidates: a.binaryCandidates(), Result: model.OKResult()}
}

// CancelNpcDownload cancels an in-progress npc download.
func (a *App) CancelNpcDownload() model.OperationResult {
	a.mu.Lock()
	cancel := a.npcCancel
	a.npcCancel = nil
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return model.OKResult()
}

// CheckAppUpdate queries GitHub for a newer client release.
func (a *App) CheckAppUpdate(force bool) UpdateStateResult {
	if a.updater == nil {
		return UpdateStateResult{State: model.UpdateStateInfo{State: model.UpdateDisabled, CurrentVersion: buildinfo.Version}, Result: model.FailResult("disabled", "更新未启用")}
	}
	ctx, cancel := context.WithTimeout(a.baseContext(), 25*time.Second)
	defer cancel()
	state, err := a.updater.Check(ctx, force)
	if state.CheckedAt > 0 && a.store != nil {
		s := a.currentSettings()
		s.LastUpdateCheck = state.CheckedAt
		_, _ = a.store.SaveSettings(s)
	}
	res := model.OKResult()
	if err != nil {
		res = model.FailResult("update_check", err.Error())
	}
	return UpdateStateResult{State: state, Result: res}
}

// DownloadAppUpdate downloads and stages the available client update.
func (a *App) DownloadAppUpdate() UpdateStateResult {
	if a.updater == nil {
		return UpdateStateResult{State: model.UpdateStateInfo{State: model.UpdateDisabled}, Result: model.FailResult("disabled", "更新未启用")}
	}
	ctx, cancel := context.WithTimeout(a.baseContext(), 30*time.Minute)
	defer cancel()
	state, err := a.updater.Download(ctx)
	res := model.OKResult()
	if err != nil {
		res = model.FailResult("update_download", err.Error())
	}
	return UpdateStateResult{State: state, Result: res}
}

// CancelAppUpdate cancels the active client update download.
func (a *App) CancelAppUpdate() model.OperationResult {
	if a.updater != nil {
		a.updater.CancelDownload()
	}
	return model.OKResult()
}

// DiscardAppUpdate removes a staged client update.
func (a *App) DiscardAppUpdate() model.OperationResult {
	if a.updater != nil {
		if err := a.updater.Discard(); err != nil {
			return model.FailResult("discard", err.Error())
		}
	}
	return model.OKResult()
}

// GetUpdateState returns the current updater state.
func (a *App) GetUpdateState() model.UpdateStateInfo {
	if a.updater == nil {
		return model.UpdateStateInfo{State: model.UpdateDisabled, CurrentVersion: buildinfo.Version}
	}
	return a.updater.State()
}

// ApplyAppUpdate stops profiles and restarts into the staged client version.
func (a *App) ApplyAppUpdate(stopAll bool) UpdateActionResult {
	if a.updater == nil {
		return UpdateActionResult{Message: "更新未启用"}
	}
	if stopAll {
		for _, res := range a.StopAll() {
			if !res.OK {
				return UpdateActionResult{Message: "停止连接失败：" + res.Message}
			}
		}
	} else if a.manager != nil && a.manager.RunningCount() > 0 {
		return UpdateActionResult{Message: "请先停止所有连接再更新"}
	}
	exe, err := os.Executable()
	if err != nil {
		return UpdateActionResult{Message: "无法定位当前应用：" + err.Error()}
	}
	if err := a.updater.Apply(a.baseContext(), exe); err != nil {
		return UpdateActionResult{Message: err.Error()}
	}
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	ctx := a.baseContext()
	go func() {
		time.Sleep(700 * time.Millisecond)
		wailsRuntime.Quit(ctx)
	}()
	return UpdateActionResult{OK: true, Message: "正在重启并更新"}
}

// OpenUpdateLog opens the updater log if it exists.
func (a *App) OpenUpdateLog() model.OperationResult {
	path := filepath.Join(a.dataDir, "updates", "update.log")
	if _, err := os.Stat(path); err != nil {
		return model.FailResult("not_found", "还没有更新日志")
	}
	if err := platform.OpenPath(path); err != nil {
		return model.FailResult("open_failed", err.Error())
	}
	return model.OKResult()
}

// OpenDataDir opens the application data directory.
func (a *App) OpenDataDir() model.OperationResult {
	if err := platform.OpenPath(a.dataDir); err != nil {
		return model.FailResult("open_failed", err.Error())
	}
	return model.OKResult()
}

// OpenExternal opens a validated HTTP(S) URL in the system browser.
func (a *App) OpenExternal(raw string) model.OperationResult {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return model.FailResult("invalid_url", "只允许打开 HTTP(S) 链接")
	}
	wailsRuntime.BrowserOpenURL(a.baseContext(), raw)
	return model.OKResult()
}

// OpenReleasePage opens the client GitHub release page.
func (a *App) OpenReleasePage() model.OperationResult {
	if buildinfo.Repo == "" {
		return model.FailResult("no_repo", "开发构建没有关联 GitHub 仓库")
	}
	wailsRuntime.BrowserOpenURL(a.baseContext(), "https://github.com/"+buildinfo.Repo+"/releases")
	return model.OKResult()
}

// OpenNpcReleasePage opens the upstream npc release page.
func (a *App) OpenNpcReleasePage() model.OperationResult {
	wailsRuntime.BrowserOpenURL(a.baseContext(), "https://github.com/djylb/nps/releases")
	return model.OKResult()
}

// PickDirectory opens a directory picker.
func (a *App) PickDirectory(title string) StringResult {
	path, err := wailsRuntime.OpenDirectoryDialog(a.baseContext(), wailsRuntime.OpenDialogOptions{
		Title:            title,
		DefaultDirectory: a.dataDir,
	})
	if err != nil || path == "" {
		return StringResult{Result: model.FailResult("cancel", "已取消选择")}
	}
	return StringResult{Value: path, Result: model.OKResult()}
}

// GetDiagnostics returns a redacted diagnostic summary.
func (a *App) GetDiagnostics() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "NPC GUI %s (%s)\n", buildinfo.Version, buildinfo.Commit)
	fmt.Fprintf(&sb, "build: %s\nrepo: %s\n", buildinfo.BuildDate, buildinfo.Repo)
	fmt.Fprintf(&sb, "data: %s\n", a.dataDir)
	if a.store != nil {
		profiles := a.store.Profiles()
		fmt.Fprintf(&sb, "profiles: %d\n", len(profiles))
		for _, p := range profiles {
			fmt.Fprintf(&sb, "- %s | %s | %s | autoStart=%v\n", p.DisplayName(), p.Server, p.ConnType, p.AutoStart)
		}
	}
	if bin := a.currentBinary(); bin != nil {
		fmt.Fprintf(&sb, "npc: %s %s (%s)\n", bin.Path, bin.Version, bin.SHA256)
	} else {
		sb.WriteString("npc: not configured\n")
	}
	if a.updater != nil {
		state := a.updater.State()
		fmt.Fprintf(&sb, "update: %s current=%s latest=%s\n", state.State, state.CurrentVersion, latestVersion(state))
	}
	return sb.String()
}

func latestVersion(s model.UpdateStateInfo) string {
	if s.Latest != nil {
		return s.Latest.Version
	}
	return ""
}

// ConfirmQuit stops all profiles and exits.
func (a *App) ConfirmQuit() {
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	if a.store != nil && a.manager != nil {
		for _, p := range a.store.Profiles() {
			_ = a.manager.Stop(p)
		}
	}
	wailsRuntime.Quit(a.baseContext())
}

func (a *App) beforeClose(ctx context.Context) bool {
	if a.isQuitting() {
		return false
	}
	settings := a.currentSettings()
	if settings.TrayEnabled && settings.MinimizeToTray && a.trayStarted() {
		wailsRuntime.WindowHide(ctx)
		return true
	}
	count := 0
	if a.manager != nil {
		count = a.manager.RunningCount()
	}
	if count > 0 && settings.ConfirmOnQuit {
		a.emit("app:close-request", map[string]int{"running": count})
		return true
	}
	return false
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	a.quitting = true
	a.mu.Unlock()
	a.stopTray()
	if a.manager != nil {
		a.manager.Close()
		// Best effort: if a process somehow survived Close, stop it before exit.
		cleaned, warnings := a.manager.CleanupStale()
		for _, message := range cleaned {
			a.logger.Info("npc process cleaned during shutdown", "message", message)
		}
		for _, message := range warnings {
			a.logger.Warn("npc process left during shutdown", "message", message)
		}
	}
	if a.logFile != nil {
		_ = a.logFile.Close()
	}
}

func (a *App) secondInstance(_ options.SecondInstanceData) {
	a.mu.RLock()
	ctx := a.ctx
	a.mu.RUnlock()
	if ctx != nil {
		wailsRuntime.WindowUnminimise(ctx)
		wailsRuntime.WindowShow(ctx)
	}
}
