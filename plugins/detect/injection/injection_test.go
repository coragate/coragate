package injection

import (
	"context"
	"testing"

	"github.com/coragate/coragate/kernel"
)

func TestAC1_夹具表必须命中与不得误拦(t *testing.T) {
	p, err := New(Rule{ID: "inj-1"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		hit  string
		miss string
	}{
		{"F1", FixtureF1Hit, FixtureF1Miss},
		{"F2", FixtureF2Hit, FixtureF2Miss},
		{"F3", FixtureF3Hit, FixtureF3Miss},
		{"F4", FixtureF4Hit, FixtureF4Miss},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := p.InspectInput(context.Background(), "prefix "+tc.hit+" suffix")
			if !got.Hit {
				t.Fatalf("必须命中 %q：%+v", tc.hit, got)
			}
			if got.RuleID != "inj-1" {
				t.Fatalf("rule_id=%s", got.RuleID)
			}
			if len(got.Matches) == 0 || got.Matches[0].EntityType != EntityPromptInjection {
				t.Fatalf("entity_type 应为 prompt_injection：%+v", got)
			}
			if kernel.MatchAction(got.Matches[0]) != kernel.ActionBlock {
				t.Fatalf("默认动作应为 block，得到 %s", got.Matches[0].Action)
			}
			miss := p.InspectInput(context.Background(), tc.miss)
			if miss.Hit {
				t.Fatalf("不得误拦 %q", tc.miss)
			}
		})
	}
}

func TestAC5_缺省动作为block(t *testing.T) {
	p, err := New(Rule{ID: "inj-default"})
	if err != nil {
		t.Fatal(err)
	}
	if p.action != kernel.ActionBlock {
		t.Fatalf("缺 action 应为 block，得到 %s", p.action)
	}
}

func TestAC9_未知type拒绝_空type视为注入(t *testing.T) {
	if _, err := New(Rule{ID: "x", Type: "email"}); err == nil {
		t.Fatal("未知 type 应拒绝")
	}
	p, err := New(Rule{ID: "ok", Type: ""})
	if err != nil {
		t.Fatal(err)
	}
	got := p.InspectInput(context.Background(), FixtureF1Hit)
	if !got.Hit {
		t.Fatal("空 type 仍须匹配夹具")
	}
}

func TestAC4_最长夹具不超过holdback(t *testing.T) {
	if !MustFitWindow() {
		t.Fatalf("最长夹具 %d 字节，hold-back=%d", LongestFixtureBytes(), WindowHoldbackBytes)
	}
}

func TestAC12_正常路径不返回引擎错误(t *testing.T) {
	p, err := New(Rule{ID: "inj-ok"})
	if err != nil {
		t.Fatal(err)
	}
	got := p.InspectInput(context.Background(), FixtureF1Hit)
	if got.EngineError != "" {
		t.Fatalf("夹具命中不应是引擎错误：%s", got.EngineError)
	}
}
