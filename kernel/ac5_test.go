package kernel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestAC5_DataplaneWorksWithoutControlPlane runs only the Go Handler; Next is not started.
func TestAC5_DataplaneWorksWithoutControlPlane(t *testing.T) {
	var forwarded atomic.Bool
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded.Store(true)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"id\":\"no-panel\"}\n\n")
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      PolicyEnforce,
		Inspectors:      []Inspector{stubInspector{needle: "secret", id: "t-ac5"}},
	}))
	t.Cleanup(gw.Close)

	okResp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	okBody, _ := io.ReadAll(okResp.Body)
	_ = okResp.Body.Close()
	if !forwarded.Load() {
		t.Fatal("allowed request should reach upstream without the panel")
	}
	if okResp.StatusCode != http.StatusOK || !strings.Contains(string(okBody), "no-panel") {
		t.Fatalf("allow failed status=%d body=%s", okResp.StatusCode, okBody)
	}
	if okResp.Header.Get(headerGatewayMark) != "1" {
		t.Fatal("response should come from the dataplane")
	}

	forwarded.Store(false)
	blockResp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","messages":[{"role":"user","content":"secret"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(blockResp.Body)
	_ = blockResp.Body.Close()
	if forwarded.Load() {
		t.Fatal("enforce hit still reached upstream without the panel")
	}
	if blockResp.StatusCode != http.StatusForbidden {
		t.Fatalf("block status = %d", blockResp.StatusCode)
	}
}
