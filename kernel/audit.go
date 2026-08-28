package kernel

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

func promptHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (c Config) enqueueAudit(body []byte, started time.Time, input, output InspectResult, outcome string) {
	if c.Auditor == nil {
		return
	}
	rule := RuleNone
	if input.Hit {
		rule = input.RuleID
	}
	if output.Hit {
		rule = output.RuleID
	}
	engineErr := input.EngineError
	if output.EngineError != "" {
		engineErr = output.EngineError
	}
	c.Auditor.Enqueue(Envelope{
		SchemaVersion: EnvelopeSchemaVersion,
		Time:          started.UTC().Format(time.RFC3339Nano),
		DurationMS:    time.Since(started).Milliseconds(),
		RuleID:        rule,
		PromptHash:    promptHash(body),
		PolicyMode:    c.policyMode(),
		Outcome:       outcome,
		EngineError:   engineErr,
	})
}
