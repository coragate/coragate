package kernel

import (
	"context"
	"fmt"
	"sort"
)

// PolicyEnforce blocks on a hit and skips upstream. PolicyObserve still detects and audits, but a hit does not block.
const (
	PolicyEnforce = "enforce"
	PolicyObserve = "observe"
)

// Behavior when the detection engine is unavailable (AC-9 / ADR-0007). Default is fail_open.
const (
	FailOpen   = "fail_open"
	FailClosed = "fail_closed"
)

const (
	OutcomeForwarded  = "forwarded"
	OutcomeBlocked    = "blocked"
	OutcomeRedacted   = "redacted"
	OutcomeFailOpen   = "fail_open"
	OutcomeFailClosed = "fail_closed"
)

// Rule actions (SPEC-pii-entities).
const (
	ActionBlock  = "block"
	ActionRedact = "redact"
)

const (
	InterventionNone      = "none"
	InterventionAuditOnly = "audit_only"
	InterventionApplied   = "applied"
)

const PluginPII = "pii"
const PluginKeyword = "keyword"

// Span is a byte range in the string passed to the Inspector (extracted text or window snapshot).
type Span struct {
	Start int
	End   int
}

// Match is one plugin hit. A single Inspector call may return many Matches.
type Match struct {
	RuleID     string
	EntityType string
	Action     string
	Spans      []Span
}

// InspectResult is a plugin verdict on a text span. Matching logic stays inside the plugin.
// A non-empty EngineError means the engine is down, not a rule hit.
type InspectResult struct {
	Hit         bool
	RuleID      string
	EngineError string
	Matches     []Match
}

// Inspector is the dataplane detection plugin interface (AC-10). Phase 1 registers in-process; no plugin.Open.
type Inspector interface {
	Name() string
	InspectInput(ctx context.Context, text string) InspectResult
	InspectOutputWindow(ctx context.Context, window string) InspectResult
}

// ResolveAction fills a missing action: pii defaults to redact, everything else to block.
func ResolveAction(plugin, action string) string {
	if action == ActionBlock || action == ActionRedact {
		return action
	}
	if plugin == PluginPII {
		return ActionRedact
	}
	return ActionBlock
}

// MatchAction returns block when Action is empty (legacy keyword / stubs).
func MatchAction(m Match) string {
	if m.Action == ActionRedact {
		return ActionRedact
	}
	return ActionBlock
}

// Blocks reports an enforce-style block verdict (input 403 / output stop).
func (r InspectResult) Blocks() bool {
	if r.EngineError != "" {
		return false
	}
	for _, m := range r.Matches {
		if MatchAction(m) == ActionBlock {
			return true
		}
	}
	return r.Hit && len(r.Matches) == 0
}

// Redacts reports at least one redact span.
func (r InspectResult) Redacts() bool {
	for _, m := range r.Matches {
		if MatchAction(m) == ActionRedact {
			return true
		}
	}
	return false
}

// Placeholder builds the stable redact token for a match.
func Placeholder(m Match) string {
	typ := m.EntityType
	if typ == "" {
		typ = PluginKeyword
	}
	return "[REDACTED:" + typ + "]"
}

// ApplyRedact replaces redact spans from right to left. Block spans are left untouched.
func ApplyRedact(text string, r InspectResult) string {
	type piece struct {
		start, end int
		repl       string
	}
	var ps []piece
	for _, m := range r.Matches {
		if MatchAction(m) != ActionRedact {
			continue
		}
		ph := Placeholder(m)
		for _, s := range m.Spans {
			if s.Start < 0 || s.End > len(text) || s.Start >= s.End {
				continue
			}
			ps = append(ps, piece{s.Start, s.End, ph})
		}
	}
	if len(ps) == 0 {
		return text
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].start != ps[j].start {
			return ps[i].start < ps[j].start
		}
		return (ps[i].end - ps[i].start) > (ps[j].end - ps[j].start)
	})
	var kept []piece
	for _, p := range ps {
		overlap := false
		for _, q := range kept {
			if p.start < q.end && p.end > q.start {
				overlap = true
				break
			}
		}
		if !overlap {
			kept = append(kept, p)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].start > kept[j].start })
	b := []byte(text)
	for _, p := range kept {
		if p.end > len(b) || p.start > len(b) {
			continue
		}
		b = append(append([]byte{}, b[:p.start]...), append([]byte(p.repl), b[p.end:]...)...)
	}
	return string(b)
}

func mergeInspect(dst *InspectResult, r InspectResult) {
	if r.EngineError != "" {
		*dst = r
		return
	}
	if !r.Hit && len(r.Matches) == 0 {
		return
	}
	dst.Hit = true
	if len(r.Matches) > 0 {
		dst.Matches = append(dst.Matches, r.Matches...)
	} else {
		dst.Matches = append(dst.Matches, Match{RuleID: r.RuleID, Action: ActionBlock})
	}
	dst.RuleID = primaryRuleID(*dst)
}

func primaryRuleID(r InspectResult) string {
	for _, m := range r.Matches {
		if MatchAction(m) == ActionBlock && m.RuleID != "" {
			return m.RuleID
		}
	}
	for _, m := range r.Matches {
		if m.RuleID != "" {
			return m.RuleID
		}
	}
	return r.RuleID
}

func primaryMatch(r InspectResult) Match {
	id := primaryRuleID(r)
	for _, m := range r.Matches {
		if m.RuleID == id {
			return m
		}
	}
	if len(r.Matches) > 0 {
		return r.Matches[0]
	}
	return Match{RuleID: r.RuleID, Action: ActionBlock}
}

// InspectInput calls every plugin and merges hits (SPEC-pii-entities: all spans, not first-plugin-wins).
// EngineError still short-circuits.
func InspectInput(ctx context.Context, inspectors []Inspector, text string) InspectResult {
	var out InspectResult
	for _, p := range inspectors {
		if p == nil {
			continue
		}
		r := safeInspect(func() InspectResult { return p.InspectInput(ctx, text) })
		if r.EngineError != "" {
			return r
		}
		mergeInspect(&out, r)
	}
	return out
}

// InspectOutputWindow calls plugins on the sliding window (scan-while-streaming).
func InspectOutputWindow(ctx context.Context, inspectors []Inspector, window string) InspectResult {
	var out InspectResult
	for _, p := range inspectors {
		if p == nil {
			continue
		}
		r := safeInspect(func() InspectResult { return p.InspectOutputWindow(ctx, window) })
		if r.EngineError != "" {
			return r
		}
		mergeInspect(&out, r)
	}
	return out
}

type outputRedactor interface {
	OutputRedacts() bool
}

// NeedsOutputHoldback is true when an inspector may redact streaming output.
func NeedsOutputHoldback(inspectors []Inspector) bool {
	for _, p := range inspectors {
		if r, ok := p.(outputRedactor); ok && r.OutputRedacts() {
			return true
		}
	}
	return false
}

func safeInspect(fn func() InspectResult) (out InspectResult) {
	defer func() {
		if rec := recover(); rec != nil {
			out = InspectResult{EngineError: fmt.Sprint(rec)}
		}
	}()
	out = fn()
	return
}
