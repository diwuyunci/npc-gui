package ghrelease

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"npc-gui/internal/buildinfo"
)

// Asset is one GitHub release asset.
type Asset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest,omitempty"`
	ContentType        string `json:"content_type,omitempty"`
}

// Release is the subset of a GitHub release we need.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Assets      []Asset   `json:"assets"`
}

// AssetByName returns an asset by exact name.
func (r *Release) AssetByName(name string) *Asset {
	for i := range r.Assets {
		if r.Assets[i].Name == name {
			return &r.Assets[i]
		}
	}
	return nil
}

// Client talks to the GitHub Releases API.
type Client struct {
	Repo         string
	APIBase      string
	HTTP         *http.Client
	MirrorPrefix string
}

// NewClient creates a GitHub release client for owner/repo.
func NewClient(repo string) *Client {
	return &Client{
		Repo:    repo,
		APIBase: "https://api.github.com",
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) apiBase() string {
	if strings.TrimSpace(c.APIBase) == "" {
		return "https://api.github.com"
	}
	return strings.TrimRight(c.APIBase, "/")
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// Latest fetches the latest stable release.
func (c *Client) Latest(ctx context.Context) (*Release, error) {
	if strings.TrimSpace(c.Repo) == "" {
		return nil, fmt.Errorf("未配置更新仓库")
	}
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", c.apiBase(), c.Repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "npc-gui/"+buildinfo.Version)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 GitHub 发布信息失败：%w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(body, &apiErr)
		msg := strings.TrimSpace(apiErr.Message)
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("GitHub API 返回 %s：%s", resp.Status, msg)
	}
	var rel Release
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, fmt.Errorf("解析 GitHub 发布信息失败：%w", err)
	}
	if rel.Draft || rel.Prerelease {
		return nil, fmt.Errorf("latest 接口返回了非正式版本")
	}
	return &rel, nil
}

// Download fetches an asset into dest and verifies its SHA-256.
func (c *Client) Download(ctx context.Context, asset Asset, dest string, expectedSHA string, progress func(received, total int64)) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return "", err
	}
	downloadURL := c.publicURL(asset.BrowserDownloadURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "npc-gui/"+buildinfo.Version)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败：HTTP %s", resp.Status)
	}
	tmp := dest + ".part"
	file, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	total := resp.ContentLength
	if total <= 0 {
		total = asset.Size
	}
	buf := make([]byte, 128*1024)
	var received int64
	lastReport := time.Now()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, err := file.Write(chunk); err != nil {
				file.Close()
				_ = os.Remove(tmp)
				return "", err
			}
			_, _ = hasher.Write(chunk)
			received += int64(n)
			if progress != nil && time.Since(lastReport) > 100*time.Millisecond {
				lastReport = time.Now()
				progress(received, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			file.Close()
			_ = os.Remove(tmp)
			return "", fmt.Errorf("下载中断：%w", readErr)
		}
	}
	if err := file.Sync(); err != nil {
		file.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if progress != nil {
		progress(received, received)
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	expected := normalizeSHA(expectedSHA, asset.Digest)
	if expected != "" && !strings.EqualFold(expected, sum) {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("SHA-256 校验失败：期望 %s，实际 %s", expected, sum)
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return sum, nil
}

func (c *Client) publicURL(raw string) string {
	prefix := strings.TrimSpace(c.MirrorPrefix)
	if prefix == "" || !strings.Contains(raw, "github.com") {
		return raw
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + raw
}

func normalizeSHA(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		v = strings.TrimPrefix(strings.ToLower(v), "sha256:")
		if len(v) == 64 {
			if _, err := hex.DecodeString(v); err == nil {
				return v
			}
		}
	}
	return ""
}

// Checksums downloads and parses checksums.txt from a release.
func (c *Client) Checksums(ctx context.Context, rel *Release) (map[string]string, error) {
	asset := rel.AssetByName("checksums.txt")
	if asset == nil {
		return nil, nil
	}
	url := c.publicURL(asset.BrowserDownloadURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "npc-gui/"+buildinfo.Version)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载 checksums.txt 失败：HTTP %s", resp.Status)
	}
	out := map[string]string{}
	scanner := bufio.NewScanner(io.LimitReader(resp.Body, 1<<20))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		hash := strings.ToLower(fields[0])
		name := strings.TrimPrefix(fields[1], "*")
		if len(hash) == 64 {
			out[name] = hash
		}
	}
	return out, scanner.Err()
}

var numericRE = regexp.MustCompile(`^[0-9]+$`)

// NormalizeVersion strips a leading v and whitespace.
func NormalizeVersion(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), "v")
}

// CompareVersions compares two semantic-ish versions.
func CompareVersions(a, b string) int {
	pa := parseVersion(a)
	pb := parseVersion(b)
	max := len(pa)
	if len(pb) > max {
		max = len(pb)
	}
	for i := 0; i < max; i++ {
		var av, bv int
		if i < len(pa) {
			av = pa[i]
		}
		if i < len(pb) {
			bv = pb[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func parseVersion(v string) []int {
	v = NormalizeVersion(v)
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		if !numericRE.MatchString(p) {
			out = append(out, 0)
			continue
		}
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

// AssetURL returns a browser download URL for testing or display.
func AssetURL(repo, tag, name string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, url.PathEscape(tag), url.PathEscape(name))
}
