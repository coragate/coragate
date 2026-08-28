package kernel

import "context"

// PolicyEnforce 命中则阻断且不打上游。PolicyObserve 仍检测但不阻断（T6 审计再补）。
const (
	PolicyEnforce = "enforce"
	PolicyObserve = "observe"
)

// InspectResult 是插件对一段文本的判定。匹配逻辑只在插件内。
type InspectResult struct {
	Hit    bool
	RuleID string
}

// Inspector 为数据面检测插件接口（AC-10）。第一期同进程注册，不走 plugin.Open。
type Inspector interface {
	Name() string
	InspectInput(ctx context.Context, text string) InspectResult
	InspectOutputWindow(ctx context.Context, window string) InspectResult
}

// InspectInput 按注册顺序调用插件，返回第一次命中。
func InspectInput(ctx context.Context, inspectors []Inspector, text string) InspectResult {
	for _, p := range inspectors {
		if p == nil {
			continue
		}
		r := p.InspectInput(ctx, text)
		if r.Hit {
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
		r := p.InspectOutputWindow(ctx, window)
		if r.Hit {
			return r
		}
	}
	return InspectResult{}
}
