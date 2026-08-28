package kernel

import "testing"

func TestSSEWindowJoinsHitTextAcrossChunks(t *testing.T) {
	w := newSSEWindow(4096)
	ev := func(content string) []byte {
		return []byte(`data: {"choices":[{"delta":{"content":"` + content + `"}}]}` + "\n")
	}
	if snaps := w.Feed(ev("sec")); len(snaps) != 1 || snaps[0] != "sec" {
		t.Fatalf("first window = %v", snaps)
	}
	// Partial JSON must not enter the window
	if snaps := w.Feed([]byte(`data: {"choices":[{"delta":{"content":"re`)); len(snaps) != 0 || w.Text() != "sec" {
		t.Fatalf("partial line should not be scanned: snaps=%v text=%q", snaps, w.Text())
	}
	if snaps := w.Feed([]byte("t\"}}]}\n")); len(snaps) != 1 || snaps[0] != "secret" {
		t.Fatalf("joined window = %v text=%q", snaps, w.Text())
	}
}

func TestSSEIgnoresDONEAndNonJSON(t *testing.T) {
	w := newSSEWindow(64)
	w.Feed([]byte("data: [DONE]\n"))
	w.Feed([]byte("data: {\"id\":\"chunk-1\"}\n"))
	if w.Text() != "" {
		t.Fatalf("should not extract text: %q", w.Text())
	}
}
