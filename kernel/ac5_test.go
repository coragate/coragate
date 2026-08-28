package kernel

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestAC5_面板未启动仍代理与拦截 只起 Go Handler，不启动 Next 控制面。
func TestAC5_面板未启动仍代理与拦截(t *testing.T) {
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
		t.Fatal("未起面板时放行请求应打到上游")
	}
	if okResp.StatusCode != http.StatusOK || !strings.Contains(string(okBody), "no-panel") {
		t.Fatalf("放行失败 status=%d body=%s", okResp.StatusCode, okBody)
	}
	if okResp.Header.Get(headerGatewayMark) != "1" {
		t.Fatal("响应应来自数据面")
	}

	forwarded.Store(false)
	blockResp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"fake","messages":[{"role":"user","content":"secret"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(blockResp.Body)
	_ = blockResp.Body.Close()
	if forwarded.Load() {
		t.Fatal("未起面板时 enforce 命中仍打了上游")
	}
	if blockResp.StatusCode != http.StatusForbidden {
		t.Fatalf("拦截状态码 = %d", blockResp.StatusCode)
	}
}
