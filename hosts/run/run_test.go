package run

import (
	"context"
	"testing"

	"github.com/coragate/coragate/kernel"
)

func Test从快照编译关键字插件(t *testing.T) {
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
		t.Fatalf("插件数 = %d", len(ins))
	}
	hit := ins[0].InspectInput(context.Background(), "say HELLO")
	if !hit.Hit || hit.RuleID != "t-hello" {
		t.Fatalf("结果 = %+v", hit)
	}
}

func Test未知插件拒绝编译(t *testing.T) {
	_, err := compileSnapshot(kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules:         []kernel.SnapshotRule{{ID: "x", Plugin: "onnx", Pattern: "a"}},
	})
	if err == nil {
		t.Fatal("未知插件应失败")
	}
}

func Test非法正则保留调用方处理(t *testing.T) {
	_, err := compileSnapshot(kernel.RuleSnapshot{
		SchemaVersion: kernel.RulesSchemaVersion,
		Rules:         []kernel.SnapshotRule{{ID: "x", Plugin: "keyword", Pattern: "("}},
	})
	if err == nil {
		t.Fatal("非法正则应失败")
	}
}
