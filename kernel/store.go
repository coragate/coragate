package kernel

import "context"

// EnvelopeSchemaVersion is the audit-event schema. Storage adapters still read this JSON.
const EnvelopeSchemaVersion = 1

// RuleNone means no rule matched (AC-7). The string value is unchanged.
const RuleNone = "未命中"

// Envelope is the canonical audit event, not a SQL table (ADR-0012).
type Envelope struct {
	SchemaVersion int    `json:"schema_version"`
	Time          string `json:"time"`
	DurationMS    int64  `json:"duration_ms"`
	RuleID        string `json:"rule_id"`
	PromptHash    string `json:"prompt_hash"`
	PolicyMode    string `json:"policy_mode"`
	Outcome       string `json:"outcome,omitempty"`
	EngineError   string `json:"engine_error,omitempty"`
}

// Store is the storage adapter. The kernel only knows this interface; the default impl may be a file or SQLite.
type Store interface {
	Append(ctx context.Context, env Envelope) error
	List(ctx context.Context, limit int) ([]Envelope, error)
}
