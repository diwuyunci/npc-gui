package ghrelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.0", "0.1.0", 0},
		{"0.1.0", "0.2.0", -1},
		{"1.0.0", "0.9.9", 1},
		{"0.1.0-dev", "0.1.0", 0},
		{"", "0.1.0", -1},
	}
	for _, c := range cases {
		if got := CompareVersions(c.a, c.b); got != c.want {
			t.Errorf("CompareVersions(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestDownloadChecksum(t *testing.T) {
	payload := []byte("hello release asset")
	sum := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	client := NewClient("example/repo")
	asset := Asset{Name: "file", Size: int64(len(payload)), BrowserDownloadURL: server.URL + "/file"}
	dest := filepath.Join(t.TempDir(), "file")
	digest := hex.EncodeToString(sum[:])
	if _, err := client.Download(context.Background(), asset, dest, digest, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != string(payload) {
		t.Fatalf("downloaded data = %q, err=%v", data, err)
	}
	dest2 := filepath.Join(t.TempDir(), "file2")
	if _, err := client.Download(context.Background(), asset, dest2, strings.Repeat("a", 64), nil); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestChecksums(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  app.zip\n"))
	}))
	defer server.Close()
	rel := &Release{Assets: []Asset{{Name: "checksums.txt", BrowserDownloadURL: server.URL + "/checksums.txt"}}}
	client := NewClient("example/repo")
	sums, err := client.Checksums(context.Background(), rel)
	if err != nil {
		t.Fatal(err)
	}
	if sums["app.zip"] != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("sums = %#v", sums)
	}
}

func TestAssetByName(t *testing.T) {
	rel := &Release{Assets: []Asset{{Name: "a"}, {Name: "b"}}}
	if rel.AssetByName("b") == nil || rel.AssetByName("c") != nil {
		t.Fatal("asset lookup failed")
	}
}
