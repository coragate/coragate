package kernel

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAC1_ExposesChatCompletionsWithSSE(t *testing.T) {
	var hit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("upstream path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fl, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("upstream cannot Flush")
		}
		_, _ = io.WriteString(w, "data: {\"id\":\"chunk-1\"}\n\n")
		fl.Flush()
		time.Sleep(20 * time.Millisecond)
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{UpstreamBaseURL: up.URL}))
	t.Cleanup(gw.Close)

	req, err := http.NewRequest(http.MethodPost, gw.URL+"/v1/chat/completions", strings.NewReader(`{"model":"fake","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %s", ct)
	}

	rd := bufio.NewReader(resp.Body)
	first, err := rd.ReadString('\n')
	if err != nil {
		t.Fatalf("did not read an SSE line before end: %v", err)
	}
	if !strings.Contains(first, "chunk-1") {
		t.Fatalf("first chunk = %q", first)
	}
}

func TestAC2_RequestHitsUpstreamResponseFromGateway(t *testing.T) {
	var hit atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		if r.ContentLength <= 0 {
			t.Errorf("upstream Content-Length=%d; a non-Go server would see an empty body", r.ContentLength)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"model":"fake"`) {
			t.Errorf("upstream did not receive client body: %s", body)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"object\":\"chat.completion.chunk\"}\n\n")
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{UpstreamBaseURL: up.URL}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := io.ReadAll(resp.Body)

	if !hit.Load() {
		t.Fatal("request did not reach fake upstream")
	}
	if resp.Header.Get(headerGatewayMark) != "1" {
		t.Fatal("response missing gateway mark; may look like a direct upstream")
	}
	if strings.Contains(string(got), "chat.completion.chunk") == false {
		t.Fatalf("client did not receive upstream stream from gateway: %s", got)
	}
}

func TestRejectsWhenUpstreamUnset(t *testing.T) {
	gw := httptest.NewServer(Handler(Config{}))
	t.Cleanup(gw.Close)
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}
