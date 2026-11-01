package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"npc-gui/internal/platform"
)

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := platform.EnsureDir(filepath.Dir(dst)); err != nil {
		return err
	}
	tmp := dst + ".part"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Chmod(tmp, perm)
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(dst)
		if err2 := os.Rename(tmp, dst); err2 != nil {
			_ = os.Remove(tmp)
			return err2
		}
	}
	return nil
}

func extractAppZip(archivePath, stageDir string) (string, string, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", "", fmt.Errorf("打开应用压缩包失败：%w", err)
	}
	defer zr.Close()
	root := filepath.Join(stageDir, "unpacked")
	_ = os.RemoveAll(root)
	if err := platform.EnsureDir(root); err != nil {
		return "", "", err
	}
	var total int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return "", "", fmt.Errorf("应用压缩包中包含符号链接，已拒绝")
		}
		clean := filepath.Clean(f.Name)
		if clean == "." || strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) || strings.Contains(clean, ".."+string(os.PathSeparator)) {
			return "", "", fmt.Errorf("应用压缩包路径不合法：%s", f.Name)
		}
		total += int64(f.UncompressedSize64)
		if total > 500<<20 {
			return "", "", fmt.Errorf("应用压缩包解压后过大")
		}
		dst := filepath.Join(root, clean)
		if !strings.HasPrefix(dst, root+string(os.PathSeparator)) && dst != root {
			return "", "", fmt.Errorf("应用压缩包路径越界：%s", f.Name)
		}
		rc, err := f.Open()
		if err != nil {
			return "", "", err
		}
		if err := platform.EnsureDir(filepath.Dir(dst)); err != nil {
			rc.Close()
			return "", "", err
		}
		mode := f.Mode().Perm()
		if mode == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			rc.Close()
			return "", "", err
		}
		_, copyErr := io.Copy(out, io.LimitReader(rc, int64(f.UncompressedSize64)+1))
		rc.Close()
		out.Close()
		if copyErr != nil {
			return "", "", copyErr
		}
	}
	var appDir string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || appDir != "" {
			return nil
		}
		if info.IsDir() && strings.HasSuffix(path, ".app") {
			appDir = path
			return filepath.SkipDir
		}
		return nil
	})
	if appDir == "" {
		return "", "", fmt.Errorf("应用压缩包中没有找到 .app")
	}
	bin := filepath.Join(appDir, "Contents", "MacOS")
	entries, err := os.ReadDir(bin)
	if err != nil || len(entries) == 0 {
		return "", "", fmt.Errorf("应用压缩包中没有找到可执行文件")
	}
	binPath := filepath.Join(bin, entries[0].Name())
	return appDir, binPath, nil
}

func extractTarBinary(archivePath, dest string) error {
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
	_ = platform.EnsureDir(filepath.Dir(dest))
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(h.Name)
		if base != "npc-gui" && base != "npc-gui.exe" && h.FileInfo().Mode()&0o111 == 0 {
			continue
		}
		if h.Size <= 0 || h.Size > 300<<20 {
			return fmt.Errorf("更新二进制大小异常")
		}
		tmp := dest + ".part"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.CopyN(out, tr, h.Size); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return err
		}
		if err := out.Close(); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		_ = os.Chmod(tmp, 0o755)
		if err := os.Rename(tmp, dest); err != nil {
			_ = os.Remove(dest)
			return err
		}
		return nil
	}
	return fmt.Errorf("更新包中没有找到可执行文件")
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := platform.EnsureDir(dst); err != nil {
		return err
	}
	_ = os.Chmod(dst, info.Mode().Perm())
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return platform.EnsureDir(target)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("不支持符号链接：%s", path)
		}
		return copyFile(path, target, fi.Mode().Perm())
	})
}
