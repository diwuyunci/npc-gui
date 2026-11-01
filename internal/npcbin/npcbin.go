package npcbin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"npc-gui/internal/ghrelease"
	"npc-gui/internal/model"
	"npc-gui/internal/platform"
)

// Manager discovers and installs the upstream npc binary.
type Manager struct {
	DataDir string
	Mirror  string
}

// NewManager creates an npc binary manager.
func NewManager(dataDir string) *Manager {
	return &Manager{DataDir: dataDir}
}

// AssetName maps the current platform to an upstream release asset.
func AssetName() (string, error) {
	return AssetNameFor(runtime.GOOS, runtime.GOARCH)
}

// AssetNameFor maps a platform to an upstream release asset.
func AssetNameFor(goos, goarch string) (string, error) {
	arch := goarch
	switch {
	case goos == "linux" && goarch == "arm":
		arch = "arm_v7"
	case goos == "linux" && goarch == "mips":
		arch = "mips"
	}
	name := fmt.Sprintf("%s_%s_client.tar.gz", goos, arch)
	return name, nil
}

// ManagedDir returns the install directory for a release tag.
func (m *Manager) ManagedDir(tag string) string {
	tag = strings.TrimPrefix(strings.TrimSpace(tag), "v")
	return filepath.Join(m.DataDir, "bin", tag, runtime.GOOS+"_"+runtime.GOARCH)
}

// ManagedBinary returns the installed binary path for a release tag.
func (m *Manager) ManagedBinary(tag string) string {
	name := "npc"
	if runtime.GOOS == "windows" {
		name = "npc.exe"
	}
	return filepath.Join(m.ManagedDir(tag), name)
}

// Probe executes npc -version and records metadata.
func Probe(ctx context.Context, path, source string) (model.BinaryInfo, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return model.BinaryInfo{}, fmt.Errorf("二进制路径为空")
	}
	info, err := os.Stat(path)
	if err != nil {
		return model.BinaryInfo{}, fmt.Errorf("无法访问 npc：%w", err)
	}
	if info.IsDir() {
		return model.BinaryInfo{}, fmt.Errorf("选择的是目录，不是 npc 可执行文件")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, path, "-version")
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return model.BinaryInfo{}, fmt.Errorf("执行 npc -version 失败：%w（输出：%s）", err, strings.TrimSpace(string(out)))
	}
	version, core := ParseVersionOutput(string(out))
	if version == "" {
		return model.BinaryInfo{}, fmt.Errorf("无法从 npc -version 输出识别版本：%s", strings.TrimSpace(string(out)))
	}
	sum, _ := platform.SHA256File(path)
	return model.BinaryInfo{
		Path:        path,
		Version:     version,
		CoreVersion: core,
		Source:      source,
		SHA256:      sum,
		Managed:     false,
	}, nil
}

// ParseVersionOutput extracts Version and Core version lines.
func ParseVersionOutput(out string) (version, core string) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "version:"):
			version = strings.TrimSpace(line[len("version:"):])
		case strings.HasPrefix(lower, "core version:"):
			core = strings.TrimSpace(line[len("core version:"):])
		}
	}
	return version, core
}

// CandidatePaths returns possible npc locations in priority order.
func (m *Manager) CandidatePaths(settingsPath string) []struct {
	Path   string
	Source string
} {
	var out []struct {
		Path   string
		Source string
	}
	add := func(path, source string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err == nil {
			path = abs
		}
		for _, existing := range out {
			if existing.Path == path {
				return
			}
		}
		out = append(out, struct {
			Path   string
			Source string
		}{Path: path, Source: source})
	}
	if settingsPath != "" {
		add(settingsPath, "自定义")
	}
	// Managed versions, newest first.
	binRoot := filepath.Join(m.DataDir, "bin")
	if entries, err := os.ReadDir(binRoot); err == nil {
		var tags []string
		for _, e := range entries {
			if e.IsDir() {
				tags = append(tags, e.Name())
			}
		}
		sort.Slice(tags, func(i, j int) bool {
			return ghrelease.CompareVersions(tags[i], tags[j]) > 0
		})
		for _, tag := range tags {
			add(m.ManagedBinary(tag), "GUI 下载")
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		name := "npc"
		if runtime.GOOS == "windows" {
			name = "npc.exe"
		}
		add(filepath.Join(dir, name), "应用目录")
		// macOS app bundle: also check Contents/MacOS.
		if runtime.GOOS == "darwin" && strings.Contains(dir, ".app/Contents/MacOS") {
			add(filepath.Join(dir, name), "应用目录")
		}
	}
	if path, err := exec.LookPath("npc"); err == nil {
		add(path, "PATH")
	}
	common := []string{
		"/usr/local/bin/npc",
		"/opt/homebrew/bin/npc",
		"/usr/bin/npc",
	}
	if home, err := os.UserHomeDir(); err == nil {
		common = append(common, filepath.Join(home, "npc", "npc"), filepath.Join(home, ".npc", "npc"))
	}
	common = append(common,
		currentExeAdjacent("npc.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "npc", "npc.exe"),
		filepath.Join(os.Getenv("USERPROFILE"), "npc", "npc.exe"),
	)
	for _, p := range common {
		add(p, "常见路径")
	}
	return out
}

func currentExeAdjacent(name string) string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Join(filepath.Dir(exe), name)
}

// Detect returns all valid npc candidates.
func (m *Manager) Detect(ctx context.Context, settingsPath string) []model.BinaryInfo {
	var out []model.BinaryInfo
	for _, candidate := range m.CandidatePaths(settingsPath) {
		info, err := Probe(ctx, candidate.Path, candidate.Source)
		if err != nil {
			continue
		}
		if strings.HasPrefix(candidate.Path, filepath.Join(m.DataDir, "bin")) {
			info.Managed = true
		}
		out = append(out, info)
	}
	return out
}

// DownloadLatest downloads the latest upstream npc release.
func (m *Manager) DownloadLatest(ctx context.Context, progress func(stage string, received, total int64)) (model.BinaryInfo, error) {
	client := ghrelease.NewClient("djylb/nps")
	client.MirrorPrefix = m.Mirror
	rel, err := client.Latest(ctx)
	if err != nil {
		return model.BinaryInfo{}, err
	}
	assetName, err := AssetName()
	if err != nil {
		return model.BinaryInfo{}, err
	}
	asset := rel.AssetByName(assetName)
	if asset == nil {
		return model.BinaryInfo{}, fmt.Errorf("上游 v%s 没有当前平台的资产 %s", ghrelease.NormalizeVersion(rel.TagName), assetName)
	}
	if progress != nil {
		progress("下载 npc", 0, asset.Size)
	}
	tmpDir := filepath.Join(m.DataDir, "updates", "npc-"+ghrelease.NormalizeVersion(rel.TagName))
	if err := platform.EnsureDir(tmpDir); err != nil {
		return model.BinaryInfo{}, err
	}
	archivePath := filepath.Join(tmpDir, assetName)
	expected := ""
	if sums, err := client.Checksums(ctx, rel); err == nil {
		expected = sums[assetName]
	}
	if expected == "" {
		expected = asset.Digest
	}
	sum, err := client.Download(ctx, *asset, archivePath, expected, func(received, total int64) {
		if progress != nil {
			progress("下载 npc", received, total)
		}
	})
	if err != nil {
		return model.BinaryInfo{}, err
	}
	if progress != nil {
		progress("解压 npc", 0, 0)
	}
	target := m.ManagedBinary(rel.TagName)
	if err := ExtractTarGz(archivePath, target); err != nil {
		return model.BinaryInfo{}, err
	}
	_ = os.Remove(archivePath)
	if progress != nil {
		progress("校验 npc", 0, 0)
	}
	info, err := Probe(ctx, target, "GUI 下载")
	if err != nil {
		_ = os.Remove(target)
		return model.BinaryInfo{}, err
	}
	info.Managed = true
	info.Tag = rel.TagName
	info.InstalledAt = time.Now().UnixMilli()
	if err := platform.AtomicWriteFile(filepath.Join(filepath.Dir(target), "meta.json"), []byte(fmt.Sprintf("{\n  \"tag\": %q,\n  \"sha256\": %q,\n  \"installedAt\": %d\n}\n", rel.TagName, sum, info.InstalledAt)), 0o600); err != nil {
		// Metadata is helpful but not required to run.
		_ = err
	}
	return info, nil
}

// ExtractTarGz extracts the npc binary from a release archive.
func ExtractTarGz(archivePath, target string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("不是有效的 tar.gz：%w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	binaryName := "npc"
	if runtime.GOOS == "windows" {
		binaryName = "npc.exe"
	}
	var found bool
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取归档失败：%w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}
		if header.Size <= 0 || header.Size > 200<<20 {
			return fmt.Errorf("归档中的 npc 大小异常")
		}
		if err := platform.EnsureDir(filepath.Dir(target)); err != nil {
			return err
		}
		tmp := target + ".part"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.CopyN(out, tr, header.Size); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("写入 npc 失败：%w", err)
		}
		if err := out.Close(); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		if err := os.Rename(tmp, target); err != nil {
			// Windows does not replace an existing file with Rename.
			_ = os.Remove(target)
			if err2 := os.Rename(tmp, target); err2 != nil {
				_ = os.Remove(tmp)
				return err2
			}
		}
		_ = os.Chmod(target, 0o755)
		found = true
		break
	}
	if !found {
		return fmt.Errorf("归档中没有找到 %s", binaryName)
	}
	return nil
}
