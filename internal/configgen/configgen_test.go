package configgen

import (
	"strings"
	"testing"

	"npc-gui/internal/model"
)

func baseProfile() model.Profile {
	p := model.NewProfile()
	p.ID = "p1"
	p.Name = "公司跳板"
	p.Server = "nps.example.com:8024"
	p.VKey = "secret-vkey"
	p.ConnType = "tls"
	return p
}

func TestGenerateAndMask(t *testing.T) {
	p := baseProfile()
	p.ProxyURL = "socks5://user:pass@127.0.0.1:1080"
	p.LocalIP = "192.168.1.20"
	p.DisconnectTimeout = 30
	p.ExtraCommon = "max_conn=100"
	p.ExtraSections = "[tcp]\nmode=tcp\nserver_port=10000\ntarget_addr=127.0.0.1:8080"
	content, err := Generate(p)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	for _, want := range []string{
		"server_addr=nps.example.com:8024",
		"conn_type=tls",
		"vkey=secret-vkey",
		"proxy_url=socks5://user:pass@127.0.0.1:1080",
		"local_ip=192.168.1.20",
		"disconnect_timeout=30",
		"max_conn=100",
		"[tcp]",
		"server_port=10000",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q:\n%s", want, content)
		}
	}
	masked := MaskSecrets(content)
	if strings.Contains(masked, "secret-vkey") {
		t.Errorf("masked config still contains vkey")
	}
	if strings.Contains(masked, "pass@127.0.0.1") {
		t.Errorf("masked config still contains proxy password")
	}
	if !strings.Contains(masked, "server_addr=nps.example.com:8024") {
		t.Errorf("masked config lost non-secret values")
	}
}

func TestValidate(t *testing.T) {
	p := baseProfile()
	if res := Validate(p); !res.OK {
		t.Fatalf("valid profile rejected: %+v", res)
	}

	bad := p
	bad.Server = "a:1,b:2"
	if res := Validate(bad); res.OK || res.FieldErrors["server"] == "" {
		t.Fatalf("multi-server should be rejected: %+v", res)
	}

	bad = p
	bad.ConnType = "http"
	if res := Validate(bad); res.OK || res.FieldErrors["connType"] == "" {
		t.Fatalf("bad conn type should be rejected: %+v", res)
	}

	bad = p
	bad.ProxyURL = "ftp://example.com:21"
	if res := Validate(bad); res.OK || res.FieldErrors["proxyUrl"] == "" {
		t.Fatalf("bad proxy scheme should be rejected: %+v", res)
	}
}

func TestExtraConfigValidation(t *testing.T) {
	p := baseProfile()
	p.ExtraCommon = "server_addr=bad"
	if res := Validate(p); res.OK || res.FieldErrors["extraCommon"] == "" {
		t.Fatalf("reserved common key should fail: %+v", res)
	}

	p = baseProfile()
	p.ExtraCommon = "max_conn=1\nmax_conn=2"
	if res := Validate(p); res.OK || res.FieldErrors["extraCommon"] == "" {
		t.Fatalf("duplicate common key should fail: %+v", res)
	}

	p = baseProfile()
	p.ExtraSections = "[tcp]\nmode=tcp\n[tcp]\nmode=udp"
	if res := Validate(p); res.OK || res.FieldErrors["extraSections"] == "" {
		t.Fatalf("duplicate section should fail: %+v", res)
	}

	p = baseProfile()
	p.ExtraSections = "[common]\nserver_addr=x"
	if res := Validate(p); res.OK || res.FieldErrors["extraSections"] == "" {
		t.Fatalf("reserved [common] should fail: %+v", res)
	}

	p = baseProfile()
	p.ExtraSections = "mode=tcp"
	if res := Validate(p); res.OK || res.FieldErrors["extraSections"] == "" {
		t.Fatalf("sectionless line should fail: %+v", res)
	}
}

func TestBuildArgs(t *testing.T) {
	p := baseProfile()
	p.LogLevel = ""
	version := 3
	p.ProtoVersion = &version
	p.SkipTLSVerify = true
	p.DisableP2P = true
	p.KeepAlive = 20
	args := BuildArgs(p, "/tmp/npc.conf", "warn")
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"-config=/tmp/npc.conf",
		"-log=stdout",
		"-debug=false",
		"-log_color=false",
		"-log_level=warn",
		"-disable_remote_update=true",
		"-proto_version=3",
		"-skip_verify=true",
		"-disable_p2p=true",
		"-keepalive=20",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("args missing %q: %s", want, joined)
		}
	}
}

func TestTokenizeCommand(t *testing.T) {
	tokens := TokenizeCommand(`npc.exe install -server="a:1,b:2" -vkey='k1,k2' -type="tls,tcp"`)
	want := []string{"npc.exe", "install", "-server=a:1,b:2", "-vkey=k1,k2", "-type=tls,tcp"}
	if len(tokens) != len(want) {
		t.Fatalf("tokens = %#v, want %#v", tokens, want)
	}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("token %d = %q, want %q", i, tokens[i], want[i])
		}
	}
}

func TestParseCommandMultiServer(t *testing.T) {
	raw := `npc.exe install -server="a.example.com:1,b.example.com:2" -vkey="k1,k2" -type="tls,tcp" -log=off -debug=false`
	preview := ParseCommand(raw)
	if len(preview.Errors) > 0 {
		t.Fatalf("ParseCommand errors: %v", preview.Errors)
	}
	if len(preview.Candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(preview.Candidates))
	}
	if preview.Candidates[0].Profile.Server != "a.example.com:1" || preview.Candidates[0].Profile.ConnType != "tls" {
		t.Fatalf("unexpected first candidate: %+v", preview.Candidates[0])
	}
	if preview.Candidates[1].Profile.Server != "b.example.com:2" || preview.Candidates[1].Profile.ConnType != "tcp" {
		t.Fatalf("unexpected second candidate: %+v", preview.Candidates[1])
	}
	if len(preview.Warnings) == 0 {
		t.Fatalf("expected warnings for split and ignored flags")
	}
}

func TestParseCommandMissing(t *testing.T) {
	preview := ParseCommand("npc -server=a:1")
	if len(preview.Errors) == 0 {
		t.Fatalf("expected missing vkey error")
	}
}

func TestParseCommandChinesePunctuation(t *testing.T) {
	preview := ParseCommand("npc -server=127.0.0.1：8024 -vkey=abc")
	if len(preview.Errors) > 0 || len(preview.Candidates) != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.Candidates[0].Profile.Server != "127.0.0.1:8024" {
		t.Fatalf("server = %q", preview.Candidates[0].Profile.Server)
	}
}
func TestParseCommandDotSlashNpc(t *testing.T) {
	raw := `./npc -server="198.51.100.10:30050" -vkey="test-vkey-value" -type="tcp"`
	preview := ParseCommand(raw)
	if len(preview.Errors) > 0 {
		t.Fatalf("errors: %v", preview.Errors)
	}
	if len(preview.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(preview.Candidates))
	}
	p := preview.Candidates[0].Profile
	if p.Server != "198.51.100.10:30050" {
		t.Errorf("server = %q", p.Server)
	}
	if p.VKey != "test-vkey-value" {
		t.Errorf("vkey = %q", p.VKey)
	}
	if p.ConnType != "tcp" {
		t.Errorf("connType = %q", p.ConnType)
	}
	if p.Name != "198.51.100.10" {
		t.Errorf("name = %q", p.Name)
	}
}
