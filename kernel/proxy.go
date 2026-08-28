package kernel

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	chatPath          = "/v1/chat/completions"
	inspectPath       = "/v1/inspect"
	reloadPath        = "/v1/reload"
	rulesPath         = "/v1/rules"
	auditPath         = "/v1/audit"
	healthPath        = "/health"
	headerGatewayMark = "X-Coragate-Proxy"
	headerHitRule     = "X-Coragate-Hit"
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

// Handler returns the OpenAI-compatible surface. Tests can wrap it with httptest.
func Handler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(chatPath, func(w http.ResponseWriter, r *http.Request) {
		proxyChatCompletions(cfg, w, r)
	})
	mux.HandleFunc(inspectPath, func(w http.ResponseWriter, r *http.Request) {
		handleInspect(cfg, w, r)
	})
	mux.HandleFunc(reloadPath, func(w http.ResponseWriter, r *http.Request) {
		handleReload(cfg, w, r)
	})
	mux.HandleFunc(rulesPath, func(w http.ResponseWriter, r *http.Request) {
		handleRules(cfg, w, r)
	})
	mux.HandleFunc(auditPath, func(w http.ResponseWriter, r *http.Request) {
		handleAudit(cfg, w, r)
	})
	mux.HandleFunc(healthPath, handleHealth)
	return mux
}

func proxyChatCompletions(cfg Config, w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headerGatewayMark, "1")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":{"message":"method not allowed","type":"coragate_error"}}`, http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":{"message":"read body","type":"coragate_error"}}`, http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	started := time.Now()
	var inputHit, outputHit InspectResult
	outcome := OutcomeForwarded
	defer func() {
		cfg.enqueueAudit(body, started, inputHit, outputHit, outcome)
	}()

	inspectors := cfg.inspectors()
	inputHit = InspectInput(r.Context(), inspectors, ExtractChatText(body))
	if inputHit.EngineError != "" {
		if cfg.failClosed() {
			outcome = OutcomeFailClosed
			writeEngineClosed(w)
			return
		}
		outcome = OutcomeFailOpen
	} else if inputHit.Hit {
		w.Header().Set(headerHitRule, inputHit.RuleID)
		if cfg.policyMode() == PolicyEnforce {
			outcome = OutcomeBlocked
			writeBlocked(w, inputHit.RuleID)
			return
		}
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

	ureq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		http.Error(w, `{"error":{"message":"build upstream request","type":"coragate_error"}}`, http.StatusBadGateway)
		return
	}
	copyRequestHeaders(ureq.Header, r.Header)
	ureq.ContentLength = int64(len(body))
	ureq.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	ureq.Header.Set("Accept", "text/event-stream, application/json")

	resp, err := cfg.client().Do(ureq)
	if err != nil {
		http.Error(w, `{"error":{"message":"upstream unreachable","type":"coragate_error"}}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.Header().Set(headerGatewayMark, "1")
	if inputHit.Hit {
		w.Header().Set(headerHitRule, inputHit.RuleID)
	}
	w.WriteHeader(resp.StatusCode)
	outputHit, _ = copyFlushScan(r.Context(), w, resp.Body, cfg, inspectors)
	if outputHit.EngineError != "" {
		if cfg.failClosed() {
			outcome = OutcomeFailClosed
		} else {
			outcome = OutcomeFailOpen
		}
	} else if outputHit.Hit && cfg.policyMode() == PolicyEnforce {
		outcome = OutcomeBlocked
	}
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
		// Already a full path.
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
		if http.CanonicalHeaderKey(k) == "Host" {
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

// copyFlushScan forwards while scanning: raw chunks Flush immediately; complete data: lines enter the sliding window and plugins.
func copyFlushScan(ctx context.Context, dst http.ResponseWriter, src io.Reader, cfg Config, inspectors []Inspector) (InspectResult, error) {
	var last InspectResult
	flusher, _ := dst.(http.Flusher)
	win := newSSEWindow(defaultWindowBytes)
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			if _, werr := dst.Write(chunk); werr != nil {
				return last, werr
			}
			if flusher != nil {
				flusher.Flush()
			}
			if len(inspectors) > 0 {
				for _, snap := range win.Feed(chunk) {
					hit := InspectOutputWindow(ctx, inspectors, snap)
					if hit.EngineError != "" {
						last = hit
						if cfg.failClosed() {
							return last, nil
						}
						continue
					}
					if hit.Hit {
						last = hit
						if cfg.policyMode() == PolicyEnforce {
							return last, nil
						}
					}
				}
			}
		}
		if err == io.EOF {
			return last, nil
		}
		if err != nil {
			return last, err
		}
	}
}
