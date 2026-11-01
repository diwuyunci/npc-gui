package update

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"npc-gui/internal/platform"
)

// IsHelperInvocation reports whether args request updater helper mode.
func IsHelperInvocation(args []string) bool {
	for _, arg := range args[1:] {
		if arg == "--apply-update" || arg == "-apply-update" {
			return true
		}
	}
	return false
}

type helperConfig struct {
	Target     string
	Staged     string
	StagedApp  string
	Kind       string
	OldPID     int
	AppPath    string
	LogPath    string
	Relaunch   string
	NoRelaunch bool
}

// RunHelper executes updater helper mode and returns an exit code.
func RunHelper(args []string) int {
	cfg, err := parseHelperArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "update helper:", err)
		return 2
	}
	logFile := openHelperLog(cfg.LogPath)
	if logFile != nil {
		defer logFile.Close()
	}
	logf := func(format string, values ...any) {
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, values...))
		if logFile != nil {
			_, _ = logFile.WriteString(line)
			_ = logFile.Sync()
		}
	}
	logf("update helper start: target=%s kind=%s oldPid=%d", cfg.Target, cfg.Kind, cfg.OldPID)
	if cfg.OldPID > 0 {
		if !waitForProcessExit(cfg.OldPID, 90*time.Second) {
			err := fmt.Errorf("等待旧进程退出超时")
			logf("ERROR: %v", err)
			writeResult(cfg, false, err.Error())
			return 1
		}
	}
	if err := applyHelper(cfg, logf); err != nil {
		logf("ERROR: %v", err)
		writeResult(cfg, false, err.Error())
		return 1
	}
	if !cfg.NoRelaunch && cfg.Relaunch != "" {
		if err := relaunchTarget(cfg.Relaunch); err != nil {
			logf("relaunch failed: %v", err)
			writeResult(cfg, false, "更新完成但重启失败："+err.Error())
			return 1
		}
	}
	logf("update helper finished successfully")
	writeResult(cfg, true, "更新完成")
	return 0
}

func parseHelperArgs(args []string) (helperConfig, error) {
	var cfg helperConfig
	fs := flag.NewFlagSet("npc-gui-updater", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Bool("apply-update", false, "")
	fs.StringVar(&cfg.Target, "target", "", "")
	fs.StringVar(&cfg.Staged, "staged", "", "")
	fs.StringVar(&cfg.StagedApp, "staged-app", "", "")
	fs.StringVar(&cfg.Kind, "kind", "binary", "")
	fs.IntVar(&cfg.OldPID, "old-pid", 0, "")
	fs.StringVar(&cfg.AppPath, "app-path", "", "")
	fs.StringVar(&cfg.LogPath, "log", "", "")
	fs.StringVar(&cfg.Relaunch, "relaunch", "", "")
	fs.BoolVar(&cfg.NoRelaunch, "no-relaunch", false, "")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if strings.TrimSpace(cfg.Target) == "" {
		return cfg, fmt.Errorf("缺少 --target")
	}
	if strings.TrimSpace(cfg.Staged) == "" {
		return cfg, fmt.Errorf("缺少 --staged")
	}
	if cfg.Kind != "app" && cfg.Kind != "binary" {
		return cfg, fmt.Errorf("--kind 必须是 app 或 binary")
	}
	return cfg, nil
}

func openHelperLog(path string) *os.File {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := platform.EnsureDir(filepath.Dir(path)); err != nil {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}
	return f
}

func applyHelper(cfg helperConfig, logf func(string, ...any)) error {
	switch cfg.Kind {
	case "app":
		return applyApp(cfg, logf)
	case "binary":
		return applyBinary(cfg, logf)
	default:
		return fmt.Errorf("未知更新类型 %s", cfg.Kind)
	}
}

func applyBinary(cfg helperConfig, logf func(string, ...any)) error {
	target := cfg.Target
	backup := target + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("备份旧版本失败：%w", err)
	}
	if err := copyFile(cfg.Staged, target, 0o755); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("写入新版本失败：%w", err)
	}
	_ = os.Chmod(target, 0o755)
	logf("binary replaced: %s", target)
	return nil
}

func applyApp(cfg helperConfig, logf func(string, ...any)) error {
	if strings.TrimSpace(cfg.StagedApp) == "" {
		return fmt.Errorf("缺少新版本 .app 路径")
	}
	target := cfg.Target
	backup := target + ".old"
	_ = os.RemoveAll(backup)
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("备份旧应用失败：%w", err)
	}
	if err := copyDir(cfg.StagedApp, target); err != nil {
		_ = os.RemoveAll(target)
		_ = os.Rename(backup, target)
		return fmt.Errorf("写入新应用失败：%w", err)
	}
	logf("application replaced: %s", target)
	return nil
}

func writeResult(cfg helperConfig, ok bool, message string) {
	dir := filepath.Dir(cfg.LogPath)
	if strings.TrimSpace(cfg.LogPath) == "" {
		dir = filepath.Dir(cfg.Target)
	}
	path := filepath.Join(dir, "last-result.json")
	payload := map[string]any{
		"ok":         ok,
		"message":    message,
		"target":     cfg.Target,
		"kind":       cfg.Kind,
		"finishedAt": time.Now().UnixMilli(),
		"os":         runtime.GOOS,
	}
	data, _ := json.MarshalIndent(payload, "", "  ")
	_ = platform.AtomicWriteFile(path, data, 0o600)
}

func commandForHelper(path string, args ...string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	detachHelper(cmd)
	return cmd
}
