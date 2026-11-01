package configgen

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"npc-gui/internal/model"
	"npc-gui/internal/platform"
)

const (
	maxExtraCommon   = 64 * 1024
	maxExtraSections = 128 * 1024
)

var (
	sectionRE = regexp.MustCompile(`^\[([^\[\]]+)\]$`)
	keyRE     = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
)

var coreKeys = map[string]bool{
	"server_addr": true, "conn_type": true, "vkey": true,
	"auto_reconnection": true, "dns_server": true, "local_ip": true,
	"proxy_url": true, "ntp_server": true, "ntp_interval": true,
	"disconnect_timeout": true, "basic_username": true, "basic_password": true,
	"web_username": true, "web_password": true, "crypt": true, "compress": true,
	"max_conn": true, "rate_limit": true, "flow_limit": true, "remark": true,
	"pprof_addr": true,
}

var validConnTypes = map[string]bool{
	"tcp": true, "tls": true, "kcp": true, "quic": true, "ws": true, "wss": true,
}

var validLogLevels = map[string]bool{
	"trace": true, "debug": true, "info": true, "warn": true, "error": true,
}

// Validate checks a profile before it is persisted or started.
func Validate(p model.Profile) model.OperationResult {
	fields := map[string]string{}
	warnings := []string{}

	if strings.TrimSpace(p.Name) == "" {
		fields["name"] = "请填写档案名称"
	}
	if len([]rune(p.Name)) > 64 {
		fields["name"] = "名称最多 64 个字符"
	}
	if strings.TrimSpace(p.Server) == "" {
		fields["server"] = "请填写服务端地址"
	} else if err := validateServer(p.Server); err != nil {
		fields["server"] = err.Error()
	}
	if strings.TrimSpace(p.VKey) == "" {
		fields["vkey"] = "请填写 vkey"
	} else if strings.ContainsAny(p.VKey, "\r\n") {
		fields["vkey"] = "vkey 不能包含换行"
	}
	if p.ConnType == "" {
		fields["connType"] = "请选择连接方式"
	} else if !validConnTypes[strings.ToLower(p.ConnType)] {
		fields["connType"] = "连接方式必须是 tcp/tls/kcp/quic/ws/wss"
	}
	if p.ProxyURL != "" {
		if err := validateProxyURL(p.ProxyURL); err != nil {
			fields["proxyUrl"] = err.Error()
		}
	}
	if p.LocalIP != "" {
		for _, item := range splitList(p.LocalIP) {
			if net.ParseIP(item) == nil && !isHostname(item) {
				warnings = append(warnings, "local_ip 不是标准 IP："+item)
			}
		}
	}
	if p.NTPInterval < 0 {
		fields["ntpInterval"] = "NTP 间隔不能为负数"
	}
	if p.DisconnectTimeout < 0 {
		fields["disconnectTimeout"] = "断线超时必须大于等于 0"
	}
	if p.MaxConn < 0 || p.RateLimit < 0 || p.FlowLimit < 0 || p.KeepAlive < 0 {
		fields["__limits"] = "连接/速率/流量/保活参数不能为负数"
	}
	if p.ProtoVersion != nil && (*p.ProtoVersion < 0 || *p.ProtoVersion > 32) {
		fields["protoVersion"] = "协议版本必须在 0-32 之间"
	}
	if p.LogLevel != "" && !validLogLevels[strings.ToLower(p.LogLevel)] {
		fields["logLevel"] = "日志等级必须是 trace/debug/info/warn/error"
	}
	if p.SkipTLSVerify {
		warnings = append(warnings, "已开启跳过 TLS 证书校验，仅建议在可信内网使用")
	}
	if p.DisableP2P {
		warnings = append(warnings, "已禁用 P2P，所有流量将走服务器中转")
	}

	if err := validateExtraCommon(p.ExtraCommon, coreKeys); err != nil {
		fields["extraCommon"] = err.Error()
	}
	if err := validateExtraSections(p.ExtraSections); err != nil {
		fields["extraSections"] = err.Error()
	}
	if len(fields) > 0 {
		return model.FieldResult(fields, warnings...)
	}
	return model.OKResult(warnings...)
}

func validateServer(server string) error {
	s := strings.TrimSpace(server)
	if s == "" {
		return fmt.Errorf("请填写服务端地址")
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return fmt.Errorf("服务端地址不能包含空格或换行")
	}
	if strings.Contains(s, "://") {
		return fmt.Errorf("请填写 host:port，不要带协议前缀")
	}
	if strings.Contains(s, ",") || strings.Contains(s, "，") {
		return fmt.Errorf("一个档案只能填写一个服务端，多个服务端请创建多个档案")
	}
	hostPart := s
	if i := strings.IndexByte(hostPart, '@'); i >= 0 {
		hostPart = hostPart[:i]
	}
	if i := strings.IndexByte(hostPart, '/'); i >= 0 {
		hostPart = hostPart[:i]
	}
	if hostPart == "" {
		return fmt.Errorf("服务端地址缺少主机")
	}
	if host, port, err := net.SplitHostPort(hostPart); err == nil {
		if host == "" {
			return fmt.Errorf("服务端地址缺少主机")
		}
		if n, err := strconv.Atoi(port); err != nil || n <= 0 || n > 65535 {
			return fmt.Errorf("端口必须在 1-65535 之间")
		}
	}
	return nil
}

func isHostname(s string) bool {
	if len(s) == 0 || len(s) > 253 {
		return false
	}
	for _, part := range strings.Split(s, ".") {
		if part == "" || len(part) > 63 {
			return false
		}
		for _, r := range part {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-') {
				return false
			}
		}
	}
	return true
}

func validateProxyURL(raw string) error {
	if strings.ContainsAny(raw, "\r\n") {
		return fmt.Errorf("代理地址不能包含换行")
	}
	if !strings.Contains(raw, "://") {
		if _, _, err := net.SplitHostPort(raw); err != nil {
			return fmt.Errorf("代理地址格式应为 socks5://user:pass@host:port 或 host:port")
		}
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("代理地址无法解析")
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https", "socks4", "socks4a", "socks5", "socks5h":
	default:
		return fmt.Errorf("代理协议仅支持 http/https/socks4/socks4a/socks5/socks5h")
	}
	if u.Host == "" {
		return fmt.Errorf("代理地址缺少主机")
	}
	return nil
}

func validateExtraCommon(raw string, reserved map[string]bool) error {
	if len(raw) > maxExtraCommon {
		return fmt.Errorf("[common] 扩展内容过大")
	}
	seen := map[string]bool{}
	for lineNo, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("第 %d 行缺少等号", lineNo+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || !keyRE.MatchString(key) {
			return fmt.Errorf("第 %d 行的键名不合法", lineNo+1)
		}
		if reserved[key] {
			return fmt.Errorf("%s 由基础表单管理，不能在扩展 common 中重复设置", key)
		}
		if seen[key] {
			return fmt.Errorf("扩展 common 中重复的键：%s", key)
		}
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("第 %d 行的值不能包含换行", lineNo+1)
		}
		seen[key] = true
	}
	return nil
}

func validateExtraSections(raw string) error {
	if len(raw) > maxExtraSections {
		return fmt.Errorf("扩展段落内容过大")
	}
	seen := map[string]bool{}
	inSection := false
	for lineNo, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			m := sectionRE.FindStringSubmatch(line)
			if m == nil {
				return fmt.Errorf("第 %d 行的段落头不合法", lineNo+1)
			}
			name := strings.TrimSpace(m[1])
			lower := strings.ToLower(name)
			if lower == "common" {
				return fmt.Errorf("[common] 由基础表单生成，不能在扩展段落中重复")
			}
			if strings.ContainsAny(name, "\r\n[]") {
				return fmt.Errorf("第 %d 行的段落名不合法", lineNo+1)
			}
			if seen[lower] {
				return fmt.Errorf("重复的段落名：[%s]", name)
			}
			seen[lower] = true
			inSection = true
			continue
		}
		if !inSection {
			return fmt.Errorf("第 %d 行不属于任何段落", lineNo+1)
		}
		key, _, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return fmt.Errorf("第 %d 行缺少键值等号", lineNo+1)
		}
	}
	return nil
}

func splitList(v string) []string {
	v = strings.ReplaceAll(v, "，", ",")
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Generate renders the npc.conf content for a profile.
func Generate(p model.Profile) (string, error) {
	var b strings.Builder
	b.WriteString("# Generated by NPC GUI. Manual changes will be overwritten.\n")
	b.WriteString("[common]\n")
	writeKV(&b, "server_addr", strings.TrimSpace(p.Server))
	writeKV(&b, "conn_type", strings.ToLower(strings.TrimSpace(p.ConnType)))
	writeKV(&b, "vkey", strings.TrimSpace(p.VKey))
	writeBool(&b, "auto_reconnection", p.AutoReconnection)
	if strings.TrimSpace(p.DNSServer) != "" {
		writeKV(&b, "dns_server", strings.TrimSpace(p.DNSServer))
	}
	if strings.TrimSpace(p.ProxyURL) != "" {
		writeKV(&b, "proxy_url", strings.TrimSpace(p.ProxyURL))
	}
	if strings.TrimSpace(p.LocalIP) != "" {
		writeKV(&b, "local_ip", strings.TrimSpace(p.LocalIP))
	}
	if strings.TrimSpace(p.NTPServer) != "" {
		writeKV(&b, "ntp_server", strings.TrimSpace(p.NTPServer))
	}
	if p.NTPInterval > 0 {
		writeKV(&b, "ntp_interval", strconv.Itoa(p.NTPInterval))
	}
	if p.DisconnectTimeout > 0 {
		writeKV(&b, "disconnect_timeout", strconv.Itoa(p.DisconnectTimeout))
	}
	if p.BasicUsername != "" {
		writeKV(&b, "basic_username", p.BasicUsername)
	}
	if p.BasicPassword != "" {
		writeKV(&b, "basic_password", p.BasicPassword)
	}
	if p.WebUsername != "" {
		writeKV(&b, "web_username", p.WebUsername)
	}
	if p.WebPassword != "" {
		writeKV(&b, "web_password", p.WebPassword)
	}
	if p.Crypt {
		writeBool(&b, "crypt", true)
	}
	if p.Compress {
		writeBool(&b, "compress", true)
	}
	if p.MaxConn > 0 {
		writeKV(&b, "max_conn", strconv.Itoa(p.MaxConn))
	}
	if p.RateLimit > 0 {
		writeKV(&b, "rate_limit", strconv.Itoa(p.RateLimit))
	}
	if p.FlowLimit > 0 {
		writeKV(&b, "flow_limit", strconv.Itoa(p.FlowLimit))
	}
	if p.Remark != "" {
		writeKV(&b, "remark", p.Remark)
	}
	if p.PprofAddr != "" {
		writeKV(&b, "pprof_addr", p.PprofAddr)
	}
	if strings.TrimSpace(p.ExtraCommon) != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimRight(strings.ReplaceAll(p.ExtraCommon, "\r\n", "\n"), "\n"))
		b.WriteString("\n")
	}
	if strings.TrimSpace(p.ExtraSections) != "" {
		b.WriteString("\n")
		b.WriteString(strings.TrimRight(strings.ReplaceAll(p.ExtraSections, "\r\n", "\n"), "\n"))
		b.WriteString("\n")
	}
	return b.String(), nil
}

func writeKV(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString("=")
	b.WriteString(value)
	b.WriteString("\n")
}

func writeBool(b *strings.Builder, key string, value bool) {
	writeKV(b, key, strconv.FormatBool(value))
}

// BuildArgs returns the npc invocation for a generated config.
func BuildArgs(p model.Profile, configPath, defaultLogLevel string) []string {
	level := strings.ToLower(strings.TrimSpace(p.LogLevel))
	if level == "" {
		level = strings.ToLower(strings.TrimSpace(defaultLogLevel))
	}
	if !validLogLevels[level] {
		level = "info"
	}
	args := []string{
		"-config=" + configPath,
		"-log=stdout",
		"-debug=false",
		"-log_color=false",
		"-log_level=" + level,
		"-disable_remote_update=true",
	}
	if p.ProtoVersion != nil {
		args = append(args, "-proto_version="+strconv.Itoa(*p.ProtoVersion))
	}
	if p.SkipTLSVerify {
		args = append(args, "-skip_verify=true")
	}
	if p.DisableP2P {
		args = append(args, "-disable_p2p=true")
	}
	if p.KeepAlive > 0 {
		args = append(args, "-keepalive="+strconv.Itoa(p.KeepAlive))
	}
	return args
}

// FormatCommand returns a display-only command string.
func FormatCommand(npcPath string, args []string) string {
	return platform.QuoteCommand(append([]string{npcPath}, args...))
}

// ConfigHash returns a stable hash of generated config content.
func ConfigHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// WriteConfig writes npc.conf atomically with 0600 permissions.
func WriteConfig(path, content string) error {
	if err := platform.EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return platform.AtomicWriteFile(path, []byte(content), 0o600)
}

// MaskSecrets masks secret values in a config preview.
func MaskSecrets(content string) string {
	maskKeys := []string{"vkey", "basic_password", "web_password", "proxy_url", "password", "secret", "fallback_secret"}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		key, _, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(strings.ToLower(key))
		for _, mk := range maskKeys {
			if key == mk {
				lines[i] = strings.SplitN(line, "=", 2)[0] + "=********"
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

// Preview builds a config preview and launch command.
func Preview(p model.Profile, configPath, npcPath, defaultLogLevel string, reveal bool) (model.ConfigPreview, error) {
	content, err := Generate(p)
	if err != nil {
		return model.ConfigPreview{}, err
	}
	args := BuildArgs(p, configPath, defaultLogLevel)
	preview := content
	if !reveal {
		preview = MaskSecrets(content)
	}
	return model.ConfigPreview{
		Path:    configPath,
		Content: preview,
		Command: FormatCommand(npcPath, args),
	}, nil
}

// ReadConfig reads an existing generated config.
func ReadConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// TokenizeCommand splits a pasted command while respecting quotes.
func TokenizeCommand(raw string) []string {
	raw = strings.ReplaceAll(raw, "\ufeff", "")
	raw = strings.ReplaceAll(raw, "，", ",")
	raw = strings.ReplaceAll(raw, "：", ":")
	var out []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range raw {
		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
			continue
		}
		switch {
		case r == '"' || r == '\'':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

type parsedFlags struct {
	values  map[string]string
	order   []string
	ignored []string
}

var flagAliases = map[string]string{
	"server": "server", "s": "server",
	"vkey": "vkey", "k": "vkey",
	"type": "type", "t": "type",
	"proxy": "proxy", "local_ip": "local_ip",
	"dns_server": "dns_server", "ntp_server": "ntp_server",
	"ntp_interval": "ntp_interval", "disconnect_timeout": "disconnect_timeout",
	"auto_reconnect": "auto_reconnect", "proto_version": "proto_version",
	"skip_verify": "skip_verify", "disable_p2p": "disable_p2p",
	"keepalive": "keepalive", "log_level": "log_level",
	"config": "config",
}

var ignoredFlags = map[string]bool{
	"log": true, "debug": true, "log_color": true, "log_path": true,
	"log_max_size": true, "log_max_days": true, "log_max_files": true,
	"log_compress": true, "tls_enable": true, "pprof": true, "timezone": true,
	"disable_remote_update": true, "local_type": true, "local_port": true,
	"password": true, "target": true, "target_type": true, "local_proxy": true,
	"fallback_secret": true, "time": true, "gen2fa": true, "get2fa": true,
	"version": true, "local_ip_forward": true,
}

func parseFlags(tokens []string) parsedFlags {
	pf := parsedFlags{values: map[string]string{}}
	start := 0
	if len(tokens) > 0 {
		base := strings.ToLower(filepath.Base(tokens[0]))
		if base == "npc" || base == "npc.exe" || strings.HasSuffix(base, "npc") || strings.HasSuffix(base, "npc.exe") {
			start = 1
		}
	}
	for i := start; i < len(tokens); i++ {
		tok := tokens[i]
		lower := strings.ToLower(tok)
		switch lower {
		case "install", "start", "stop", "restart", "status", "register", "update", "update-force", "uninstall":
			continue
		}
		if !strings.HasPrefix(tok, "-") {
			continue
		}
		name := strings.TrimLeft(tok, "-")
		value := ""
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			value = name[idx+1:]
			name = name[:idx]
		}
		name = strings.ReplaceAll(strings.ToLower(name), "-", "_")
		if name == "" {
			continue
		}
		canonical, ok := flagAliases[name]
		if !ok {
			if ignoredFlags[name] {
				pf.ignored = append(pf.ignored, name)
			} else {
				pf.ignored = append(pf.ignored, name)
			}
			continue
		}
		if value == "" && i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
			value = tokens[i+1]
			i++
		}
		if canonical == "config" {
			pf.ignored = append(pf.ignored, "config")
			continue
		}
		pf.values[canonical] = value
		pf.order = append(pf.order, canonical)
	}
	return pf
}

// ParseCommand converts a pasted NPS launch command into profile candidates.
func ParseCommand(raw string) model.ImportPreview {
	result := model.ImportPreview{}
	tokens := TokenizeCommand(raw)
	if len(tokens) == 0 {
		result.Errors = append(result.Errors, "没有可解析的命令")
		return result
	}
	pf := parseFlags(tokens)
	server := strings.TrimSpace(pf.values["server"])
	vkey := strings.TrimSpace(pf.values["vkey"])
	if server == "" || vkey == "" {
		result.Errors = append(result.Errors, "命令中缺少 -server 或 -vkey")
		return result
	}
	servers := splitList(server)
	vkeys := splitList(vkey)
	types := splitList(pf.values["type"])
	if len(types) == 0 {
		types = []string{"tcp"}
	}
	max := len(servers)
	if len(vkeys) > max {
		max = len(vkeys)
	}
	if len(types) > max {
		max = len(types)
	}
	if max == 0 {
		result.Errors = append(result.Errors, "命令中没有有效的服务端")
		return result
	}
	repeat := func(list []string) []string {
		if len(list) == 0 {
			return list
		}
		out := append([]string(nil), list...)
		for len(out) < max {
			out = append(out, out[len(out)-1])
		}
		return out
	}
	servers = repeat(servers)
	vkeys = repeat(vkeys)
	types = repeat(types)
	for len(servers) < max {
		servers = append(servers, "")
	}
	for len(vkeys) < max {
		vkeys = append(vkeys, "")
	}
	for len(types) < max {
		types = append(types, "tcp")
	}
	for i := 0; i < max; i++ {
		p := model.NewProfile()
		p.Server = strings.TrimSpace(servers[i])
		p.VKey = strings.TrimSpace(vkeys[i])
		p.ConnType = strings.ToLower(strings.TrimSpace(types[i]))
		if p.ConnType == "" {
			p.ConnType = "tcp"
		}
		p.Name = serverLabel(p.Server, i, max)
		if v := pf.values["proxy"]; v != "" {
			p.ProxyURL = v
		}
		if v := pf.values["local_ip"]; v != "" {
			p.LocalIP = v
		}
		if v := pf.values["dns_server"]; v != "" {
			p.DNSServer = v
		}
		if v := pf.values["ntp_server"]; v != "" {
			p.NTPServer = v
		}
		if v := pf.values["ntp_interval"]; v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				p.NTPInterval = n
			}
		}
		if v := pf.values["disconnect_timeout"]; v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				p.DisconnectTimeout = n
			}
		}
		if v := pf.values["keepalive"]; v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				p.KeepAlive = n
			}
		}
		if v := pf.values["auto_reconnect"]; v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				p.AutoReconnection = b
			}
		}
		if v := pf.values["proto_version"]; v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				p.ProtoVersion = &n
			}
		}
		if v := pf.values["skip_verify"]; v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				p.SkipTLSVerify = b
			}
		}
		if v := pf.values["disable_p2p"]; v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				p.DisableP2P = b
			}
		}
		if v := pf.values["log_level"]; v != "" {
			p.LogLevel = strings.ToLower(v)
		}
		result.Candidates = append(result.Candidates, model.ImportCandidate{
			Profile:  p,
			Selected: true,
			Note:     fmt.Sprintf("服务端 %s", p.Server),
		})
	}
	if max > 1 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("检测到 %d 个服务端，将拆分为 %d 个独立档案", max, max))
	}
	seenIgnored := map[string]bool{}
	for _, name := range pf.ignored {
		if seenIgnored[name] {
			continue
		}
		seenIgnored[name] = true
		switch name {
		case "config":
			result.Warnings = append(result.Warnings, "已忽略 -config：GUI 会为每个档案生成独立配置")
		case "log", "debug", "log_color", "log_path", "log_max_size", "log_max_days", "log_max_files", "log_compress":
			result.Warnings = append(result.Warnings, "已忽略日志相关参数：GUI 统一管理日志")
		default:
			result.Warnings = append(result.Warnings, "已忽略暂不支持导入的参数："+name)
		}
	}
	return result
}

func serverLabel(server string, index, total int) string {
	host := server
	if i := strings.IndexByte(host, '@'); i >= 0 {
		host = host[:i]
	}
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "" {
		host = "NPS"
	}
	if total > 1 {
		return fmt.Sprintf("%s (%d)", host, index+1)
	}
	return host
}

// WriteCommand writes a display command to a file for export.
func WriteCommand(path, command string) error {
	return platform.AtomicWriteFile(path, []byte(command+"\n"), 0o600)
}
