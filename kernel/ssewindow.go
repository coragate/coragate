package kernel

import (
	"bytes"
	"encoding/json"
	"strings"
)

const defaultWindowBytes = 4096

// sseWindow buffers text extracted from complete SSE lines. It does not parse partial JSON (ADR-0003).
type sseWindow struct {
	partial []byte
	text    strings.Builder
	max     int
}

func newSSEWindow(max int) *sseWindow {
	if max <= 0 {
		max = defaultWindowBytes
	}
	return &sseWindow{max: max}
}

func (s *sseWindow) Text() string { return s.text.String() }

// Feed consumes a raw chunk. Each complete data: line updates the window and returns a snapshot for plugins.
func (s *sseWindow) Feed(p []byte) []string {
	if s == nil {
		return nil
	}
	s.partial = append(s.partial, p...)
	var snaps []string
	for {
		i := bytes.IndexByte(s.partial, '\n')
		if i < 0 {
			break
		}
		line := bytes.TrimSuffix(s.partial[:i], []byte{'\r'})
		s.partial = s.partial[i+1:]
		payload, ok := sseDataPayload(line)
		if !ok {
			continue
		}
		piece := extractStreamText(payload)
		if piece == "" {
			continue
		}
		s.append(piece)
		snaps = append(snaps, s.Text())
	}
	return snaps
}

func sseDataPayload(line []byte) ([]byte, bool) {
	if !bytes.HasPrefix(line, []byte("data:")) && !bytes.HasPrefix(line, []byte("data: ")) {
		return nil, false
	}
	payload := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
	if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
		return nil, false
	}
	return payload, true
}

func (s *sseWindow) append(piece string) {
	s.text.WriteString(piece)
	if s.text.Len() <= s.max {
		return
	}
	all := s.text.String()
	s.text.Reset()
	s.text.WriteString(all[len(all)-s.max:])
}

// extractStreamText reads delta.content from a complete JSON object. Failures are ignored; partial JSON is not parsed.
func extractStreamText(payload []byte) string {
	var obj struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &obj); err != nil {
		return ""
	}
	var b strings.Builder
	for _, c := range obj.Choices {
		b.WriteString(c.Delta.Content)
	}
	return b.String()
}
