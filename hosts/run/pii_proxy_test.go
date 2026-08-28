package run

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coragate/coragate/kernel"
)

func TestAC2_PIIEnforceBlockSkipsUpstream(t *testing.T) {
	ins, err := compileSnapshot(kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules: []kernel.SnapshotRule{{
			ID:     "pii-email",
			Plugin: "pii",
			Type:   "email",
			Action: "block",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	var upHit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upHit.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(kernel.Handler(kernel.Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      kernel.PolicyEnforce,
		Inspectors:      ins,
	}))
	t.Cleanup(gw.Close)

	body := `{"model":"fake","messages":[{"role":"user","content":"mail alice@example.com please"}]}`
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)

	if upHit.Load() {
		t.Fatal("enforce+block PII still reached upstream")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", resp.StatusCode, got)
	}
}

func TestAC6_PUTSwitchesPIIAction(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	snap := kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules: []kernel.SnapshotRule{{
			ID:     "pii-email",
			Plugin: "pii",
			Type:   "email",
		}},
	}
	if err := kernel.WriteSnapshotFile(path, snap); err != nil {
		t.Fatal(err)
	}
	rs := kernel.NewRuleset(compileSnapshot)
	if err := rs.LoadFile(path); err != nil {
		t.Fatal(err)
	}

	var upHits atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upHits.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(kernel.Handler(kernel.Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      kernel.PolicyEnforce,
		Rules:           rs,
		RulesPath:       path,
	}))
	t.Cleanup(gw.Close)

	raw := `{"model":"fake","messages":[{"role":"user","content":"mail alice@example.com please"}]}`
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("default redact must forward, status=%d", resp.StatusCode)
	}
	if upHits.Load() != 1 {
		t.Fatalf("upstream hits after redact = %d", upHits.Load())
	}

	block := kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules: []kernel.SnapshotRule{{
			ID:     "pii-email",
			Plugin: "pii",
			Type:   "email",
			Action: "block",
		}},
	}
	body, err := json.Marshal(block)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPut, gw.URL+"/v1/rules", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	putBody, _ := io.ReadAll(putResp.Body)
	_ = putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /v1/rules status=%d body=%s", putResp.StatusCode, putBody)
	}

	resp2, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Fatalf("after PUT block, status=%d", resp2.StatusCode)
	}
	if upHits.Load() != 1 {
		t.Fatalf("block request still hit upstream, hits=%d", upHits.Load())
	}
}
