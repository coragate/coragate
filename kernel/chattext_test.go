package kernel

import "testing"

func TestExtractChatText_拼接messages(t *testing.T) {
	body := []byte(`{"model":"x","messages":[{"role":"user","content":"hello"},{"role":"user","content":"world"}]}`)
	got := ExtractChatText(body)
	if got != "hello\nworld" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractChatText_非法JSON退回原文(t *testing.T) {
	got := ExtractChatText([]byte("not-json"))
	if got != "not-json" {
		t.Fatalf("got %q", got)
	}
}
