package npcbin

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestAssetNameFor(t *testing.T) {
	cases := map[string]string{
		"darwin/arm64":  "darwin_arm64_client.tar.gz",
		"windows/amd64": "windows_amd64_client.tar.gz",
		"linux/arm":     "linux_arm_v7_client.tar.gz",
	}
	for platform, want := range cases {
		var goos, goarch string
		for i := 0; i < len(platform); i++ {
			if platform[i] == '/' {
				goos, goarch = platform[:i], platform[i+1:]
				break
			}
		}
		got, err := AssetNameFor(goos, goarch)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("AssetNameFor(%s) = %q, want %q", platform, got, want)
		}
	}
}

func TestParseVersionOutput(t *testing.T) {
	version, core := ParseVersionOutput("Version: 0.34.10\nCore version: 0.34.0\nSame core version...")
	if version != "0.34.10" || core != "0.34.0" {
		t.Fatalf("version=%q core=%q", version, core)
	}
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "client.tar.gz")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	content := []byte("#!/bin/sh\necho npc\n")
	if err := tw.WriteHeader(&tar.Header{Name: "npc", Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	conf := []byte("server_addr=1:1\n")
	if err := tw.WriteHeader(&tar.Header{Name: "conf/npc.conf", Mode: 0o644, Size: int64(len(conf))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(conf); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "bin", "npc")
	if err := ExtractTarGz(archive, target); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(content) {
		t.Fatalf("extracted content = %q", data)
	}
	info, _ := os.Stat(target)
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("binary is not executable: %v", info.Mode())
	}
}

func TestExtractTarGzMissingBinary(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "client.tar.gz")
	file, _ := os.Create(archive)
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "conf/npc.conf", Mode: 0o644, Size: 1})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gz.Close()
	_ = file.Close()
	if err := ExtractTarGz(archive, filepath.Join(dir, "npc")); err == nil {
		t.Fatal("expected missing binary error")
	}
}
