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
	rawBody := body
	var inputHit, outputHit InspectResult
	outcome := OutcomeForwarded
	defer func() {
		cfg.enqueueAudit(rawBody, started, inputHit, outputHit, outcome)
	}()

	inspectors := cfg.inspectors()
	enforce := cfg.policyMode() == PolicyEnforce
	inputHit, body = InspectAndRewriteChat(r.Context(), inspectors, rawBody, enforce)
	if inputHit.EngineError != "" {
		if cfg.failClosed() {
			outcome = OutcomeFailClosed
			writeEngineClosed(w)
			return
		}
		outcome = OutcomeFailOpen
	} else if inputHit.Blocks() && enforce {
		w.Header().Set(headerHitRule, inputHit.RuleID)
		outcome = OutcomeBlocked
		writeBlocked(w, inputHit.RuleID)
		return
	} else if inputHit.Hit {
		w.Header().Set(headerHitRule, inputHit.RuleID)
		if inputHit.Redacts() && enforce {
			outcome = OutcomeRedacted
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
	if cfg.policyMode() == PolicyEnforce && NeedsOutputHoldback(inspectors) {
		outputHit, _ = copyHoldbackRedact(r.Context(), w, resp.Body, cfg, inspectors)
	} else {
		outputHit, _ = copyFlushScan(r.Context(), w, resp.Body, cfg, inspectors)
	}
	if outputHit.EngineError != "" {
		if cfg.failClosed() {
			outcome = OutcomeFailClosed
		} else {
			outcome = OutcomeFailOpen
		}
	} else if outputHit.Blocks() && cfg.policyMode() == PolicyEnforce {
		outcome = OutcomeBlocked
	} else if outputHit.Redacts() && cfg.policyMode() == PolicyEnforce && outcome == OutcomeForwarded {
		outcome = OutcomeRedacted
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

// copyFlushScan forwards complete SSE lines then scans. Used for block / observe (AC-11: stop later lines; do not recall flushed prefix).
func copyFlushScan(ctx context.Context, dst http.ResponseWriter, src io.Reader, cfg Config, inspectors []Inspector) (InspectResult, error) {
	var last InspectResult
	flusher, _ := dst.(http.Flusher)
	win := newSSEWindow(defaultWindowBytes)
	var pending []byte
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			for {
				i := bytes.IndexByte(pending, '\n')
				if i < 0 {
					break
				}
				line := pending[:i+1]
				pending = pending[i+1:]
				if _, werr := dst.Write(line); werr != nil {
					return last, werr
				}
				if flusher != nil {
					flusher.Flush()
				}
				if len(inspectors) == 0 {
					continue
				}
				for _, snap := range win.Feed(line) {
					hit := InspectOutputWindow(ctx, inspectors, snap)
					if hit.EngineError != "" {
						last = hit
						if cfg.failClosed() {
							return last, nil
						}
						continue
					}
					if hit.Blocks() {
						last = hit
						if cfg.policyMode() == PolicyEnforce {
							return last, nil
						}
					} else if hit.Hit {
						last = hit
					}
				}
			}
		}
		if err == io.EOF {
			if len(pending) > 0 {
				_, _ = dst.Write(pending)
			}
			return last, nil
		}
		if err != nil {
			return last, err
		}
	}
}

func writeAndFlush(dst http.ResponseWriter, flusher http.Flusher, p []byte) error {
	if len(p) == 0 {
		return nil
	}
	if _, err := dst.Write(p); err != nil {
		return err
	}
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

func emitRedactedDelta(dst http.ResponseWriter, flusher http.Flusher, content string) error {
	if content == "" {
		return nil
	}
	line, err := sseDataLine(content)
	if err != nil {
		return err
	}
	return writeAndFlush(dst, flusher, line)
}

// copyHoldbackRedact buffers complete SSE lines, holds back up to OutputHoldbackRunes of extracted
// text, and forwards redacted delta.content so fixture plaintext never enters flushed bytes (AC-4).
func copyHoldbackRedact(ctx context.Context, dst http.ResponseWriter, src io.Reader, cfg Config, inspectors []Inspector) (InspectResult, error) {
	var last InspectResult
	flusher, _ := dst.(http.Flusher)
	var pending []byte
	var extracted strings.Builder
	emitted := ""
	buf := make([]byte, 4096)

	release := func(eof bool) error {
		hold := OutputHoldbackRunes
		if eof {
			hold = 0
		}
		all := extracted.String()
		hit := InspectOutputWindow(ctx, inspectors, all)
		if hit.EngineError != "" {
			last = hit
			if cfg.failClosed() {
				return io.EOF
			}
			return nil
		}
		if hit.Blocks() {
			last = hit
			return errOutputBlocked
		}
		if hit.Hit {
			last = hit
		}
		prefix := splitHoldback(all, hold)
		red := ApplyRedact(prefix, clipRedactToPrefix(hit, len(prefix)))
		if !strings.HasPrefix(red, emitted) {
			if emitted == "" {
				if err := emitRedactedDelta(dst, flusher, red); err != nil {
					return err
				}
				emitted = red
			}
			return nil
		}
		extra := red[len(emitted):]
		if extra == "" {
			return nil
		}
		if err := emitRedactedDelta(dst, flusher, extra); err != nil {
			return err
		}
		emitted = red
		return nil
	}

	handleLine := func(line []byte) error {
		if bytes.HasPrefix(line, []byte("data:")) {
			rest := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
			if bytes.Equal(rest, []byte("[DONE]")) {
				if err := release(true); err != nil {
					return err
				}
				return writeAndFlush(dst, flusher, append(append([]byte{}, line...), '\n'))
			}
		}
		payload, ok := sseDataPayload(line)
		if !ok {
			return writeAndFlush(dst, flusher, append(append([]byte{}, line...), '\n'))
		}
		piece := extractStreamText(payload)
		if piece == "" {
			return writeAndFlush(dst, flusher, append(append([]byte{}, line...), '\n'))
		}
		extracted.WriteString(piece)
		return release(false)
	}

	for {
		n, err := src.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			for {
				i := bytes.IndexByte(pending, '\n')
				if i < 0 {
					break
				}
				line := bytes.TrimSuffix(pending[:i], []byte{'\r'})
				pending = pending[i+1:]
				if herr := handleLine(line); herr != nil {
					if herr == errOutputBlocked {
						return last, nil
					}
					return last, herr
				}
			}
		}
		if err == io.EOF {
			if rel := release(true); rel != nil && rel != errOutputBlocked {
				return last, rel
			}
			return last, nil
		}
		if err != nil {
			return last, err
		}
	}
}

var errOutputBlocked = fmt.Errorf("coragate: output blocked")
