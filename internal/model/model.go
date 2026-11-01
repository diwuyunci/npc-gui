package model

import "time"

const (
	AppName = "NPC 客户端"

	StateStopped      = "stopped"
	StateStarting     = "starting"
	StateRunning      = "running"
	StateOnline       = "online"
	StateReconnecting = "reconnecting"
	StateStopping     = "stopping"
	StateError        = "error"

	UpdateDisabled    = "disabled"
	UpdateIdle        = "idle"
	UpdateChecking    = "checking"
	UpdateAvailable   = "available"
	UpdateDownloading = "downloading"
	UpdateReady       = "ready"
	UpdateApplying    = "applying"
	UpdateError       = "error"
)

// Profile is one npc connection profile. Every profile owns one npc process.
type Profile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Server   string `json:"server"`
	VKey     string `json:"vkey"`
	ConnType string `json:"connType"`

	LocalIP           string `json:"localIp,omitempty"`
	ProxyURL          string `json:"proxyUrl,omitempty"`
	DNSServer         string `json:"dnsServer,omitempty"`
	NTPServer         string `json:"ntpServer,omitempty"`
	NTPInterval       int    `json:"ntpInterval,omitempty"`
	AutoReconnection  bool   `json:"autoReconnection"`
	DisconnectTimeout int    `json:"disconnectTimeout,omitempty"`

	ExtraCommon   string `json:"extraCommon,omitempty"`
	ExtraSections string `json:"extraSections,omitempty"`

	AutoStart bool   `json:"autoStart,omitempty"`
	LogLevel  string `json:"logLevel,omitempty"`

	Crypt     bool   `json:"crypt,omitempty"`
	Compress  bool   `json:"compress,omitempty"`
	MaxConn   int    `json:"maxConn,omitempty"`
	RateLimit int    `json:"rateLimit,omitempty"`
	FlowLimit int    `json:"flowLimit,omitempty"`
	Remark    string `json:"remark,omitempty"`

	BasicUsername string `json:"basicUsername,omitempty"`
	BasicPassword string `json:"basicPassword,omitempty"`
	WebUsername   string `json:"webUsername,omitempty"`
	WebPassword   string `json:"webPassword,omitempty"`

	PprofAddr string `json:"pprofAddr,omitempty"`

	ProtoVersion  *int `json:"protoVersion,omitempty"`
	SkipTLSVerify bool `json:"skipTlsVerify,omitempty"`
	DisableP2P    bool `json:"disableP2p,omitempty"`
	KeepAlive     int  `json:"keepalive,omitempty"`

	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
}

// Settings are application-level preferences.
type Settings struct {
	NpcBinaryPath    string `json:"npcBinaryPath,omitempty"`
	DownloadMirror   string `json:"downloadMirror,omitempty"`
	AutoCheckUpdates bool   `json:"autoCheckUpdates"`
	Theme            string `json:"theme"`
	Locale           string `json:"locale"`
	DefaultLogLevel  string `json:"defaultLogLevel"`
	MaxLogLines      int    `json:"maxLogLines"`
	ConfirmOnQuit    bool   `json:"confirmOnQuit"`
	LogToDisk        bool   `json:"logToDisk"`
	LaunchAtLogin    bool   `json:"launchAtLogin"`
	TrayEnabled      bool   `json:"trayEnabled"`
	MinimizeToTray   bool   `json:"minimizeToTray"`
	Notifications    bool   `json:"notifications"`
	LastUpdateCheck  int64  `json:"lastUpdateCheck,omitempty"`
}

// DefaultSettings returns the settings used on first launch.
func DefaultSettings() Settings {
	return Settings{
		AutoCheckUpdates: true,
		Theme:            "dark",
		Locale:           "zh-CN",
		DefaultLogLevel:  "info",
		MaxLogLines:      5000,
		ConfirmOnQuit:    true,
		LogToDisk:        true,
		LaunchAtLogin:    false,
		TrayEnabled:      true,
		MinimizeToTray:   true,
		Notifications:    true,
	}
}

// NewProfile returns a profile with sensible defaults.
func NewProfile() Profile {
	return Profile{
		ConnType:         "tls",
		DNSServer:        "8.8.8.8",
		NTPInterval:      5,
		AutoReconnection: true,
		LogLevel:         "info",
	}
}

// LogLevelOr returns the profile log level or a fallback.
func (p Profile) LogLevelOr(fallback string) string {
	if p.LogLevel != "" {
		return p.LogLevel
	}
	if fallback != "" {
		return fallback
	}
	return "info"
}

// DisplayName returns a non-empty label for lists and events.
func (p Profile) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	if p.Server != "" {
		return p.Server
	}
	return "未命名连接"
}

// BinaryInfo describes a discovered or downloaded npc executable.
type BinaryInfo struct {
	Path        string `json:"path"`
	Version     string `json:"version,omitempty"`
	CoreVersion string `json:"coreVersion,omitempty"`
	Source      string `json:"source,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	Tag         string `json:"tag,omitempty"`
	Managed     bool   `json:"managed"`
	InstalledAt int64  `json:"installedAt,omitempty"`
}

// RuntimeInfo is the live state of one npc process.
type RuntimeInfo struct {
	ProfileID      string `json:"profileId"`
	State          string `json:"state"`
	PID            int    `json:"pid"`
	StartedAt      int64  `json:"startedAt"`
	OnlineSince    int64  `json:"onlineSince"`
	LastLogAt      int64  `json:"lastLogAt"`
	LastError      string `json:"lastError"`
	Reconnects     int    `json:"reconnects"`
	ConfigHash     string `json:"configHash"`
	BinaryVersion  string `json:"binaryVersion"`
	PendingRestart bool   `json:"pendingRestart"`
}

// LogEntry is one parsed npc log line.
type LogEntry struct {
	Seq       uint64 `json:"seq"`
	ProfileID string `json:"profileId,omitempty"`
	Time      int64  `json:"time"`
	Level     string `json:"level"`
	Text      string `json:"text"`
	Stream    string `json:"stream,omitempty"`
}

// LogPage is a batch of log entries plus the current bounds.
type LogPage struct {
	ProfileID string     `json:"profileId,omitempty"`
	Entries   []LogEntry `json:"entries"`
	LastSeq   uint64     `json:"lastSeq"`
	Truncated bool       `json:"truncated"`
}

// OperationResult is returned for expected validation and lifecycle failures.
type OperationResult struct {
	OK          bool              `json:"ok"`
	Code        string            `json:"code,omitempty"`
	Message     string            `json:"message,omitempty"`
	ProfileID   string            `json:"profileId,omitempty"`
	FieldErrors map[string]string `json:"fieldErrors,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
}

// OKResult returns a successful operation result.
func OKResult(warnings ...string) OperationResult {
	return OperationResult{OK: true, Warnings: warnings}
}

// FailResult returns a failed operation result.
func FailResult(code, message string) OperationResult {
	return OperationResult{OK: false, Code: code, Message: message}
}

// FieldResult returns a failed operation result with field-level errors.
func FieldResult(fields map[string]string, warnings ...string) OperationResult {
	return OperationResult{OK: false, Code: "validation", Message: "请检查表单", FieldErrors: fields, Warnings: warnings}
}

// AppInfo describes the GUI build.
type AppInfo struct {
	ProductName   string `json:"productName"`
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	BuildDate     string `json:"buildDate"`
	Repo          string `json:"repo,omitempty"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	UpdateEnabled bool   `json:"updateEnabled"`
	DataDir       string `json:"dataDir"`
}

// Bootstrap is the single snapshot requested by the frontend at startup.
type Bootstrap struct {
	App              AppInfo         `json:"app"`
	Profiles         []Profile       `json:"profiles"`
	Runtimes         []RuntimeInfo   `json:"runtimes"`
	Settings         Settings        `json:"settings"`
	Binary           *BinaryInfo     `json:"binary,omitempty"`
	BinaryCandidates []BinaryInfo    `json:"binaryCandidates"`
	Update           UpdateStateInfo `json:"update"`
	Warnings         []string        `json:"warnings"`
	FirstRun         bool            `json:"firstRun"`
}

// ConfigPreview is the generated npc.conf and its launch command.
type ConfigPreview struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Command string `json:"command"`
}

// ImportCandidate is one profile parsed from a pasted NPS command.
type ImportCandidate struct {
	Profile  Profile `json:"profile"`
	Selected bool    `json:"selected"`
	Note     string  `json:"note,omitempty"`
}

// ImportPreview is the result of parsing an NPS launch command.
type ImportPreview struct {
	Candidates []ImportCandidate `json:"candidates"`
	Warnings   []string          `json:"warnings"`
	Errors     []string          `json:"errors"`
}

// UpdateInfo describes one available application release.
type UpdateInfo struct {
	Version     string `json:"version"`
	Tag         string `json:"tag"`
	Name        string `json:"name,omitempty"`
	Notes       string `json:"notes,omitempty"`
	ReleaseURL  string `json:"releaseUrl,omitempty"`
	AssetName   string `json:"assetName,omitempty"`
	AssetSize   int64  `json:"assetSize,omitempty"`
	DownloadURL string `json:"downloadUrl,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	PublishedAt int64  `json:"publishedAt,omitempty"`
}

// UpdateProgress describes a running download.
type UpdateProgress struct {
	Stage       string  `json:"stage"`
	Received    int64   `json:"received"`
	Total       int64   `json:"total"`
	Percent     float64 `json:"percent"`
	BytesPerSec int64   `json:"bytesPerSec,omitempty"`
}

// UpdateStateInfo is the current state of the application updater.
type UpdateStateInfo struct {
	State          string          `json:"state"`
	CurrentVersion string          `json:"currentVersion"`
	Latest         *UpdateInfo     `json:"latest,omitempty"`
	Progress       *UpdateProgress `json:"progress,omitempty"`
	Error          string          `json:"error,omitempty"`
	CheckedAt      int64           `json:"checkedAt,omitempty"`
	StagedVersion  string          `json:"stagedVersion,omitempty"`
}

// UpdateApplyResult describes the outcome of staging an update for restart.
type UpdateApplyResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Version string `json:"version,omitempty"`
}

// NowMillis returns the current Unix time in milliseconds.
func NowMillis() int64 { return time.Now().UnixMilli() }
