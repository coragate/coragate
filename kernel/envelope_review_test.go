package kernel

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestAC8_EnvelopeJSONHasNoPromptOrHitPlaintext(t *testing.T) {
	typ := reflect.TypeOf(Envelope{})
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		switch name {
		case "prompt", "prompt_text", "hit", "hit_text", "plaintext", "span", "spans", "message", "body":
			t.Fatalf("envelope must not serialize hit/prompt plaintext as %q", name)
		}
	}

	b, err := json.Marshal(Envelope{
		SchemaVersion: EnvelopeSchemaVersion,
		RuleID:        "pii-email",
		PromptHash:    "abc",
		PolicyMode:    PolicyEnforce,
		Outcome:       OutcomeRedacted,
		EntityType:    "email",
		RuleAction:    ActionRedact,
		Intervention:  InterventionApplied,
	})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"prompt", "hit_text", "plaintext", "span"} {
		if _, ok := m[bad]; ok {
			t.Fatalf("marshaled envelope has forbidden field %q: %s", bad, b)
		}
	}
	if _, ok := m["prompt_hash"]; !ok {
		t.Fatal("prompt_hash must remain")
	}
}
