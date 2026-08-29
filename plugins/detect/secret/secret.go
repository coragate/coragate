package secret

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coragate/coragate/kernel"
)

const (
	TypeAWSAccessKey  = "aws_access_key"
	TypeGitHubPAT     = "github_pat"
	TypeOpenAIAPIKey  = "openai_api_key"
	TypePEMPrivateKey = "pem_private_key"
)

// DefaultAction is the snapshot default when action is omitted.
const DefaultAction = kernel.ActionBlock

// Info is the host catalog entry (Factory Method registry).
func Info() kernel.PluginInfo {
	return kernel.PluginInfo{
		ID:            kernel.PluginSecret,
		DefaultAction: DefaultAction,
		EntityTypes:   []string{TypeAWSAccessKey, TypeGitHubPAT, TypeOpenAIAPIKey, TypePEMPrivateKey},
	}
}

// Compile builds an Inspector from a snapshot row.
func Compile(r kernel.SnapshotRule) (kernel.Inspector, error) {
	return New(Rule{ID: r.ID, Type: r.Type, Action: r.Action})
}

// 夹具表（公开假值，SPEC-secret-entities）。
const (
	FixtureS1Hit       = `AKIAIOSFODNN7EXAMPLE`
	FixtureS1Miss      = `AKIAIOSFODNN7`
	FixtureS2Hit       = `ghp_0123456789abcdefghijklmnopqrstuvwxyz`
	FixtureS2Miss      = `ghp_short`
	FixtureS3Hit       = `sk-proj-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUV`
	FixtureS3MissSkill = `skill issue`
	FixtureS3MissShort = `sk-xx`
	FixtureS4Hit       = `-----BEGIN PRIVATE KEY-----`
	FixtureS4MissPub   = `-----BEGIN PUBLIC KEY-----`
	FixtureS4MissCert  = `-----BEGIN CERTIFICATE-----`
)

var (
	awsAccessKeyRe  = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	githubPATRe     = regexp.MustCompile(`\bghp_[A-Za-z0-9]{36}\b`)
	openaiAPIKeyRe  = regexp.MustCompile(`\bsk-proj-[A-Za-z0-9]{20,}\b`)
	pemPrivateKeyRe = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)
)

// Rule is one secret entity rule.
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

// New builds a detector for a single entity type. type is required.
func New(rule Rule) (*Plugin, error) {
	if _, ok := matchers[rule.Type]; !ok {
		return nil, fmt.Errorf("secret: unknown type %q", rule.Type)
	}
	id := rule.ID
	if id == "" {
		id = "secret-" + rule.Type
	}
	return &Plugin{
		id:     id,
		typ:    rule.Type,
		action: kernel.ResolveAction(rule.Action, DefaultAction),
	}, nil
}

func (p *Plugin) Name() string { return kernel.PluginSecret }

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
	if p == nil {
		return kernel.InspectResult{}
	}
	fn, ok := matchers[p.typ]
	if !ok {
		return kernel.InspectResult{}
	}
	spans := fn(text)
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

type matcher func(string) []kernel.Span

var matchers = map[string]matcher{
	TypeAWSAccessKey:  func(text string) []kernel.Span { return indexFirst(awsAccessKeyRe, text) },
	TypeGitHubPAT:     func(text string) []kernel.Span { return indexFirst(githubPATRe, text) },
	TypeOpenAIAPIKey:  func(text string) []kernel.Span { return indexFirst(openaiAPIKeyRe, text) },
	TypePEMPrivateKey: func(text string) []kernel.Span { return indexFirst(pemPrivateKeyRe, text) },
}

func indexFirst(re *regexp.Regexp, text string) []kernel.Span {
	loc := re.FindStringIndex(text)
	if loc == nil {
		return nil
	}
	return []kernel.Span{{Start: loc[0], End: loc[1]}}
}

// LongestFixtureBytes returns the max UTF-8 byte length of must-hit fixtures.
func LongestFixtureBytes() int {
	n := 0
	for _, s := range []string{FixtureS1Hit, FixtureS2Hit, FixtureS3Hit, FixtureS4Hit} {
		if l := len(s); l > n {
			n = l
		}
	}
	return n
}

// MustFitWindow: fixtures must fit the kernel sliding window.
func MustFitWindow() bool {
	return LongestFixtureBytes() <= kernel.DefaultWindowBytes
}
