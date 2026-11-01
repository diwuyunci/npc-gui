package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func makeAsset(t *testing.T, name string) []byte {
	t.Helper()
	var buf bytes.Buffer
	binary := []byte("new-binary")
	switch {
	case runtime.GOOS == "darwin":
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("Test.app/Contents/MacOS/npc-gui")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write(binary)
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	case runtime.GOOS == "windows":
		buf.Write(binary)
	default:
		gz := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gz)
		_ = tw.WriteHeader(&tar.Header{Name: "npc-gui", Mode: 0o755, Size: int64(len(binary))})
		_, _ = tw.Write(binary)
		_ = tw.Close()
		_ = gz.Close()
	}
	return buf.Bytes()
}

func TestManagerCheckDownload(t *testing.T) {
	assetName := appAssetName("0.2.0", runtime.GOOS, runtime.GOARCH)
	asset := makeAsset(t, assetName)
	sum := sha256.Sum256(asset)
	digest := hex.EncodeToString(sum[:])

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/repo/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name":     "v0.2.0",
				"name":         "test release",
				"body":         "notes",
				"draft":        false,
				"prerelease":   false,
				"published_at": time.Now().UTC(),
				"html_url":     "https://example.invalid/release",
				"assets": []map[string]any{{
					"name":                 assetName,
					"size":                 len(asset),
					"browser_download_url": server.URL + "/asset",
					"digest":               "sha256:" + digest,
				}},
			})
		case "/asset":
			_, _ = w.Write(asset)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("NPC_GUI_UPDATE_REPO", "test/repo")
	t.Setenv("NPC_GUI_UPDATE_API", server.URL)
	m := New(t.TempDir(), nil, nil)
	if !m.Enabled() {
		t.Fatal("update manager should be enabled in dev override")
	}
	state, err := m.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if state.State != "available" || state.Latest == nil || state.Latest.Version != "0.2.0" {
		t.Fatalf("state after check = %+v", state)
	}
	state, err = m.Download(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.State != "ready" || state.StagedVersion != "0.2.0" {
		t.Fatalf("state after download = %+v", state)
	}
	staged, err := readStaged(m.dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(staged.StagedBin); err != nil {
		t.Fatalf("staged binary missing: %v", err)
	}
	helper := filepath.Join(m.dataDir, "updates", "0.2.0", helperBinaryName())
	if _, err := os.Stat(helper); err != nil {
		t.Fatalf("helper missing: %v", err)
	}
}
