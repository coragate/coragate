package kernel

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type outputSpy struct {
	n       atomic.Int32
	mu      sync.Mutex
	windows []string
}

func (o *outputSpy) Name() string { return "output-spy" }

func (o *outputSpy) InspectInput(context.Context, string) InspectResult {
	return InspectResult{}
}

func (o *outputSpy) InspectOutputWindow(_ context.Context, window string) InspectResult {
	o.n.Add(1)
	o.mu.Lock()
	o.windows = append(o.windows, window)
	o.mu.Unlock()
	return InspectResult{}
}

func (o *outputSpy) last() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.windows) == 0 {
		return ""
	}
	return o.windows[len(o.windows)-1]
}

func sseDelta(content string) string {
	return `data: {"choices":[{"delta":{"content":"` + content + `"}}]}` + "\n"
}

func TestAC3_ScansDuringForwardWithoutWaitingForEnd(t *testing.T) {
	spy := &outputSpy{}
	proceed := make(chan struct{})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		_, _ = io.WriteString(w, sseDelta("hel"))
		fl.Flush()
		<-proceed
		_, _ = io.WriteString(w, sseDelta("lo"))
		_, _ = io.WriteString(w, "data: [DONE]\n")
		fl.Flush()
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		Inspectors:      []Inspector{spy},
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	rd := bufio.NewReader(resp.Body)
	first, err := rd.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first, "hel") {
		t.Fatalf("first chunk = %q", first)
	}
	if spy.n.Load() < 1 {
		t.Fatal("no output scan before the second upstream chunk; waiting for the full stream")
	}
	close(proceed)
	rest, _ := io.ReadAll(rd)
	if !strings.Contains(string(rest), "lo") {
		t.Fatalf("missing later chunk: %s", rest)
	}
	if spy.last() != "hello" {
		t.Fatalf("window did not join across chunks: %q", spy.last())
	}
}

func TestAC3_KeywordHitsAcrossSSEEvents(t *testing.T) {
	spy := &outputSpy{}
	combo := []Inspector{spy, outputOnly{needle: "secret", id: "out"}}

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		_, _ = io.WriteString(w, sseDelta("sec"))
		fl.Flush()
		_, _ = io.WriteString(w, sseDelta("ret"))
		fl.Flush()
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      PolicyObserve,
		Inspectors:      combo,
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"x","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.ReadAll(resp.Body)
	if spy.last() != "secret" {
		t.Fatalf("cross-event window = %q", spy.last())
	}
}

type outputOnly struct {
	needle, id string
}

func (o outputOnly) Name() string { return "output-only" }

func (o outputOnly) InspectInput(context.Context, string) InspectResult { return InspectResult{} }

func (o outputOnly) InspectOutputWindow(_ context.Context, window string) InspectResult {
	if strings.Contains(window, o.needle) {
		return InspectResult{Hit: true, RuleID: o.id}
	}
	return InspectResult{}
}
