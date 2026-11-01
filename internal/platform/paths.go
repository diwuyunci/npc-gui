package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DataDir returns the per-user application data directory.
func DataDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("NPC_GUI_DATA_DIR")); custom != "" {
		return filepath.Abs(custom)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "npc-gui"), nil
}

// EnsureDir creates a directory with 0700 permissions.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	// Tighten an existing directory too.
	_ = os.Chmod(path, 0o700)
	return nil
}

// EnsureSubdirs creates the application's standard directory layout.
func EnsureSubdirs(dataDir string) error {
	for _, name := range []string{"", "profiles", "logs", "bin", "updates", "runtime"} {
		if err := EnsureDir(filepath.Join(dataDir, name)); err != nil {
			return err
		}
	}
	return nil
}

// AtomicWriteFile writes data to path via a temporary file and rename.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if err := tmp.Chmod(perm); err != nil {
		cleanup()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		// Windows does not replace an existing file with Rename.
		if runtime.GOOS == "windows" {
			_ = os.Remove(path)
			if err2 := os.Rename(tmpName, path); err2 == nil {
				return nil
			}
		}
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// BackupFile copies path to path+".bak" if it exists.
func BackupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return AtomicWriteFile(path+".bak", data, info.Mode().Perm())
}

// SHA256 returns the hex digest of data or a file.
func SHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SHA256File returns the hex digest of a file.
func SHA256File(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return SHA256(data), nil
}

// OpenPath opens a directory or file with the platform file manager.
func OpenPath(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// QuoteCommand formats a command for display in the UI. It is not used for execution.
func QuoteCommand(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		switch {
		case p == "":
			out = append(out, "''")
		case strings.ContainsAny(p, " \t\"'$"):
			out = append(out, "'"+strings.ReplaceAll(p, "'", "'\\''")+"'")
		default:
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}
