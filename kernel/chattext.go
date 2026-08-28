package kernel

import (
	"context"
	"encoding/json"
	"strings"
)

type chatRequest struct {
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Content json.RawMessage `json:"content"`
}

// ExtractChatText pulls inspectable text from an OpenAI chat request. Parse failure returns the raw body.
func ExtractChatText(body []byte) string {
	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil || len(req.Messages) == 0 {
		return string(body)
	}
	var b strings.Builder
	for _, m := range req.Messages {
		if s := contentText(m.Content); s != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(s)
		}
	}
	if b.Len() == 0 {
		return string(body)
	}
	return b.String()
}

// InspectAndRewriteChat inspects each message content string separately (offsets stay in-field).
// enforce=false (observe) never rewrites. Block verdicts skip rewrite; the caller 403s.
func InspectAndRewriteChat(ctx context.Context, inspectors []Inspector, body []byte, enforce bool) (InspectResult, []byte) {
	var req map[string]json.RawMessage
	if err := json.Unmarshal(body, &req); err != nil {
		r := InspectInput(ctx, inspectors, string(body))
		if enforce && r.Redacts() && !r.Blocks() {
			return r, []byte(ApplyRedact(string(body), r))
		}
		return r, body
	}
	msgsRaw, ok := req["messages"]
	if !ok {
		r := InspectInput(ctx, inspectors, ExtractChatText(body))
		return r, body
	}
	var msgs []map[string]json.RawMessage
	if err := json.Unmarshal(msgsRaw, &msgs); err != nil {
		r := InspectInput(ctx, inspectors, ExtractChatText(body))
		return r, body
	}
	var merged InspectResult
	changed := false
	for i := range msgs {
		c, ok := msgs[i]["content"]
		if !ok {
			continue
		}
		newc, r, did := rewriteContent(ctx, inspectors, c, enforce)
		mergeInspect(&merged, r)
		if did {
			msgs[i]["content"] = newc
			changed = true
		}
	}
	if !enforce || merged.Blocks() || !merged.Redacts() || !changed {
		return merged, body
	}
	b, err := json.Marshal(msgs)
	if err != nil {
		return merged, body
	}
	req["messages"] = b
	out, err := json.Marshal(req)
	if err != nil {
		return merged, body
	}
	return merged, out
}

func rewriteContent(ctx context.Context, inspectors []Inspector, raw json.RawMessage, enforce bool) (json.RawMessage, InspectResult, bool) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		r := InspectInput(ctx, inspectors, s)
		if !enforce || !r.Redacts() || r.Blocks() {
			return raw, r, false
		}
		nb, err := json.Marshal(ApplyRedact(s, r))
		if err != nil {
			return raw, r, false
		}
		return nb, r, true
	}
	var parts []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &parts); err != nil {
		r := InspectInput(ctx, inspectors, string(raw))
		return raw, r, false
	}
	var merged InspectResult
	changed := false
	for i := range parts {
		tRaw, ok := parts[i]["text"]
		if !ok {
			continue
		}
		var t string
		if err := json.Unmarshal(tRaw, &t); err != nil {
			continue
		}
		r := InspectInput(ctx, inspectors, t)
		mergeInspect(&merged, r)
		if !enforce || !r.Redacts() || r.Blocks() {
			continue
		}
		nb, err := json.Marshal(ApplyRedact(t, r))
		if err != nil {
			continue
		}
		parts[i]["text"] = nb
		changed = true
	}
	if !changed {
		return raw, merged, false
	}
	out, err := json.Marshal(parts)
	if err != nil {
		return raw, merged, false
	}
	return out, merged, true
}

func contentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var b strings.Builder
		for _, p := range parts {
			if p.Text != "" {
				b.WriteString(p.Text)
			}
		}
		return b.String()
	}
	return string(raw)
}
