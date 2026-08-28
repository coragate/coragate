package kernel

import "testing"

func TestSSE窗口跨chunk拼出命中文本(t *testing.T) {
	w := newSSEWindow(4096)
	ev := func(content string) []byte {
		return []byte(`data: {"choices":[{"delta":{"content":"` + content + `"}}]}` + "\n")
	}
	if snaps := w.Feed(ev("sec")); len(snaps) != 1 || snaps[0] != "sec" {
		t.Fatalf("第一段窗口 = %v", snaps)
	}
	// 半截 JSON 不得进窗口
	if snaps := w.Feed([]byte(`data: {"choices":[{"delta":{"content":"re`)); len(snaps) != 0 || w.Text() != "sec" {
		t.Fatalf("半截行不应扫描: snaps=%v text=%q", snaps, w.Text())
	}
	if snaps := w.Feed([]byte("t\"}}]}\n")); len(snaps) != 1 || snaps[0] != "secret" {
		t.Fatalf("拼完后窗口 = %v text=%q", snaps, w.Text())
	}
}

func TestSSE忽略DONE与非JSON(t *testing.T) {
	w := newSSEWindow(64)
	w.Feed([]byte("data: [DONE]\n"))
	w.Feed([]byte("data: {\"id\":\"chunk-1\"}\n"))
	if w.Text() != "" {
		t.Fatalf("不应抽出文本: %q", w.Text())
	}
}
