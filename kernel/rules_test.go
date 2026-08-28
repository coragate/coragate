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
			return nil, fmt.Errorf("compile failed")
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

func TestParseSnapshot_RequiresSchemaVersion(t *testing.T) {
	if _, err := ParseSnapshot([]byte(`{"rules":[]}`)); err == nil {
		t.Fatal("missing schema_version should be rejected")
	}
	if _, err := ParseSnapshot([]byte(`{"schema_version":2,"rules":[]}`)); err == nil {
		t.Fatal("unknown schema_version should be rejected")
	}
	snap, err := ParseSnapshot([]byte(`{"schema_version":1,"rules":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if snap.SchemaVersion != 1 || snap.Rules == nil {
		t.Fatalf("snapshot = %+v", snap)
	}
	snap, err = ParseSnapshot([]byte(`{"schema_version":1,"rules":[{"id":"pii-email","plugin":"pii","type":"email"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if snap.Rules[0].Type != "email" || snap.Rules[0].Action != "" {
		t.Fatalf("optional type/action: %+v", snap.Rules[0])
	}
}

func TestAC8_ReloadAppliesNewRulesToNewRequests(t *testing.T) {
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
		t.Fatalf("reload status = %d body=%s", resp.StatusCode, body)
	}

	assertInspect("contains alpha", false, "")
	assertInspect("contains beta", true, "r-beta")
}

func TestAC8_FailedReloadKeepsOldRules(t *testing.T) {
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
		t.Fatalf("invalid snapshot should be 400, got %d", resp.StatusCode)
	}

	got := InspectInput(context.Background(), rs.Inspectors(), "please keep-me")
	if !got.Hit || got.RuleID != "r-keep" {
		t.Fatalf("failed reload must not drop old rules: %+v", got)
	}
}

func TestAC8_PollLoadsNewRulesFile(t *testing.T) {
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
				t.Fatal("still hitting old rule after poll")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("poll did not load new rules within 1s")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
