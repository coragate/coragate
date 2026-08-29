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

// DefaultAction is the snapshot default when action is omitted.
const DefaultAction = kernel.ActionBlock

// Info is the host catalog entry (Factory Method registry).
func Info() kernel.PluginInfo {
	return kernel.PluginInfo{
		ID:            kernel.PluginKeyword,
		DefaultAction: DefaultAction,
		NeedsPattern:  true,
	}
}

// Compile builds an Inspector from a snapshot row.
func Compile(r kernel.SnapshotRule) (kernel.Inspector, error) {
	return New(Rule{ID: r.ID, Pattern: r.Pattern, Action: r.Action})
}

// Rule is one keyword/regex rule.
type Rule struct {
	ID      string
	Pattern string
	Action  string
}

// Plugin is the built-in detector, linked in-process with the kernel.
type Plugin struct {
	id     string
	action string
	re     *regexp.Regexp
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
		return nil, fmt.Errorf("keyword: compile rule %s: %w", id, err)
	}
	return &Plugin{
		id:     id,
		action: kernel.ResolveAction(rule.Action, DefaultAction),
		re:     re,
	}, nil
}

func (p *Plugin) Name() string { return kernel.PluginKeyword }

// OutputRedacts reports whether this rule rewrites streaming output (hold-back path).
func (p *Plugin) OutputRedacts() bool {
	return p != nil && p.action == kernel.ActionRedact
}

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
	loc := p.re.FindStringIndex(text)
	if loc == nil {
		return kernel.InspectResult{}
	}
	m := kernel.Match{
		RuleID: p.id,
		Action: p.action,
		Spans:  []kernel.Span{{Start: loc[0], End: loc[1]}},
	}
	return kernel.InspectResult{Hit: true, RuleID: p.id, Matches: []kernel.Match{m}}
}
