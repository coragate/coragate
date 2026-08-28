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

func TestAC13_HealthIncludesVersionAndSchema(t *testing.T) {
	gw := httptest.NewServer(Handler(Config{}))
	t.Cleanup(gw.Close)
	resp, err := http.Get(gw.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatal(err)
	}
	if info.Version == "" {
		t.Fatal("missing version")
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

func TestAC13_CLIReportsVersion(t *testing.T) {
	if !WantVersion([]string{"--version"}) || !WantVersion([]string{"-version"}) {
		t.Fatal("--version should be recognized")
	}
	if WantVersion(nil) || WantVersion([]string{"serve"}) {
		t.Fatal("missing version flag should not be treated as --version")
	}
	var buf bytes.Buffer
	WriteVersion(&buf)
	got := buf.String()
	if !strings.Contains(got, Version) {
		t.Fatalf("CLI output missing version: %s", got)
	}
	if !strings.Contains(got, "config_schema_version=") || !strings.Contains(got, "rules_schema_version=") {
		t.Fatalf("CLI output missing schema: %s", got)
	}
}

func TestAC13_ConfigSnapshotRequiresSchemaVersion(t *testing.T) {
	if _, err := ParseConfigSnapshot([]byte(`{"listen":"127.0.0.1:8080"}`)); err == nil {
		t.Fatal("config without schema_version should be rejected")
	}
	if _, err := ParseConfigSnapshot([]byte(`{"schema_version":2,"listen":"x"}`)); err == nil {
		t.Fatal("unknown schema_version should be rejected")
	}
	snap, err := ParseConfigSnapshot([]byte(`{"schema_version":1,"listen":"127.0.0.1:8080","policy_mode":"enforce","fail_mode":"fail_open"}`))
	if err != nil {
		t.Fatal(err)
	}
	if snap.SchemaVersion != ConfigSchemaVersion {
		t.Fatalf("schema_version=%d", snap.SchemaVersion)
	}
}

func TestAC13_LoadConfigExportsVersionedSnapshot(t *testing.T) {
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
		t.Fatalf("serialized snapshot missing schema_version: %s", b)
	}
}

func TestAC13_ChangelogFileExists(t *testing.T) {
	if _, err := os.Stat("../CHANGELOG.md"); err != nil {
		t.Fatalf("Changelog file should exist: %v", err)
	}
}
