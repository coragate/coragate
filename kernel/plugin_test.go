package kernel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type stubInspector struct {
	needle string
	id     string
}

func (s stubInspector) Name() string { return "stub" }

func (s stubInspector) InspectInput(_ context.Context, text string) InspectResult {
	if strings.Contains(strings.ToLower(text), strings.ToLower(s.needle)) {
		return InspectResult{Hit: true, RuleID: s.id}
	}
	return InspectResult{}
}

func (s stubInspector) InspectOutputWindow(ctx context.Context, window string) InspectResult {
	return s.InspectInput(ctx, window)
}

func TestAC4_enforce命中不打上游(t *testing.T) {
	var hit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      PolicyEnforce,
		Inspectors:      []Inspector{stubInspector{needle: "secret", id: "t-block"}},
	}))
	t.Cleanup(gw.Close)

	body := `{"model":"fake","messages":[{"role":"user","content":"leak SECRET now"}]}`
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)

	if hit.Load() {
		t.Fatal("enforce 命中后仍打了上游")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("状态码 = %d body=%s", resp.StatusCode, got)
	}
	if resp.Header.Get(headerHitRule) != "t-block" {
		t.Fatalf("缺少命中规则头: %v", resp.Header)
	}
}

func TestAC4_observe命中仍转发(t *testing.T) {
	var hit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: ok\n\n")
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      PolicyObserve,
		Inspectors:      []Inspector{stubInspector{needle: "secret", id: "t-block"}},
	}))
	t.Cleanup(gw.Close)

	body := `{"model":"fake","messages":[{"role":"user","content":"secret"}]}`
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !hit.Load() {
		t.Fatal("observe 应仍转发上游")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码 = %d", resp.StatusCode)
	}
	if resp.Header.Get(headerHitRule) != "t-block" {
		t.Fatal("observe 也应标记命中规则")
	}
}

func TestAC10_检测走插件接口(t *testing.T) {
	p := stubInspector{needle: "needle", id: "t-block"}
	r := InspectInput(context.Background(), []Inspector{p}, "find needle here")
	if !r.Hit || r.RuleID != "t-block" {
		t.Fatalf("插件未命中: %+v", r)
	}
}

func Test沙盒检测端点不走上游(t *testing.T) {
	var hit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		Inspectors:      []Inspector{stubInspector{needle: "secret", id: "t-block"}},
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/inspect", "application/json", strings.NewReader(`{"text":"contains secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out inspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if hit.Load() {
		t.Fatal("沙盒检测打了上游")
	}
	if !out.Hit || out.RuleID != "t-block" {
		t.Fatalf("沙盒结果 = %+v", out)
	}
}
