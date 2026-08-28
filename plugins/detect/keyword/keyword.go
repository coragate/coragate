package keyword

import (
	"context"
	"fmt"
	"regexp"

	"github.com/coragate/coragate/kernel"
)

// DefaultPattern 第一期演示规则，避免误拦真实业务词。
const DefaultPattern = `(?i)coragate-block-me`

const defaultRuleID = "demo-keyword"

// Rule 一条关键字/正则规则。
type Rule struct {
	ID      string
	Pattern string
}

// Plugin 内置检测插件，与内核同进程链接。
type Plugin struct {
	id string
	re *regexp.Regexp
}

// New 编译一条规则。Pattern 为空时使用 DefaultPattern。
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
