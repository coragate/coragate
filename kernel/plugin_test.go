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
	"time"
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

func TestAC4_EnforceHitSkipsUpstream(t *testing.T) {
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
		t.Fatal("enforce hit still reached upstream")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d body=%s", resp.StatusCode, got)
	}
	if resp.Header.Get(headerHitRule) != "t-block" {
		t.Fatalf("missing hit-rule header: %v", resp.Header)
	}
}

func TestAC4_ObserveHitStillForwardsAndAudits(t *testing.T) {
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
		PolicyMode:      PolicyObserve,
		Auditor:         a,
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
		t.Fatal("observe should still forward upstream")
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if resp.Header.Get(headerHitRule) != "t-block" {
		t.Fatal("observe should still mark the hit rule")
	}
	select {
	case env := <-ch:
		if env.PolicyMode != PolicyObserve {
			t.Fatalf("policy_mode=%s", env.PolicyMode)
		}
		if env.RuleID != "t-block" {
			t.Fatalf("rule_id=%s", env.RuleID)
		}
		if env.Outcome != OutcomeForwarded {
			t.Fatalf("observe must not set outcome to blocked: %s", env.Outcome)
		}
	case <-time.After(time.Second):
		t.Fatal("observe did not write audit")
	}
}

func TestAC10_DetectionGoesThroughPluginInterface(t *testing.T) {
	p := stubInspector{needle: "needle", id: "t-block"}
	r := InspectInput(context.Background(), []Inspector{p}, "find needle here")
	if !r.Hit || r.RuleID != "t-block" {
		t.Fatalf("plugin did not hit: %+v", r)
	}
}

func TestSandboxInspectDoesNotCallUpstream(t *testing.T) {
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
		t.Fatal("sandbox inspect called upstream")
	}
	if !out.Hit || out.RuleID != "t-block" {
		t.Fatalf("sandbox result = %+v", out)
	}
	if len(out.Matches) != 1 || out.Matches[0].RuleID != "t-block" {
		t.Fatalf("sandbox matches = %+v", out.Matches)
	}
}
