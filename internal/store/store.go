package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"npc-gui/internal/model"
	"npc-gui/internal/platform"
)

const SchemaVersion = 1

type profilesFile struct {
	SchemaVersion int             `json:"schemaVersion"`
	Profiles      []model.Profile `json:"profiles"`
}

type settingsFile struct {
	SchemaVersion int            `json:"schemaVersion"`
	Settings      model.Settings `json:"settings"`
}

type binaryFile struct {
	SchemaVersion int               `json:"schemaVersion"`
	Binary        *model.BinaryInfo `json:"binary,omitempty"`
}

// Store owns the on-disk application state.
type Store struct {
	mu       sync.RWMutex
	dir      string
	profiles []model.Profile
	settings model.Settings
	binary   *model.BinaryInfo
	warnings []string
}

// New loads or creates the state under dir.
func New(dir string) (*Store, error) {
	if err := platform.EnsureSubdirs(dir); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, settings: model.DefaultSettings()}
	s.loadProfiles()
	s.loadSettings()
	s.loadBinary()
	sort.SliceStable(s.profiles, func(i, j int) bool {
		return s.profiles[i].CreatedAt < s.profiles[j].CreatedAt
	})
	return s, nil
}

// DataDir returns the directory owned by the store.
func (s *Store) DataDir() string { return s.dir }

// Warnings returns non-fatal load problems.
func (s *Store) Warnings() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string(nil), s.warnings...)
}

func (s *Store) loadProfiles() {
	path := filepath.Join(s.dir, "profiles.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.warnings = append(s.warnings, "读取连接配置失败："+err.Error())
		}
		return
	}
	var pf profilesFile
	if err := json.Unmarshal(data, &pf); err != nil || pf.SchemaVersion > SchemaVersion {
		s.quarantine(path, "profiles")
		return
	}
	for i := range pf.Profiles {
		normalizeProfile(&pf.Profiles[i])
	}
	s.profiles = pf.Profiles
}

func (s *Store) loadSettings() {
	path := filepath.Join(s.dir, "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	sf := settingsFile{SchemaVersion: SchemaVersion, Settings: model.DefaultSettings()}
	if err := json.Unmarshal(data, &sf); err != nil || sf.SchemaVersion > SchemaVersion {
		s.quarantine(path, "settings")
		return
	}
	if sf.Settings.MaxLogLines <= 0 {
		sf.Settings.MaxLogLines = 5000
	}
	if sf.Settings.Theme == "" {
		sf.Settings.Theme = "dark"
	}
	if sf.Settings.Locale == "" {
		sf.Settings.Locale = "zh-CN"
	}
	if sf.Settings.DefaultLogLevel == "" {
		sf.Settings.DefaultLogLevel = "info"
	}
	s.settings = sf.Settings
}

func (s *Store) loadBinary() {
	path := filepath.Join(s.dir, "binary.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var bf binaryFile
	if err := json.Unmarshal(data, &bf); err != nil || bf.SchemaVersion > SchemaVersion {
		s.quarantine(path, "binary")
		return
	}
	s.binary = bf.Binary
}

func (s *Store) quarantine(path, kind string) {
	backup := fmt.Sprintf("%s.corrupt-%d", path, time.Now().Unix())
	_ = os.Rename(path, backup)
	s.warnings = append(s.warnings, fmt.Sprintf("%s 配置损坏，已备份到 %s", kind, filepath.Base(backup)))
	if data, err := os.ReadFile(path + ".bak"); err == nil {
		switch kind {
		case "profiles":
			var pf profilesFile
			if json.Unmarshal(data, &pf) == nil {
				for i := range pf.Profiles {
					normalizeProfile(&pf.Profiles[i])
				}
				s.profiles = pf.Profiles
				s.warnings = append(s.warnings, "已从备份恢复连接配置")
			}
		case "settings":
			sf := settingsFile{SchemaVersion: SchemaVersion, Settings: model.DefaultSettings()}
			if json.Unmarshal(data, &sf) == nil {
				s.settings = sf.Settings
				s.warnings = append(s.warnings, "已从备份恢复应用设置")
			}
		}
	}
}

func normalizeProfile(p *model.Profile) {
	if p.ID == "" {
		p.ID = NewID()
	}
	if p.ConnType == "" {
		p.ConnType = "tls"
	}
	if p.NTPInterval <= 0 {
		p.NTPInterval = 5
	}
	if p.CreatedAt == 0 {
		p.CreatedAt = time.Now().UnixMilli()
	}
	if p.UpdatedAt == 0 {
		p.UpdatedAt = p.CreatedAt
	}
	if p.LogLevel == "" {
		p.LogLevel = "info"
	}
}

func (s *Store) persistProfilesLocked() error {
	pf := profilesFile{SchemaVersion: SchemaVersion, Profiles: s.profiles}
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, "profiles.json")
	if err := platform.BackupFile(path); err != nil {
		s.warnings = append(s.warnings, "备份连接配置失败："+err.Error())
	}
	return platform.AtomicWriteFile(path, data, 0o600)
}

func (s *Store) persistSettingsLocked() error {
	sf := settingsFile{SchemaVersion: SchemaVersion, Settings: s.settings}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.dir, "settings.json")
	if err := platform.BackupFile(path); err != nil {
		s.warnings = append(s.warnings, "备份应用设置失败："+err.Error())
	}
	return platform.AtomicWriteFile(path, data, 0o600)
}

func (s *Store) persistBinaryLocked() error {
	bf := binaryFile{SchemaVersion: SchemaVersion, Binary: s.binary}
	data, err := json.MarshalIndent(bf, "", "  ")
	if err != nil {
		return err
	}
	return platform.AtomicWriteFile(filepath.Join(s.dir, "binary.json"), data, 0o600)
}

// NewID returns a random profile identifier.
func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("p%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// Profiles returns a copy of all profiles.
func (s *Store) Profiles() []model.Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Profile, len(s.profiles))
	copy(out, s.profiles)
	return out
}

// GetProfile returns one profile by id.
func (s *Store) GetProfile(id string) (model.Profile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.profiles {
		if p.ID == id {
			return p, true
		}
	}
	return model.Profile{}, false
}

// SaveProfile inserts or updates a profile and persists the list.
func (s *Store) SaveProfile(p model.Profile) (model.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	if p.ID == "" {
		p.ID = NewID()
		p.CreatedAt = now
	}
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	if p.ConnType == "" {
		p.ConnType = "tls"
	}
	if p.NTPInterval <= 0 {
		p.NTPInterval = 5
	}
	if p.LogLevel == "" {
		p.LogLevel = "info"
	}
	p.UpdatedAt = now

	replaced := false
	for i := range s.profiles {
		if s.profiles[i].ID == p.ID {
			s.profiles[i] = p
			replaced = true
			break
		}
	}
	if !replaced {
		s.profiles = append(s.profiles, p)
	}
	if err := s.persistProfilesLocked(); err != nil {
		return model.Profile{}, err
	}
	return p, nil
}

// AddProfiles appends new profiles with fresh ids.
func (s *Store) AddProfiles(items []model.Profile) ([]model.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	out := make([]model.Profile, 0, len(items))
	for _, p := range items {
		p.ID = NewID()
		p.CreatedAt = now
		p.UpdatedAt = now
		if p.ConnType == "" {
			p.ConnType = "tls"
		}
		if p.NTPInterval <= 0 {
			p.NTPInterval = 5
		}
		if p.LogLevel == "" {
			p.LogLevel = "info"
		}
		s.profiles = append(s.profiles, p)
		out = append(out, p)
	}
	if len(out) > 0 {
		if err := s.persistProfilesLocked(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// DeleteProfile removes a profile by id.
func (s *Store) DeleteProfile(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.profiles[:0]
	found := false
	for _, p := range s.profiles {
		if p.ID == id {
			found = true
			continue
		}
		out = append(out, p)
	}
	if !found {
		return nil
	}
	s.profiles = out
	return s.persistProfilesLocked()
}

// Settings returns the current settings.
func (s *Store) Settings() model.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// SaveSettings persists application settings.
func (s *Store) SaveSettings(v model.Settings) (model.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v.MaxLogLines <= 0 {
		v.MaxLogLines = 5000
	}
	if v.Theme == "" {
		v.Theme = "dark"
	}
	if v.Locale == "" {
		v.Locale = "zh-CN"
	}
	if v.DefaultLogLevel == "" {
		v.DefaultLogLevel = "info"
	}
	s.settings = v
	if err := s.persistSettingsLocked(); err != nil {
		return model.Settings{}, err
	}
	return v, nil
}

// Binary returns the current managed binary metadata.
func (s *Store) Binary() *model.BinaryInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.binary == nil {
		return nil
	}
	b := *s.binary
	return &b
}

// SetBinary persists binary metadata.
func (s *Store) SetBinary(info *model.BinaryInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if info == nil {
		s.binary = nil
	} else {
		c := *info
		s.binary = &c
	}
	return s.persistBinaryLocked()
}

// ProfileDir returns the directory used for one profile's generated files.
func (s *Store) ProfileDir(id string) string {
	return filepath.Join(s.dir, "profiles", id)
}

// LogDir returns the directory used for one profile's captured logs.
func (s *Store) LogDir(id string) string {
	return filepath.Join(s.dir, "logs", id)
}

// UpdatesDir returns the application update staging directory.
func (s *Store) UpdatesDir() string {
	return filepath.Join(s.dir, "updates")
}

// ConfigPath returns the generated npc.conf path for a profile.
func (s *Store) ConfigPath(id string) string {
	return filepath.Join(s.ProfileDir(id), "npc.conf")
}
