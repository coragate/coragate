package run

import (
	"context"
	"testing"

	"github.com/coragate/coragate/kernel"
)

func TestCompileKeywordPluginFromSnapshot(t *testing.T) {
	ins, err := compileSnapshot(kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules: []kernel.SnapshotRule{{
			ID:      "t-hello",
			Plugin:  "keyword",
			Pattern: `(?i)hello`,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ins) != 1 {
		t.Fatalf("plugin count = %d", len(ins))
	}
	hit := ins[0].InspectInput(context.Background(), "say HELLO")
	if !hit.Hit || hit.RuleID != "t-hello" {
		t.Fatalf("result = %+v", hit)
	}
}

func TestUnknownPluginRejected(t *testing.T) {
	_, err := compileSnapshot(kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules:         []kernel.SnapshotRule{{ID: "x", Plugin: "onnx", Pattern: "a"}},
	})
	if err == nil {
		t.Fatal("unknown plugin should fail")
	}
}

func TestInvalidRegexLeftToCaller(t *testing.T) {
	_, err := compileSnapshot(kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules:         []kernel.SnapshotRule{{ID: "x", Plugin: "keyword", Pattern: "("}},
	})
	if err == nil {
		t.Fatal("invalid regex should fail")
	}
}
