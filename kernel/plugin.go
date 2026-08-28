package kernel

import (
	"context"
	"fmt"
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
	OutcomeFailOpen   = "fail_open"
	OutcomeFailClosed = "fail_closed"
)

// InspectResult is a plugin verdict on a text span. Matching logic stays inside the plugin.
// A non-empty EngineError means the engine is down, not a rule hit.
type InspectResult struct {
	Hit         bool
	RuleID      string
	EngineError string
}

// Inspector is the dataplane detection plugin interface (AC-10). Phase 1 registers in-process; no plugin.Open.
type Inspector interface {
	Name() string
	InspectInput(ctx context.Context, text string) InspectResult
	InspectOutputWindow(ctx context.Context, window string) InspectResult
}

// InspectInput calls plugins in order and returns the first hit or the first engine failure.
func InspectInput(ctx context.Context, inspectors []Inspector, text string) InspectResult {
	for _, p := range inspectors {
		if p == nil {
			continue
		}
		r := safeInspect(func() InspectResult { return p.InspectInput(ctx, text) })
		if r.EngineError != "" || r.Hit {
			return r
		}
	}
	return InspectResult{}
}

// InspectOutputWindow calls plugins on the sliding window (scan-while-streaming).
func InspectOutputWindow(ctx context.Context, inspectors []Inspector, window string) InspectResult {
	for _, p := range inspectors {
		if p == nil {
			continue
		}
		r := safeInspect(func() InspectResult { return p.InspectOutputWindow(ctx, window) })
		if r.EngineError != "" || r.Hit {
			return r
		}
	}
	return InspectResult{}
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
