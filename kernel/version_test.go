package kernel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAC13_健康检查含版本与schema(t *testing.T) {
	gw := httptest.NewServer(Handler(Config{}))
	t.Cleanup(gw.Close)
	resp, err := http.Get(gw.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码 = %d", resp.StatusCode)
	}
	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Version == "" {
		t.Fatal("缺少 version")
	}
	if info.ConfigSchemaVersion != ConfigSchemaVersion {
		t.Fatalf("config_schema_version=%d", info.ConfigSchemaVersion)
	}
	if info.RulesSchemaVersion != RulesSchemaVersion {
		t.Fatalf("rules_schema_version=%d", info.RulesSchemaVersion)
	}
	if info.EnvelopeSchemaVersion != EnvelopeSchemaVersion {
		t.Fatalf("envelope_schema_version=%d", info.EnvelopeSchemaVersion)
	}
}

func TestAC13_CLI可查版本(t *testing.T) {
	if !WantVersion([]string{"--version"}) || !WantVersion([]string{"-version"}) {
		t.Fatal("--version 应被识别")
	}
	if WantVersion(nil) || WantVersion([]string{"serve"}) {
		t.Fatal("无版本参数不应当成查版本")
	}
	var buf bytes.Buffer
	WriteVersion(&buf)
	got := buf.String()
	if !strings.Contains(got, Version) {
		t.Fatalf("CLI 输出缺少版本: %s", got)
	}
	if !strings.Contains(got, "config_schema_version=") || !strings.Contains(got, "rules_schema_version=") {
		t.Fatalf("CLI 输出缺少 schema: %s", got)
	}
}

func TestAC13_配置快照必须带schema_version(t *testing.T) {
	if _, err := ParseConfigSnapshot([]byte(`{"listen":"127.0.0.1:8080"}`)); err == nil {
		t.Fatal("缺少 schema_version 的配置应拒绝")
	}
	if _, err := ParseConfigSnapshot([]byte(`{"schema_version":2,"listen":"x"}`)); err == nil {
		t.Fatal("未知 schema_version 应拒绝")
	}
	snap, err := ParseConfigSnapshot([]byte(`{"schema_version":1,"listen":"127.0.0.1:8080","policy_mode":"enforce","fail_mode":"fail_open"}`))
	if err != nil {
		t.Fatal(err)
	}
	if snap.SchemaVersion != ConfigSchemaVersion {
		t.Fatalf("schema_version=%d", snap.SchemaVersion)
	}
}

func TestAC13_LoadConfig可导出带版本快照(t *testing.T) {
	t.Setenv("CORAGATE_LISTEN", "")
	snap := LoadConfig(HostClient).Snapshot()
	if snap.SchemaVersion != ConfigSchemaVersion {
		t.Fatalf("schema_version=%d", snap.SchemaVersion)
	}
	if snap.Listen != DefaultListenClient {
		t.Fatalf("listen=%s", snap.Listen)
	}
	b, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"schema_version"`) {
		t.Fatalf("序列化后看不到 schema_version: %s", b)
	}
}

func TestAC13_Changelog文件存在(t *testing.T) {
	if _, err := os.Stat("../CHANGELOG.md"); err != nil {
		t.Fatalf("Changelog 占位文件应存在: %v", err)
	}
}
