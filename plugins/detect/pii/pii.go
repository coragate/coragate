package pii

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coragate/coragate/kernel"
)

const (
	TypeEmail    = "email"
	TypePhoneCN  = "phone_cn"
	TypeIDCardCN = "id_card_cn"
	TypeBankCard = "bank_card"
)

var emailRe = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
var idCardRe = regexp.MustCompile(`[0-9]{17}[0-9Xx]`)
var bankCardRe = regexp.MustCompile(`[0-9]{13,19}`)

var idWeights = []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
var idChecksum = []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}

// Rule is one PII entity rule.
type Rule struct {
	ID     string
	Type   string
	Action string
}

// Plugin detects one entity type.
type Plugin struct {
	id     string
	typ    string
	action string
}

// New builds a detector for a single entity type.
func New(rule Rule) (*Plugin, error) {
	switch rule.Type {
	case TypeEmail, TypePhoneCN, TypeIDCardCN, TypeBankCard:
	default:
		return nil, fmt.Errorf("pii: unknown type %q", rule.Type)
	}
	id := rule.ID
	if id == "" {
		id = "pii-" + rule.Type
	}
	return &Plugin{
		id:     id,
		typ:    rule.Type,
		action: kernel.ResolveAction(kernel.PluginPII, rule.Action),
	}, nil
}

func (p *Plugin) Name() string { return kernel.PluginPII }

func (p *Plugin) InspectInput(_ context.Context, text string) kernel.InspectResult {
	return p.match(text)
}

func (p *Plugin) InspectOutputWindow(_ context.Context, window string) kernel.InspectResult {
	return p.match(window)
}

func (p *Plugin) match(text string) kernel.InspectResult {
	if p == nil {
		return kernel.InspectResult{}
	}
	var spans []kernel.Span
	switch p.typ {
	case TypeEmail:
		spans = indexAll(emailRe, text)
	case TypePhoneCN:
		spans = phoneCNSpans(text)
	case TypeIDCardCN:
		for _, loc := range idCardRe.FindAllStringIndex(text, -1) {
			if idChecksumOK(text[loc[0]:loc[1]]) {
				spans = append(spans, kernel.Span{Start: loc[0], End: loc[1]})
			}
		}
	case TypeBankCard:
		for _, loc := range bankCardRe.FindAllStringIndex(text, -1) {
			num := text[loc[0]:loc[1]]
			if !digitRunBounded(text, loc[0], loc[1]) {
				continue
			}
			if len(num) == 18 && idChecksumOK(num) {
				continue
			}
			if luhnOK(num) {
				spans = append(spans, kernel.Span{Start: loc[0], End: loc[1]})
			}
		}
	}
	if len(spans) == 0 {
		return kernel.InspectResult{}
	}
	m := kernel.Match{
		RuleID:     p.id,
		EntityType: p.typ,
		Action:     p.action,
		Spans:      spans,
	}
	return kernel.InspectResult{Hit: true, RuleID: p.id, Matches: []kernel.Match{m}}
}

func indexAll(re *regexp.Regexp, text string) []kernel.Span {
	locs := re.FindAllStringIndex(text, -1)
	out := make([]kernel.Span, 0, len(locs))
	for _, loc := range locs {
		out = append(out, kernel.Span{Start: loc[0], End: loc[1]})
	}
	return out
}

// phoneCNSpans finds 11-digit mainland mobiles that are not a prefix of a longer digit run.
func phoneCNSpans(text string) []kernel.Span {
	var out []kernel.Span
	b := []byte(text)
	for i := 0; i+11 <= len(b); i++ {
		if b[i] != '1' {
			continue
		}
		if b[i+1] < '3' || b[i+1] > '9' {
			continue
		}
		ok := true
		for j := 2; j < 11; j++ {
			if b[i+j] < '0' || b[i+j] > '9' {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if i > 0 && isDigitByte(b[i-1]) {
			continue
		}
		if i+11 < len(b) && isDigitByte(b[i+11]) {
			continue
		}
		out = append(out, kernel.Span{Start: i, End: i + 11})
	}
	return out
}

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }

func digitRunBounded(text string, start, end int) bool {
	if start > 0 && isDigitByte(text[start-1]) {
		return false
	}
	if end < len(text) && isDigitByte(text[end]) {
		return false
	}
	return true
}

func idChecksumOK(id string) bool {
	if len(id) != 18 {
		return false
	}
	sum := 0
	for i := 0; i < 17; i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
		sum += int(id[i]-'0') * idWeights[i]
	}
	want := idChecksum[sum%11]
	got := id[17]
	if got == 'x' {
		got = 'X'
	}
	return got == want
}

func luhnOK(s string) bool {
	sum := 0
	alt := false
	for i := len(s) - 1; i >= 0; i-- {
		n := int(s[i] - '0')
		if n < 0 || n > 9 {
			return false
		}
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}
