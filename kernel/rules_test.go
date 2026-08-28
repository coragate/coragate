package kernel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func stubCompile(snap RuleSnapshot) ([]Inspector, error) {
	out := make([]Inspector, 0, len(snap.Rules))
	for _, r := range snap.Rules {
		if r.Pattern == "COMPILE_FAIL" {
			return nil, fmt.Errorf("编译失败")
		}
		out = append(out, stubInspector{needle: r.Pattern, id: r.ID})
	}
	return out, nil
}

func writeRules(t *testing.T, path string, needle, id string) {
	t.Helper()
	snap := RuleSnapshot{
		SchemaVersion: RulesSchemaVersion,
		Rules:         []SnapshotRule{{ID: id, Plugin: "keyword", Pattern: needle}},
	}
	if err := WriteSnapshotFile(path, snap); err != nil {
		t.Fatal(err)
	}
}

func TestParseSnapshot_必须带schema版本(t *testing.T) {
	if _, err := ParseSnapshot([]byte(`{"rules":[]}`)); err == nil {
		t.Fatal("缺少 schema_version 应拒绝")
	}
	if _, err := ParseSnapshot([]byte(`{"schema_version":2,"rules":[]}`)); err == nil {
		t.Fatal("未知 schema_version 应拒绝")
	}
	snap, err := ParseSnapshot([]byte(`{"schema_version":1,"rules":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if snap.SchemaVersion != 1 || snap.Rules == nil {
		t.Fatalf("快照 = %+v", snap)
	}
}

func TestAC8_reload后新请求用新规则(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	writeRules(t, path, "alpha", "r-alpha")

	rs := NewRuleset(stubCompile)
	if err := rs.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	gw := httptest.NewServer(Handler(Config{Rules: rs, RulesPath: path}))
	t.Cleanup(gw.Close)

	assertInspect := func(text string, wantHit bool, wantID string) {
		t.Helper()
		resp, err := http.Post(gw.URL+"/v1/inspect", "application/json", strings.NewReader(`{"text":"`+text+`"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out inspectResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if out.Hit != wantHit || out.RuleID != wantID {
			t.Fatalf("text=%s hit=%v id=%s", text, out.Hit, out.RuleID)
		}
	}

	assertInspect("contains alpha", true, "r-alpha")

	writeRules(t, path, "beta", "r-beta")
	resp, err := http.Post(gw.URL+"/v1/reload", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("reload 状态码 = %d body=%s", resp.StatusCode, body)
	}

	assertInspect("contains alpha", false, "")
	assertInspect("contains beta", true, "r-beta")
}

func TestAC8_reload失败保留旧规则(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	writeRules(t, path, "keep-me", "r-keep")

	rs := NewRuleset(stubCompile)
	if err := rs.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	gw := httptest.NewServer(Handler(Config{Rules: rs, RulesPath: path}))
	t.Cleanup(gw.Close)

	if err := os.WriteFile(path, []byte(`{"schema_version":1,"rules":[{"id":"bad","pattern":"COMPILE_FAIL"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(gw.URL+"/v1/reload", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("非法快照应 400，得到 %d", resp.StatusCode)
	}

	got := InspectInput(context.Background(), rs.Inspectors(), "please keep-me")
	if !got.Hit || got.RuleID != "r-keep" {
		t.Fatalf("失败 reload 不该丢掉旧规则: %+v", got)
	}
}

func TestAC8_短轮询加载新规则文件(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	writeRules(t, path, "alpha", "r-alpha")

	rs := NewRuleset(stubCompile)
	if err := rs.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go WatchRulesFile(ctx, rs, path, 20*time.Millisecond, nil)

	writeRules(t, path, "beta", "r-beta")
	deadline := time.Now().Add(time.Second)
	for {
		got := InspectInput(context.Background(), rs.Inspectors(), "beta now")
		if got.Hit && got.RuleID == "r-beta" {
			miss := InspectInput(context.Background(), rs.Inspectors(), "alpha only")
			if miss.Hit {
				t.Fatal("轮询后仍命中旧规则")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("短轮询未在 1s 内加载新规则")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
