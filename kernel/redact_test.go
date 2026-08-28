package kernel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type stubRedact struct {
	needle string
	id     string
	typ    string
}

func (s stubRedact) Name() string { return "stub-redact" }

func (s stubRedact) InspectInput(_ context.Context, text string) InspectResult {
	i := strings.Index(text, s.needle)
	if i < 0 {
		return InspectResult{}
	}
	m := Match{
		RuleID:     s.id,
		EntityType: s.typ,
		Action:     ActionRedact,
		Spans:      []Span{{Start: i, End: i + len(s.needle)}},
	}
	return InspectResult{Hit: true, RuleID: s.id, Matches: []Match{m}}
}

func (s stubRedact) InspectOutputWindow(ctx context.Context, window string) InspectResult {
	return s.InspectInput(ctx, window)
}

func TestAC3_EnforceRedactForwardsPlaceholder(t *testing.T) {
	var upBody atomic.Value
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		upBody.Store(string(b))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      PolicyEnforce,
		Inspectors:      []Inspector{stubRedact{needle: "alice@example.com", id: "pii-email", typ: "email"}},
	}))
	t.Cleanup(gw.Close)

	raw := `{"model":"fake","messages":[{"role":"user","content":"mail alice@example.com please"}]}`
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("redact must not 403, status=%d", resp.StatusCode)
	}
	got, _ := upBody.Load().(string)
	if strings.Contains(got, "alice@example.com") {
		t.Fatalf("upstream still saw plaintext: %s", got)
	}
	if !strings.Contains(got, "[REDACTED:email]") {
		t.Fatalf("upstream missing placeholder: %s", got)
	}
}

func TestAC3_ObserveDoesNotRewrite(t *testing.T) {
	var upBody atomic.Value
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		upBody.Store(string(b))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      PolicyObserve,
		Inspectors:      []Inspector{stubRedact{needle: "alice@example.com", id: "pii-email", typ: "email"}},
	}))
	t.Cleanup(gw.Close)

	raw := `{"model":"fake","messages":[{"role":"user","content":"mail alice@example.com please"}]}`
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, _ := upBody.Load().(string)
	if !strings.Contains(got, "alice@example.com") {
		t.Fatalf("observe must forward plaintext: %s", got)
	}
}

func TestAC7_HashUsesOriginalBody(t *testing.T) {
	raw := `{"model":"fake","messages":[{"role":"user","content":"mail alice@example.com please"}]}`
	want := sha256.Sum256([]byte(raw))
	wantHex := hex.EncodeToString(want[:])

	ch := make(chan Envelope, 1)
	a := NewAuditor(&captureStore{ch: ch}, 8)
	t.Cleanup(a.Close)

	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)

	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		PolicyMode:      PolicyEnforce,
		Auditor:         a,
		Inspectors:      []Inspector{stubRedact{needle: "alice@example.com", id: "pii-email", typ: "email"}},
	}))
	t.Cleanup(gw.Close)

	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	select {
	case env := <-ch:
		if env.PromptHash != wantHex {
			t.Fatalf("hash=%s want=%s", env.PromptHash, wantHex)
		}
		if env.Outcome != OutcomeRedacted {
			t.Fatalf("outcome=%s", env.Outcome)
		}
		if env.EntityType != "email" {
			t.Fatalf("entity_type=%s", env.EntityType)
		}
	case <-time.After(time.Second):
		t.Fatal("missing audit")
	}
}
