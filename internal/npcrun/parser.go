package npcrun

import (
	"regexp"
	"strings"
	"time"

	"npc-gui/internal/model"
)

var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

// StripANSI removes ANSI escape sequences from console output.
func StripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// ParseLine converts one npc console line into a log entry.
func ParseLine(profileID, line, stream string, now time.Time) model.LogEntry {
	line = strings.TrimRight(line, "\r\n")
	line = StripANSI(line)
	line = strings.ToValidUTF8(line, "?")
	if len(line) > 16*1024 {
		line = line[:16*1024] + " …（已截断）"
	}
	ts := now
	level := "info"
	text := strings.TrimSpace(line)

	fields := strings.Fields(text)
	if len(fields) > 0 {
		if parsed, ok := parseLogTime(fields[0]); ok {
			ts = parsed
			fields = fields[1:]
		}
	}
	if len(fields) > 0 {
		if lvl, ok := levelFromToken(fields[0]); ok {
			level = lvl
			fields = fields[1:]
		}
	}
	if len(fields) > 0 {
		text = strings.Join(fields, " ")
	} else {
		text = line
	}
	return model.LogEntry{
		ProfileID: profileID,
		Time:      ts.UnixMilli(),
		Level:     level,
		Text:      text,
		Stream:    stream,
	}
}

func parseLogTime(token string) (time.Time, bool) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z0700",
		"2006/01/02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, token); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func levelFromToken(token string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(token)) {
	case "TRC", "TRACE":
		return "trace", true
	case "DBG", "DEBUG":
		return "debug", true
	case "INF", "INFO":
		return "info", true
	case "WRN", "WARN", "WARNING":
		return "warn", true
	case "ERR", "ERROR":
		return "error", true
	case "FTL", "FATAL":
		return "fatal", true
	case "PNC", "PANIC":
		return "panic", true
	}
	return "", false
}

// StateMachine derives a runtime state from process events and npc log lines.
type StateMachine struct {
	State       string
	StartedAt   int64
	OnlineSince int64
	LastError   string
	Reconnects  int
	LastLogAt   int64
	PID         int
	ConfigHash  string
	BinaryVer   string
}

// NewStateMachine returns a stopped state machine.
func NewStateMachine() *StateMachine {
	return &StateMachine{State: model.StateStopped}
}

// Start marks the process as starting.
func (s *StateMachine) Start(pid int, configHash, binaryVersion string) model.RuntimeInfo {
	s.PID = pid
	s.State = model.StateRunning
	s.StartedAt = model.NowMillis()
	s.OnlineSince = 0
	s.LastError = ""
	s.Reconnects = 0
	s.ConfigHash = configHash
	s.BinaryVer = binaryVersion
	return s.Snapshot("")
}

// ObserveLine updates the state machine from one log line.
func (s *StateMachine) ObserveLine(level, text string, now int64) (changed bool, info model.RuntimeInfo) {
	s.LastLogAt = now
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "successful connection with server"):
		if s.State != model.StateOnline {
			if s.State == model.StateReconnecting {
				s.Reconnects++
			}
			s.State = model.StateOnline
			s.OnlineSince = now
			s.LastError = ""
			return true, s.Snapshot("")
		}
	case strings.Contains(lower, "reconnecting...") || strings.Contains(lower, "will be reconnected in five seconds"):
		if s.State != model.StateReconnecting {
			s.State = model.StateReconnecting
			s.LastError = strings.TrimSpace(text)
			return true, s.Snapshot("")
		}
	case strings.Contains(lower, "the connection server failed") ||
		strings.Contains(lower, "failed to connect") ||
		strings.Contains(lower, "connection server failed") ||
		strings.Contains(lower, "failed to send type") ||
		strings.Contains(lower, "accept server data error"):
		if s.State == model.StateOnline || s.State == model.StateRunning || s.State == model.StateStarting {
			s.State = model.StateReconnecting
			s.LastError = strings.TrimSpace(text)
			return true, s.Snapshot("")
		}
	case strings.Contains(lower, "loading error") ||
		strings.Contains(lower, "not allowed to be duplicated") ||
		strings.Contains(lower, "missing [common]") ||
		strings.Contains(lower, "web_user may have been occupied"):
		s.State = model.StateError
		s.LastError = strings.TrimSpace(text)
		return true, s.Snapshot("")
	case level == "panic" || level == "fatal":
		s.State = model.StateError
		s.LastError = strings.TrimSpace(text)
		return true, s.Snapshot("")
	}
	return false, model.RuntimeInfo{}
}

// Stop marks the process as stopping.
func (s *StateMachine) Stop() model.RuntimeInfo {
	if s.State != model.StateStopped {
		s.State = model.StateStopping
	}
	return s.Snapshot("")
}

// Exit finishes the state after the process exits.
func (s *StateMachine) Exit(userStop bool, exitCode int, message string) model.RuntimeInfo {
	if userStop {
		s.State = model.StateStopped
		s.LastError = ""
	} else {
		s.State = model.StateError
		if strings.TrimSpace(message) != "" {
			s.LastError = message
		} else if exitCode != 0 {
			s.LastError = "npc 进程已退出，退出码 " + itoa(exitCode)
		} else {
			s.LastError = "npc 进程已退出"
		}
	}
	s.PID = 0
	s.OnlineSince = 0
	return s.Snapshot("")
}

// Snapshot returns a copy of the current runtime state.
func (s *StateMachine) Snapshot(profileID string) model.RuntimeInfo {
	return model.RuntimeInfo{
		ProfileID:     profileID,
		State:         s.State,
		PID:           s.PID,
		StartedAt:     s.StartedAt,
		OnlineSince:   s.OnlineSince,
		LastLogAt:     s.LastLogAt,
		LastError:     s.LastError,
		Reconnects:    s.Reconnects,
		ConfigHash:    s.ConfigHash,
		BinaryVersion: s.BinaryVer,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
