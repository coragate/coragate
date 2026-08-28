package kernel

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAC11_HostDefaultListenAddresses(t *testing.T) {
	t.Setenv("CORAGATE_LISTEN", "")
	if got := LoadConfig(HostClient).Listen; got != DefaultListenClient {
		t.Fatalf("client default listen = %s, want %s", got, DefaultListenClient)
	}
	if got := LoadConfig(HostCluster).Listen; got != DefaultListenCluster {
		t.Fatalf("cluster default listen = %s, want %s", got, DefaultListenCluster)
	}
}

func TestAC11_ListenEnvOverridesHostDefault(t *testing.T) {
	t.Setenv("CORAGATE_LISTEN", "127.0.0.1:9999")
	if got := LoadConfig(HostCluster).Listen; got != "127.0.0.1:9999" {
		t.Fatalf("cluster listen after override = %s", got)
	}
	if got := LoadConfig(HostClient).Listen; got != "127.0.0.1:9999" {
		t.Fatalf("client listen after override = %s", got)
	}
}

func TestAC11_ClientAndClusterBindsCompleteStream(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl := w.(http.Flusher)
		_, _ = io.WriteString(w, "data: {\"id\":\"host-ok\"}\n\n")
		fl.Flush()
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
	t.Cleanup(up.Close)

	for _, bind := range []string{"127.0.0.1:0", "0.0.0.0:0"} {
		t.Run(bind, func(t *testing.T) {
			base := serveOn(t, bind, Handler(Config{UpstreamBaseURL: up.URL}))
			req, err := http.NewRequest(http.MethodPost, base+"/v1/chat/completions", strings.NewReader(`{"model":"fake","stream":true}`))
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
			if resp.Header.Get(headerGatewayMark) != "1" {
				t.Fatal("response did not pass through the gateway")
			}
			first, err := bufio.NewReader(resp.Body).ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(first, "host-ok") {
				t.Fatalf("first chunk = %q", first)
			}
		})
	}
}

func serveOn(t *testing.T, listen string, h http.Handler) string {
	t.Helper()
	ln, err := net.Listen("tcp4", listen)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	port := ln.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}
