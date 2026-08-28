package kernel

import (
	"encoding/json"
	"strings"
)

type chatRequest struct {
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Content json.RawMessage `json:"content"`
}

// ExtractChatText 从 OpenAI chat 请求抽出可检测文本；解析失败则退回原文。
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
