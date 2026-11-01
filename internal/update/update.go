package update

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"npc-gui/internal/buildinfo"
	"npc-gui/internal/ghrelease"
	"npc-gui/internal/model"
	"npc-gui/internal/platform"
)

// Emitter receives Wails events.
type Emitter func(event string, payload any)

// Manager checks, downloads and stages application updates.
type Manager struct {
	mu             sync.Mutex
	dataDir        string
	repo           string
	currentVersion string
	mirror         string
	apiBase        string
	emit           Emitter
	logger         *slog.Logger
	state          model.UpdateStateInfo
	cancel         context.CancelFunc
}

type cachedRelease struct {
	CheckedAt int64            `json:"checkedAt"`
	Latest    model.UpdateInfo `json:"latest"`
}

type stagedState struct {
	Version    string `json:"version"`
	AssetName  string `json:"assetName"`
	StagedBin  string `json:"stagedBin"`
	StagedApp  string `json:"stagedApp,omitempty"`
	Kind       string `json:"kind"`
	Downloaded int64  `json:"downloadedAt"`
}

// New creates an update manager for the current build.
func New(dataDir string, emit Emitter, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	repo := buildinfo.UpdateRepo()
	version := buildinfo.Version
	if !buildinfo.IsRelease() {
		if override := strings.TrimSpace(os.Getenv("NPC_GUI_UPDATE_REPO")); override != "" && version == "0.1.0-dev" {
			repo = override
		} else {
			repo = ""
		}
	}
	m := &Manager{
		dataDir:        dataDir,
		repo:           repo,
		currentVersion: version,
		emit:           emit,
		logger:         logger,
	}
	m.state = model.UpdateStateInfo{State: model.UpdateIdle, CurrentVersion: version}
	if !buildinfo.IsRelease() {
		m.apiBase = strings.TrimSpace(os.Getenv("NPC_GUI_UPDATE_API"))
	}
	if repo == "" {
		m.state.State = model.UpdateDisabled
	}
	m.loadStaged()
	m.loadCache()
	return m
}

// Enabled reports whether this build can check for updates.
func (m *Manager) Enabled() bool {
	return strings.TrimSpace(m.repo) != ""
}

// SetMirror sets an optional download mirror prefix.
func (m *Manager) SetMirror(prefix string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mirror = strings.TrimSpace(prefix)
}

// State returns a copy of the current update state.
func (m *Manager) State() model.UpdateStateInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked()
}

func (m *Manager) snapshotLocked() model.UpdateStateInfo {
	out := m.state
	if out.Latest != nil {
		latest := *out.Latest
		out.Latest = &latest
	}
	if out.Progress != nil {
		progress := *out.Progress
		out.Progress = &progress
	}
	return out
}

func (m *Manager) setState(state string, errMsg string) {
	m.state.State = state
	m.state.Error = errMsg
	if state != model.UpdateDownloading && state != model.UpdateApplying {
		// Keep progress only during transfer/applying.
		if state != model.UpdateReady {
			m.state.Progress = nil
		}
	}
	m.emitState()
}

func (m *Manager) emitState() {
	if m.emit == nil {
		return
	}
	m.emit("update:state", m.snapshotLocked())
}

func (m *Manager) emitProgress(stage string, received, total int64) {
	percent := 0.0
	if total > 0 {
		percent = float64(received) / float64(total) * 100
		if percent > 100 {
			percent = 100
		}
	}
	p := model.UpdateProgress{Stage: stage, Received: received, Total: total, Percent: percent}
	m.state.Progress = &p
	if m.emit != nil {
		m.emit("update:progress", p)
	}
}

// Check queries GitHub for the latest release.
func (m *Manager) Check(ctx context.Context, force bool) (model.UpdateStateInfo, error) {
	m.mu.Lock()
	if !m.Enabled() {
		m.state.State = model.UpdateDisabled
		out := m.snapshotLocked()
		m.mu.Unlock()
		return out, nil
	}
	if m.state.State == model.UpdateDownloading || m.state.State == model.UpdateApplying {
		out := m.snapshotLocked()
		m.mu.Unlock()
		return out, nil
	}
	if !force && m.state.CheckedAt > 0 && time.Since(time.UnixMilli(m.state.CheckedAt)) < 24*time.Hour {
		out := m.snapshotLocked()
		m.mu.Unlock()
		return out, nil
	}
	m.state.State = model.UpdateChecking
	m.state.Error = ""
	m.emitState()
	repo := m.repo
	mirror := m.mirror
	apiBase := m.apiBase
	m.mu.Unlock()

	cl := ghrelease.NewClient(repo)
	cl.APIBase = apiBase
	cl.MirrorPrefix = mirror
	rel, err := cl.Latest(ctx)
	if err != nil {
		m.mu.Lock()
		m.setState(model.UpdateError, err.Error())
		out := m.snapshotLocked()
		m.mu.Unlock()
		return out, err
	}
	latest := model.UpdateInfo{
		Version:     ghrelease.NormalizeVersion(rel.TagName),
		Tag:         rel.TagName,
		Name:        rel.Name,
		Notes:       rel.Body,
		ReleaseURL:  rel.HTMLURL,
		PublishedAt: rel.PublishedAt.UnixMilli(),
	}
	asset := selectAsset(rel, latest.Version)
	if asset != nil {
		latest.AssetName = asset.Name
		latest.AssetSize = asset.Size
		latest.DownloadURL = asset.BrowserDownloadURL
		latest.SHA256 = expectedSHA(asset, nil)
	}
	if sums, err := cl.Checksums(ctx, rel); err == nil && asset != nil {
		if sum := sums[asset.Name]; sum != "" {
			latest.SHA256 = sum
		}
	}
	now := time.Now().UnixMilli()
	m.mu.Lock()
	m.state.CheckedAt = now
	m.state.Latest = &latest
	if m.state.State == model.UpdateReady {
		// A downloaded update is still waiting for restart; keep it ready.
	} else if ghrelease.CompareVersions(latest.Version, m.currentVersion) > 0 {
		m.state.State = model.UpdateAvailable
	} else {
		m.state.State = model.UpdateIdle
	}
	m.state.Error = ""
	m.saveCacheLocked()
	m.emitState()
	out := m.snapshotLocked()
	m.mu.Unlock()
	return out, nil
}

func selectAsset(rel *ghrelease.Release, version string) *ghrelease.Asset {
	name := appAssetName(version, runtime.GOOS, runtime.GOARCH)
	if a := rel.AssetByName(name); a != nil {
		return a
	}
	suffix := assetSuffix(runtime.GOOS, runtime.GOARCH)
	for i := range rel.Assets {
		if strings.HasSuffix(rel.Assets[i].Name, suffix) {
			return &rel.Assets[i]
		}
	}
	return nil
}

func appAssetName(version, goos, goarch string) string {
	switch goos {
	case "darwin":
		return fmt.Sprintf("npc-gui_%s_%s_%s.app.zip", version, goos, goarch)
	case "windows":
		return fmt.Sprintf("npc-gui_%s_%s_%s.exe", version, goos, goarch)
	default:
		return fmt.Sprintf("npc-gui_%s_%s_%s.tar.gz", version, goos, goarch)
	}
}

func assetSuffix(goos, goarch string) string {
	switch goos {
	case "darwin":
		return "_" + goos + "_" + goarch + ".app.zip"
	case "windows":
		return "_" + goos + "_" + goarch + ".exe"
	default:
		return "_" + goos + "_" + goarch + ".tar.gz"
	}
}

func expectedSHA(asset *ghrelease.Asset, sums map[string]string) string {
	if sums != nil {
		if sum := sums[asset.Name]; sum != "" {
			return sum
		}
	}
	digest := strings.TrimSpace(asset.Digest)
	digest = strings.TrimPrefix(strings.ToLower(digest), "sha256:")
	return digest
}

// Download downloads and stages the available update.
func (m *Manager) Download(ctx context.Context) (model.UpdateStateInfo, error) {
	m.mu.Lock()
	if m.state.State != model.UpdateAvailable || m.state.Latest == nil {
		out := m.snapshotLocked()
		m.mu.Unlock()
		return out, fmt.Errorf("当前没有可下载的更新")
	}
	info := *m.state.Latest
	repo := m.repo
	mirror := m.mirror
	apiBase := m.apiBase
	m.state.State = model.UpdateDownloading
	m.state.Error = ""
	m.state.Progress = &model.UpdateProgress{Stage: "准备下载"}
	m.emitState()
	runCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.cancel = nil
		m.mu.Unlock()
	}()

	cl := ghrelease.NewClient(repo)
	cl.APIBase = apiBase
	cl.MirrorPrefix = mirror
	stageDir := filepath.Join(m.dataDir, "updates", info.Version)
	if err := platform.EnsureDir(stageDir); err != nil {
		return m.failDownload(err)
	}
	if info.AssetName == "" {
		return m.failDownload(fmt.Errorf("该版本没有当前平台的更新资产"))
	}
	// The asset is reconstructed from release info; download by URL.
	asset := ghrelease.Asset{
		Name:               info.AssetName,
		Size:               info.AssetSize,
		BrowserDownloadURL: info.DownloadURL,
		Digest:             info.SHA256,
	}
	if info.DownloadURL == "" {
		return m.failDownload(fmt.Errorf("更新资产缺少下载地址"))
	}
	if info.SHA256 == "" && asset.Digest == "" {
		return m.failDownload(fmt.Errorf("更新资产缺少 SHA-256，已拒绝自动更新"))
	}
	m.mu.Lock()
	m.emitProgress("下载更新", 0, info.AssetSize)
	m.mu.Unlock()
	archivePath := filepath.Join(stageDir, info.AssetName)
	sum, err := cl.Download(runCtx, asset, archivePath, info.SHA256, func(received, total int64) {
		m.mu.Lock()
		m.emitProgress("下载更新", received, total)
		m.mu.Unlock()
	})
	if err != nil {
		return m.failDownload(err)
	}
	m.mu.Lock()
	m.emitProgress("校验更新", 0, 0)
	m.mu.Unlock()
	staged := stagedState{Version: info.Version, AssetName: info.AssetName, Kind: "binary", Downloaded: time.Now().UnixMilli()}
	if strings.HasSuffix(info.AssetName, ".app.zip") {
		m.mu.Lock()
		m.emitProgress("解压应用", 0, 0)
		m.mu.Unlock()
		appDir, binPath, err := extractAppZip(archivePath, stageDir)
		if err != nil {
			return m.failDownload(err)
		}
		staged.Kind = "app"
		staged.StagedApp = appDir
		staged.StagedBin = binPath
	} else if strings.HasSuffix(info.AssetName, ".tar.gz") {
		m.mu.Lock()
		m.emitProgress("解压更新", 0, 0)
		m.mu.Unlock()
		binPath := filepath.Join(stageDir, "npc-gui")
		if err := extractTarBinary(archivePath, binPath); err != nil {
			return m.failDownload(err)
		}
		staged.StagedBin = binPath
	} else {
		staged.StagedBin = archivePath
	}
	helper := filepath.Join(stageDir, helperBinaryName())
	if err := copyFile(staged.StagedBin, helper, 0o755); err != nil {
		return m.failDownload(err)
	}
	if err := writeJSON(filepath.Join(m.dataDir, "updates", "staged.json"), staged); err != nil {
		return m.failDownload(err)
	}
	m.mu.Lock()
	m.state.State = model.UpdateReady
	m.state.StagedVersion = info.Version
	m.state.Progress = nil
	m.state.Error = ""
	m.state.Latest = &info
	m.emitState()
	out := m.snapshotLocked()
	m.mu.Unlock()
	m.logger.Info("update staged", "version", info.Version, "sha256", sum)
	return out, nil
}

func (m *Manager) failDownload(err error) (model.UpdateStateInfo, error) {
	m.mu.Lock()
	m.setState(model.UpdateError, err.Error())
	out := m.snapshotLocked()
	m.mu.Unlock()
	return out, err
}

// CancelDownload cancels the active download.
func (m *Manager) CancelDownload() {
	m.mu.Lock()
	cancel := m.cancel
	m.cancel = nil
	m.state.State = model.UpdateIdle
	m.state.Progress = nil
	m.emitState()
	m.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Apply spawns the helper that replaces this application and restarts it.
func (m *Manager) Apply(ctx context.Context, currentExe string) error {
	m.mu.Lock()
	if m.state.State != model.UpdateReady {
		m.mu.Unlock()
		return fmt.Errorf("没有已下载的更新")
	}
	info := m.state.Latest
	version := m.state.StagedVersion
	m.state.State = model.UpdateApplying
	m.state.Error = ""
	m.emitState()
	m.mu.Unlock()

	staged, err := readStaged(m.dataDir)
	if err != nil {
		return m.failApply(err)
	}
	if info == nil || version == "" {
		return m.failApply(fmt.Errorf("更新元数据缺失"))
	}
	helper := filepath.Join(m.dataDir, "updates", version, helperBinaryName())
	if _, err := os.Stat(helper); err != nil {
		return m.failApply(fmt.Errorf("更新 helper 不存在：%w", err))
	}
	target, appPath, kind, err := resolveApplyTargets(currentExe)
	if err != nil {
		return m.failApply(err)
	}
	if strings.Contains(strings.ToLower(target), "apptranslocation") {
		return m.failApply(fmt.Errorf("应用正在 macOS 临时目录运行，请先移动到“应用程序”后再更新"))
	}
	if !writableDir(filepath.Dir(target)) {
		return m.failApply(fmt.Errorf("应用目录不可写，请手动下载新版本：%s", info.ReleaseURL))
	}
	logPath := filepath.Join(m.dataDir, "updates", "update.log")
	args := []string{
		"--apply-update",
		"--target", target,
		"--staged", staged.StagedBin,
		"--staged-app", staged.StagedApp,
		"--kind", kind,
		"--old-pid", fmt.Sprintf("%d", os.Getpid()),
		"--app-path", appPath,
		"--log", logPath,
	}
	if runtime.GOOS == "darwin" && kind == "app" {
		args = append(args, "--relaunch", appPath)
	} else {
		args = append(args, "--relaunch", target)
	}
	cmd := commandForHelper(helper, args...)
	if err := cmd.Start(); err != nil {
		return m.failApply(fmt.Errorf("启动更新 helper 失败：%w", err))
	}
	return nil
}

func (m *Manager) failApply(err error) error {
	m.mu.Lock()
	m.setState(model.UpdateError, err.Error())
	m.mu.Unlock()
	return err
}

// Discard removes a staged update.
func (m *Manager) Discard() error {
	m.mu.Lock()
	version := m.state.StagedVersion
	m.state.State = model.UpdateIdle
	m.state.StagedVersion = ""
	m.state.Progress = nil
	m.state.Error = ""
	m.emitState()
	m.mu.Unlock()
	_ = os.Remove(filepath.Join(m.dataDir, "updates", "staged.json"))
	if version != "" {
		_ = os.RemoveAll(filepath.Join(m.dataDir, "updates", version))
	}
	return nil
}

func (m *Manager) loadStaged() {
	if !m.Enabled() {
		return
	}
	staged, err := readStaged(m.dataDir)
	if err != nil {
		return
	}
	if staged.StagedBin == "" {
		return
	}
	if _, err := os.Stat(staged.StagedBin); err != nil {
		return
	}
	m.state.State = model.UpdateReady
	m.state.StagedVersion = staged.Version
	var info *model.UpdateInfo
	if cached, err := readCache(m.dataDir); err == nil {
		info = &cached.Latest
	}
	if info != nil {
		m.state.Latest = info
	}
}

func (m *Manager) loadCache() {
	cached, err := readCache(m.dataDir)
	if err != nil {
		return
	}
	if !m.Enabled() {
		return
	}
	m.state.CheckedAt = cached.CheckedAt
	m.state.Latest = &cached.Latest
	if m.state.State == model.UpdateReady || m.state.State == model.UpdateDisabled {
		return
	}
	if strings.TrimSpace(cached.Latest.Version) != "" {
		if ghrelease.CompareVersions(cached.Latest.Version, m.currentVersion) > 0 {
			m.state.State = model.UpdateAvailable
		}
	}
}

func (m *Manager) saveCacheLocked() {
	if m.state.Latest == nil {
		return
	}
	_ = writeJSON(filepath.Join(m.dataDir, "updates", "latest.json"), cachedRelease{
		CheckedAt: m.state.CheckedAt,
		Latest:    *m.state.Latest,
	})
}

func readCache(dataDir string) (cachedRelease, error) {
	var c cachedRelease
	data, err := os.ReadFile(filepath.Join(dataDir, "updates", "latest.json"))
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(data, &c)
	return c, err
}

func readStaged(dataDir string) (stagedState, error) {
	var s stagedState
	data, err := os.ReadFile(filepath.Join(dataDir, "updates", "staged.json"))
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return platform.AtomicWriteFile(path, data, 0o600)
}

func resolveApplyTargets(currentExe string) (target, appPath, kind string, err error) {
	exe, err := os.Executable()
	if err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(currentExe) != "" {
		exe = currentExe
	}
	if runtime.GOOS == "darwin" {
		idx := strings.Index(exe, ".app/Contents/MacOS/")
		if idx < 0 {
			return "", "", "", fmt.Errorf("当前不是从 .app 包内运行，无法自动替换应用")
		}
		appPath = exe[:idx+len(".app")]
		return appPath, appPath, "app", nil
	}
	return exe, "", "binary", nil
}

func helperBinaryName() string {
	if runtime.GOOS == "windows" {
		return "helper.exe"
	}
	return "helper"
}

func writableDir(dir string) bool {
	f, err := os.CreateTemp(dir, ".npc-gui-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}
