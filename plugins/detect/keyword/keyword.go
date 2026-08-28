package keyword

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coragate/coragate/kernel"
)

// DefaultPattern is the phase-1 demo rule, chosen not to collide with real business words.
const DefaultPattern = `(?i)coragate-block-me`

const defaultRuleID = "demo-keyword"

// Rule is one keyword/regex rule.
type Rule struct {
	ID      string
	Pattern string
}

// Plugin is the built-in detector, linked in-process with the kernel.
type Plugin struct {
	id string
	re *regexp.Regexp
}

// New compiles one rule. An empty Pattern uses DefaultPattern.
func New(rule Rule) (*Plugin, error) {
	id := rule.ID
	if id == "" {
		id = defaultRuleID
	}
	pat := rule.Pattern
	if pat == "" {
		pat = DefaultPattern
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, fmt.Errorf("keyword: 编译规则 %s: %w", id, err)
	}
	return &Plugin{id: id, re: re}, nil
}

func (p *Plugin) Name() string { return "keyword" }

func (p *Plugin) InspectInput(_ context.Context, text string) kernel.InspectResult {
	return p.match(text)
}

func (p *Plugin) InspectOutputWindow(_ context.Context, window string) kernel.InspectResult {
	return p.match(window)
}

func (p *Plugin) match(text string) kernel.InspectResult {
	if p == nil || p.re == nil {
		return kernel.InspectResult{}
	}
	if p.re.FindString(text) == "" {
		return kernel.InspectResult{}
	}
	return kernel.InspectResult{Hit: true, RuleID: p.id}
}
