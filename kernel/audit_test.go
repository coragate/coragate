package kernel

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type blockStore struct {
	enter chan struct{}
	once  sync.Once
	wait  chan struct{}
	mu    sync.Mutex
	got   []Envelope
}

func (b *blockStore) Append(_ context.Context, env Envelope) error {
	b.once.Do(func() { close(b.enter) })
	<-b.wait
	b.mu.Lock()
	b.got = append(b.got, env)
	b.mu.Unlock()
	return nil
}

func (b *blockStore) List(context.Context, int) ([]Envelope, error) { return nil, nil }

func TestAC7_EnqueueDoesNotWaitOnAdapter(t *testing.T) {
	st := &blockStore{enter: make(chan struct{}), wait: make(chan struct{})}
	a := NewAuditor(st, 8)
	t.Cleanup(func() {
		close(st.wait)
		a.Close()
	})
	t0 := time.Now()
	a.Enqueue(Envelope{
		SchemaVersion: EnvelopeSchemaVersion,
		Time:          time.Now().UTC().Format(time.RFC3339Nano),
		RuleID:        RuleNone,
		PromptHash:    "aa",
		PolicyMode:    PolicyEnforce,
	})
	if time.Since(t0) > 80*time.Millisecond {
		t.Fatal("Enqueue blocked on flush")
	}
	select {
	case <-st.enter:
	case <-time.After(time.Second):
		t.Fatal("worker did not call Append")
	}
}

func TestAC7_SSECompletesWithoutWaitingOnDisk(t *testing.T) {
	st := &blockStore{enter: make(chan struct{}), wait: make(chan struct{})}
	a := NewAuditor(st, 8)
	t.Cleanup(func() {
		close(st.wait)
		a.Close()
	})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"id\":\"1\"}\n\n")
	}))
	t.Cleanup(up.Close)
	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		Auditor:         a,
	}))
	t.Cleanup(gw.Close)

	t0 := time.Now()
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if time.Since(t0) > 500*time.Millisecond {
		t.Fatal("SSE hot path looks like it waited on audit flush")
	}
	select {
	case <-st.enter:
	case <-time.After(time.Second):
		t.Fatal("worker still has not Append after request")
	}
}

func TestAC7_EnvelopeFields(t *testing.T) {
	ch := make(chan Envelope, 1)
	st := &captureStore{ch: ch}
	a := NewAuditor(st, 8)
	t.Cleanup(a.Close)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.Close)
	gw := httptest.NewServer(Handler(Config{
		UpstreamBaseURL: up.URL,
		Auditor:         a,
		Inspectors:      []Inspector{stubInspector{needle: "zzz", id: "no"}},
	}))
	t.Cleanup(gw.Close)
	resp, err := http.Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	select {
	case env := <-ch:
		if env.SchemaVersion != EnvelopeSchemaVersion {
			t.Fatalf("schema_version=%d", env.SchemaVersion)
		}
		if env.RuleID != RuleNone {
			t.Fatalf("rule_id=%s", env.RuleID)
		}
		if env.PromptHash == "" || env.PolicyMode == "" || env.Time == "" {
			t.Fatalf("missing fields: %+v", env)
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive envelope")
	}
}

func TestAuditorDroppedCountsFullQueue(t *testing.T) {
	st := &blockStore{enter: make(chan struct{}), wait: make(chan struct{})}
	a := NewAuditor(st, 1)
	t.Cleanup(func() {
		close(st.wait)
		a.Close()
	})
	env := Envelope{RuleID: "drop"}
	a.Enqueue(env)
	select {
	case <-st.enter:
	case <-time.After(time.Second):
		t.Fatal("worker did not take first event")
	}
	a.Enqueue(env)
	a.Enqueue(env)
	if a.Dropped() < 1 {
		t.Fatalf("dropped=%d, want at least 1", a.Dropped())
	}
}

type captureStore struct{ ch chan Envelope }

func (c *captureStore) Append(_ context.Context, env Envelope) error {
	c.ch <- env
	return nil
}
func (c *captureStore) List(context.Context, int) ([]Envelope, error) { return nil, nil }
