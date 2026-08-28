package kernel

import "context"

// EnvelopeSchemaVersion 审计事件规范版本。换存储适配器仍按此 JSON 读。
const EnvelopeSchemaVersion = 1

// RuleNone 表示未命中任何规则（AC-7）。
const RuleNone = "未命中"

// Envelope 是审计事件的规范形状，不是某张 SQL 表（ADR-0012）。
type Envelope struct {
	SchemaVersion int    `json:"schema_version"`
	Time          string `json:"time"`
	DurationMS    int64  `json:"duration_ms"`
	RuleID        string `json:"rule_id"`
	PromptHash    string `json:"prompt_hash"`
	PolicyMode    string `json:"policy_mode"`
	Outcome       string `json:"outcome,omitempty"`
}

// Store 存储适配器。内核只认本接口，默认实现可以是文件或 SQLite。
type Store interface {
	Append(ctx context.Context, env Envelope) error
	List(ctx context.Context, limit int) ([]Envelope, error)
}
