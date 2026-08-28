package kernel

import (
	"context"
	"fmt"
)

// PolicyEnforce 命中则阻断且不打上游。PolicyObserve 仍检测并审计，但不因规则命中而阻断。
const (
	PolicyEnforce = "enforce"
	PolicyObserve = "observe"
)

// 检测引擎不可用时的行为（AC-9 / ADR-0007）。默认 fail_open。
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

// InspectResult 是插件对一段文本的判定。匹配逻辑只在插件内。
// EngineError 非空表示检测引擎不可用，不是规则命中。
type InspectResult struct {
	Hit         bool
	RuleID      string
	EngineError string
}

// Inspector 为数据面检测插件接口（AC-10）。第一期同进程注册，不走 plugin.Open。
type Inspector interface {
	Name() string
	InspectInput(ctx context.Context, text string) InspectResult
	InspectOutputWindow(ctx context.Context, window string) InspectResult
}

// InspectInput 按注册顺序调用插件，返回第一次命中或第一次引擎故障。
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

// InspectOutputWindow 对滑动窗口调用插件（输出边读边扫）。
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
