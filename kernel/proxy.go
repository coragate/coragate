package kernel

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	chatPath          = "/v1/chat/completions"
	headerGatewayMark = "X-Coragate-Proxy"
)

var hopByHop = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Proxy-Connection":    true,
	"Te":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

// Handler 返回 OpenAI 兼容面。测试用 httptest 包一层即可。
func Handler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(chatPath, func(w http.ResponseWriter, r *http.Request) {
		proxyChatCompletions(cfg, w, r)
	})
	return mux
}

func proxyChatCompletions(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(cfg.UpstreamBaseURL) == "" {
		http.Error(w, `{"error":{"message":"upstream not configured","type":"coragate_error"}}`, http.StatusServiceUnavailable)
		return
	}
	target, err := joinChatURL(cfg.UpstreamBaseURL)
	if err != nil {
		http.Error(w, `{"error":{"message":"invalid upstream","type":"coragate_error"}}`, http.StatusBadGateway)
		return
	}

	ureq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, r.Body)
	if err != nil {
		http.Error(w, `{"error":{"message":"build upstream request","type":"coragate_error"}}`, http.StatusBadGateway)
		return
	}
	copyRequestHeaders(ureq.Header, r.Header)
	ureq.Header.Set("Accept", "text/event-stream, application/json")

	resp, err := cfg.client().Do(ureq)
	if err != nil {
		http.Error(w, `{"error":{"message":"upstream unreachable","type":"coragate_error"}}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.Header().Set(headerGatewayMark, "1")
	w.WriteHeader(resp.StatusCode)
	_ = copyFlush(w, resp.Body)
}

func joinChatURL(base string) (string, error) {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if b == "" {
		return "", io.ErrUnexpectedEOF
	}
	u, err := url.Parse(b)
	if err != nil {
		return "", err
	}
	path := strings.TrimSuffix(u.Path, "/")
	switch {
	case strings.HasSuffix(path, "/v1/chat/completions"):
		// 已是完整路径
	case strings.HasSuffix(path, "/v1"):
		u.Path = path + "/chat/completions"
	case path == "":
		u.Path = chatPath
	default:
		u.Path = path + chatPath
	}
	return u.String(), nil
}

func copyRequestHeaders(dst, src http.Header) {
	for k, vs := range src {
		if hopByHop[http.CanonicalHeaderKey(k)] {
			continue
		}
		switch http.CanonicalHeaderKey(k) {
		case "Host", "Content-Length":
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func copyResponseHeaders(dst, src http.Header) {
	for k, vs := range src {
		if hopByHop[http.CanonicalHeaderKey(k)] {
			continue
		}
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

// copyFlush 边读边写并 Flush，禁止攒完全部 body 再吐给客户端。
func copyFlush(dst http.ResponseWriter, src io.Reader) error {
	flusher, _ := dst.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}
