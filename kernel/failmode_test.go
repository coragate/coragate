package kernel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type panicInspector struct{}

func (panicInspector) Name() string { return "panic" }

func (panicInspector) InspectInput(context.Context, string) InspectResult {
	panic("engine down")
}

func (panicInspector) InspectOutputWindow(context.Context, string) InspectResult {
	panic("engine down")
}

func TestAC9_默认fail_open放行并打标(t *testing.T) {
	var hit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: ok\n\n")
	}))
	t.Cleanup(up.Close)

	ch := make(chan Envelope, 1)
	a := NewAuditor(&captureStore{ch: ch}, 8)
	t.Cleanup(a.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		Auditor:         a,
		Inspectors:      []Inspector{panicInspector{}},
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if !hit.Load() {
		t.Fatal("默认 fail_open 应仍转发上游")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("状态码 = %d", resp.StatusCode)
	}
	select {
	case env := <-ch:
		if env.EngineError == "" {
			t.Fatal("fail_open 审计应带 engine_error")
		}
		if env.Outcome != OutcomeFailOpen {
			t.Fatalf("outcome=%s", env.Outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到 envelope")
	}
}

func TestAC9_fail_closed才拒绝(t *testing.T) {
	var hit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
	}))
	t.Cleanup(up.Close)

	ch := make(chan Envelope, 1)
	a := NewAuditor(&captureStore{ch: ch}, 8)
	t.Cleanup(a.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		FailMode:        FailClosed,
		Auditor:         a,
		Inspectors:      []Inspector{panicInspector{}},
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)
	if hit.Load() {
		t.Fatal("fail_closed 不应打上游")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("状态码 = %d body=%s", resp.StatusCode, got)
	}
	select {
	case env := <-ch:
		if env.EngineError == "" {
			t.Fatal("fail_closed 审计应带 engine_error")
		}
		if env.Outcome != OutcomeFailClosed {
			t.Fatalf("outcome=%s", env.Outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("未收到 envelope")
	}
}

func TestAC9_空配置即fail_open(t *testing.T) {
	if (Config{}).failClosed() {
		t.Fatal("未配置 FailMode 不得视为 fail_closed")
	}
	if parseFailMode("") != FailOpen || parseFailMode("nope") != FailOpen {
		t.Fatal("未知值应回退 fail_open")
	}
	if parseFailMode(FailClosed) != FailClosed {
		t.Fatal("显式 fail_closed 才关闭")
	}
}

func TestLoadConfig默认fail_open(t *testing.T) {
	t.Setenv("CORAGATE_FAIL_MODE", "")
	cfg := LoadConfig(HostClient)
	if cfg.FailMode != FailOpen {
		t.Fatalf("FailMode=%s", cfg.FailMode)
	}
	t.Setenv("CORAGATE_FAIL_MODE", FailClosed)
	cfg = LoadConfig(HostClient)
	if cfg.FailMode != FailClosed {
		t.Fatalf("FailMode=%s", cfg.FailMode)
	}
}
