package kernel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func Test控制面GET与PUT规则后新检测生效(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	writeRules(t, path, "alpha", "r-alpha")
	rs := NewRuleset(stubCompile)
	if err := rs.LoadFile(path); err != nil {
		t.Fatal(err)
	}
	gw := httptest.NewServer(Handler(Config{Rules: rs, RulesPath: path}))
	t.Cleanup(gw.Close)

	resp, err := http.Get(gw.URL + "/v1/rules")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var got RuleSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.SchemaVersion != RulesSchemaVersion || len(got.Rules) != 1 || got.Rules[0].ID != "r-alpha" {
		t.Fatalf("GET 规则 = %+v", got)
	}

	body := `{"schema_version":1,"rules":[{"id":"r-beta","plugin":"keyword","pattern":"beta"}]}`
	req, err := http.NewRequest(http.MethodPut, gw.URL+"/v1/rules", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(putResp.Body)
		t.Fatalf("PUT 状态码 = %d body=%s", putResp.StatusCode, b)
	}

	ins, err := http.Post(gw.URL+"/v1/inspect", "application/json", strings.NewReader(`{"text":"contains beta"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer ins.Body.Close()
	var out inspectResponse
	if err := json.NewDecoder(ins.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !out.Hit || out.RuleID != "r-beta" {
		t.Fatalf("PUT 后沙盒未用新规则: %+v", out)
	}
}

func Test控制面只读审计列表(t *testing.T) {
	mem := &memStore{}
	a := NewAuditor(mem, 8)
	t.Cleanup(a.Close)
	// 直接写入已 flush 的 envelope，看板读 Store 而不是 SQLite。
	if err := mem.Append(context.Background(), Envelope{
		SchemaVersion: EnvelopeSchemaVersion,
		Time:          time.Now().UTC().Format(time.RFC3339Nano),
		RuleID:        "r-hit",
		PromptHash:    "abc",
		PolicyMode:    PolicyEnforce,
		Outcome:       OutcomeBlocked,
	}); err != nil {
		t.Fatal(err)
	}

	gw := httptest.NewServer(Handler(Config{Auditor: a}))
	t.Cleanup(gw.Close)
	resp, err := http.Get(gw.URL + "/v1/audit?limit=10")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var payload struct {
		Items []Envelope `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Items) != 1 || payload.Items[0].RuleID != "r-hit" {
		t.Fatalf("审计列表 = %+v", payload.Items)
	}
}

type memStore struct {
	items []Envelope
}

func (m *memStore) Append(_ context.Context, env Envelope) error {
	m.items = append(m.items, env)
	return nil
}

func (m *memStore) List(_ context.Context, limit int) ([]Envelope, error) {
	out := m.items
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}
