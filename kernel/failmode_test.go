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

func TestAC9_DefaultFailOpenForwardsAndTags(t *testing.T) {
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
		t.Fatal("default fail_open should still forward upstream")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	select {
	case env := <-ch:
		if env.EngineError == "" {
			t.Fatal("fail_open audit should include engine_error")
		}
		if env.Outcome != OutcomeFailOpen {
			t.Fatalf("outcome=%s", env.Outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive envelope")
	}
}

func TestAC9_FailClosedRejects(t *testing.T) {
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
		t.Fatal("fail_closed should not call upstream")
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body=%s", resp.StatusCode, got)
	}
	select {
	case env := <-ch:
		if env.EngineError == "" {
			t.Fatal("fail_closed audit should include engine_error")
		}
		if env.Outcome != OutcomeFailClosed {
			t.Fatalf("outcome=%s", env.Outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive envelope")
	}
}

func TestAC9_EmptyConfigIsFailOpen(t *testing.T) {
	if (Config{}).failClosed() {
		t.Fatal("unset FailMode must not be fail_closed")
	}
	if parseFailMode("") != FailOpen || parseFailMode("nope") != FailOpen {
		t.Fatal("unknown value should fall back to fail_open")
	}
	if parseFailMode(FailClosed) != FailClosed {
		t.Fatal("only explicit fail_closed closes")
	}
}

func TestLoadConfigDefaultsFailOpen(t *testing.T) {
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
