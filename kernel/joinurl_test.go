package kernel

import (
	"strings"
	"testing"
)

func TestJoinChatURL(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", wantErr: true},
		{in: "   ", wantErr: true},
		{in: "http://up.example", want: "http://up.example/v1/chat/completions"},
		{in: "http://up.example/", want: "http://up.example/v1/chat/completions"},
		{in: "http://up.example/v1", want: "http://up.example/v1/chat/completions"},
		{in: "http://up.example/v1/", want: "http://up.example/v1/chat/completions"},
		{in: "http://up.example/v1/chat/completions", want: "http://up.example/v1/chat/completions"},
		{in: "http://up.example/proxy", want: "http://up.example/proxy/v1/chat/completions"},
	}
	for _, tc := range cases {
		got, err := joinChatURL(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("joinChatURL(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("joinChatURL(%q): %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("joinChatURL(%q)=%q want %q", tc.in, got, tc.want)
		}
		if strings.Contains(got, "//v1") && !strings.HasPrefix(got, "http://") {
			t.Fatalf("unexpected URL %q", got)
		}
	}
}

func TestResolveActionUsesCallerDefault(t *testing.T) {
	if got := ResolveAction("", ActionRedact); got != ActionRedact {
		t.Fatalf("empty + redact default = %s", got)
	}
	if got := ResolveAction("", ActionBlock); got != ActionBlock {
		t.Fatalf("empty + block default = %s", got)
	}
	if got := ResolveAction(ActionBlock, ActionRedact); got != ActionBlock {
		t.Fatalf("explicit block must win, got %s", got)
	}
	if got := ResolveAction(ActionRedact, ActionBlock); got != ActionRedact {
		t.Fatalf("explicit redact must win, got %s", got)
	}
}
