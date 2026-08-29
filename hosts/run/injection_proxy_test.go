package run

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coragate/coragate/kernel"
	"github.com/coragate/coragate/plugins/detect/injection"
	"github.com/coragate/coragate/plugins/detect/pii"
)

func injectionInspectors(t *testing.T, action string) []kernel.Inspector {
	t.Helper()
	ins, err := compileSnapshot(kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules: []kernel.SnapshotRule{{
			ID:     "injection-prompt",
			Plugin: "injection",
			Type:   injection.EntityPromptInjection,
			Action: action,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return ins
}

func chatBody(text string) string {
	b, _ := json.Marshal(map[string]any{
		"model": "fake",
		"messages": []map[string]string{
			{"role": "user", "content": text},
		},
	})
	return string(b)
}

func TestAC11_Inspect夹具命中与对照句(t *testing.T) {
	ins := injectionInspectors(t, "")
	gw := httptest.NewServer(kernel.Handler(kernel.Config{Inspectors: ins}))
	t.Cleanup(gw.Close)

	post := func(text string) map[string]any {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{"text": text})
		resp, err := http.Post(gw.URL+"/v1/inspect", "application/json", bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	hit := post(injection.FixtureF1Hit)
	if hit["hit"] != true {
		t.Fatalf("F1 必须命中：%v", hit)
	}
	if hit["rule_id"] != "injection-prompt" {
		t.Fatalf("rule_id=%v", hit["rule_id"])
	}
	if hit["entity_type"] != injection.EntityPromptInjection {
		t.Fatalf("entity_type=%v", hit["entity_type"])
	}

	miss := post(injection.FixtureF1Miss)
	if miss["hit"] == true {
		t.Fatalf("F1 对照句不得命中：%v", miss)
	}
}

func TestAC2_EnforceBlock不打上游(t *testing.T) {
	ins := injectionInspectors(t, "")
	for _, fixture := range []string{
		injection.FixtureF1Hit,
		injection.FixtureF2Hit,
		injection.FixtureF3Hit,
		injection.FixtureF4Hit,
	} {
		var upHit atomic.Bool
		up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upHit.Store(true)
			w.WriteHeader(http.StatusOK)
		}))
		gw := httptest.NewServer(kernel.Handler(kernel.Config{
			UpstreamBaseURL: up.URL,
			PolicyMode:      kernel.PolicyEnforce,
			Inspectors:      ins,
		}))
		resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(chatBody(fixture)))
		if err != nil {
			up.Close()
			gw.Close()
			t.Fatal(err)
		}
		got, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		up.Close()
		gw.Close()
		if upHit.Load() {
			t.Fatalf("enforce+block 仍打到上游，夹具=%q", fixture)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status=%d body=%s 夹具=%q", resp.StatusCode, got, fixture)
		}
	}
}

func TestAC1_注入盖过PII脱敏_不打上游(t *testing.T) {
	inj, err := injection.New(injection.Rule{ID: "injection-prompt"})
	if err != nil {
		t.Fatal(err)
	}
	piiP, err := pii.New(pii.Rule{ID: "pii-email", Type: pii.TypeEmail})
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
		Inspectors:      []kernel.Inspector{piiP, inj},
	}))
	t.Cleanup(gw.Close)

	text := injection.FixtureF1Hit + " mail alice@example.com"
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(chatBody(text)))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if upHit.Load() {
		t.Fatal("注入 block 应盖过 PII redact，不得打上游")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestAC3_Observe不拦不改写(t *testing.T) {
	ins := injectionInspectors(t, "")
	var sawBody atomic.Value
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		sawBody.Store(string(b))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
	}))
	t.Cleanup(up.Close)
	gw := httptest.NewServer(kernel.Handler(kernel.Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      kernel.PolicyObserve,
		Inspectors:      ins,
	}))
	t.Cleanup(gw.Close)

	raw := chatBody(injection.FixtureF1Hit)
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("observe 不得拒绝，status=%d", resp.StatusCode)
	}
	got, _ := sawBody.Load().(string)
	if got != raw {
		t.Fatalf("observe 不得改写请求体：\n got=%s\nwant=%s", got, raw)
	}
}

func sseDelta(content string) string {
	b, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{
			{"delta": map[string]string{"content": content}},
		},
	})
	return "data: " + string(b) + "\n"
}

func TestAC4_输出跨SSE拼夹具后停后续chunk(t *testing.T) {
	ins := injectionInspectors(t, "")
	mid := len(injection.FixtureF1Hit) / 2
	prefix := injection.FixtureF1Hit[:mid]
	rest := injection.FixtureF1Hit[mid:]

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		_, _ = io.WriteString(w, sseDelta(prefix))
		fl.Flush()
		_, _ = io.WriteString(w, sseDelta(rest))
		fl.Flush()
		_, _ = io.WriteString(w, sseDelta("tail-must-not-forward"))
		fl.Flush()
	}))
	t.Cleanup(up.Close)
	gw := httptest.NewServer(kernel.Handler(kernel.Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      kernel.PolicyEnforce,
		Inspectors:      ins,
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	body := string(got)
	if strings.Contains(body, "tail-must-not-forward") {
		t.Fatalf("命中后仍转发后续 chunk：%s", body)
	}
}

func TestAC4_中文夹具跨SSE同样停后续(t *testing.T) {
	ins := injectionInspectors(t, "")
	runes := []rune(injection.FixtureF3Hit)
	mid := len(runes) / 2
	prefix := string(runes[:mid])
	rest := string(runes[mid:])

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		_, _ = io.WriteString(w, sseDelta(prefix))
		fl.Flush()
		_, _ = io.WriteString(w, sseDelta(rest))
		fl.Flush()
		_, _ = io.WriteString(w, sseDelta("tail-zh"))
		fl.Flush()
	}))
	t.Cleanup(up.Close)
	gw := httptest.NewServer(kernel.Handler(kernel.Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      kernel.PolicyEnforce,
		Inspectors:      ins,
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(got), "tail-zh") {
		t.Fatalf("F3 命中后仍转发后续：%s", got)
	}
}

func TestAC4_Observe输出不截流(t *testing.T) {
	ins := injectionInspectors(t, "")
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, sseDelta(injection.FixtureF1Hit))
		_, _ = io.WriteString(w, sseDelta("tail-ok"))
	}))
	t.Cleanup(up.Close)
	gw := httptest.NewServer(kernel.Handler(kernel.Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      kernel.PolicyObserve,
		Inspectors:      ins,
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(got), "tail-ok") {
		t.Fatalf("observe 不得截流：%s", got)
	}
}

func TestAC7_审计含类型动作与原始哈希无明文(t *testing.T) {
	ins := injectionInspectors(t, "")
	ch := make(chan kernel.Envelope, 1)
	a := kernel.NewAuditor(&captureAudit{ch: ch}, 8)
	t.Cleanup(a.Close)

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)
	gw := httptest.NewServer(kernel.Handler(kernel.Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      kernel.PolicyEnforce,
		Auditor:         a,
		Inspectors:      ins,
	}))
	t.Cleanup(gw.Close)

	raw := []byte(chatBody(injection.FixtureF1Hit))
	sum := sha256.Sum256(raw)
	wantHex := hex.EncodeToString(sum[:])

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	select {
	case env := <-ch:
		if env.PromptHash != wantHex {
			t.Fatalf("prompt_hash=%s want=%s", env.PromptHash, wantHex)
		}
		if env.EntityType != injection.EntityPromptInjection {
			t.Fatalf("entity_type=%s", env.EntityType)
		}
		if env.RuleAction != kernel.ActionBlock {
			t.Fatalf("rule_action=%s", env.RuleAction)
		}
		if env.SchemaVersion != kernel.EnvelopeSchemaVersion {
			t.Fatalf("schema_version=%d", env.SchemaVersion)
		}
		b, _ := json.Marshal(env)
		s := string(b)
		if strings.Contains(s, injection.FixtureF1Hit) || strings.Contains(s, string(raw)) {
			t.Fatalf("审计不得含命中明文或 Prompt：%s", s)
		}
	case <-time.After(time.Second):
		t.Fatal("缺少审计 envelope")
	}
}

func TestAC6_PUT写入injection后新请求生效(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.json")
	snap := kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules: []kernel.SnapshotRule{{
			ID:      "demo-keyword",
			Plugin:  "keyword",
			Pattern: `(?i)coragate-block-me`,
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

	raw := chatBody(injection.FixtureF1Hit)
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("写入 injection 前 F1 不应拦截，status=%d", resp.StatusCode)
	}

	next := kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules: []kernel.SnapshotRule{{
			ID:     "injection-prompt",
			Plugin: "injection",
			Type:   injection.EntityPromptInjection,
		}},
	}
	body, err := json.Marshal(next)
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
		t.Fatalf("PUT 后应拦截，status=%d", resp2.StatusCode)
	}
	if upHits.Load() != 1 {
		t.Fatalf("block 请求仍打上游，hits=%d", upHits.Load())
	}
}

type captureAudit struct{ ch chan kernel.Envelope }

func (c *captureAudit) Append(_ context.Context, env kernel.Envelope) error {
	c.ch <- env
	return nil
}
func (c *captureAudit) List(context.Context, int) ([]kernel.Envelope, error) { return nil, nil }
