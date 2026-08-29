package secret

import (
	"context"
	"testing"

	"github.com/coragate/coragate/kernel"
)

func TestAC1_夹具表必须命中与不得误拦(t *testing.T) {
	cases := []struct {
		typ  string
		hit  string
		miss []string
	}{
		{TypeAWSAccessKey, FixtureS1Hit, []string{FixtureS1Miss}},
		{TypeGitHubPAT, FixtureS2Hit, []string{FixtureS2Miss}},
		{TypeOpenAIAPIKey, FixtureS3Hit, []string{FixtureS3MissSkill, FixtureS3MissShort}},
		{TypePEMPrivateKey, FixtureS4Hit, []string{FixtureS4MissPub, FixtureS4MissCert}},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			p, err := New(Rule{ID: "sec-" + tc.typ, Type: tc.typ})
			if err != nil {
				t.Fatal(err)
			}
			got := p.InspectInput(context.Background(), "prefix "+tc.hit+" suffix")
			if !got.Hit {
				t.Fatalf("必须命中 %q：%+v", tc.hit, got)
			}
			if got.Matches[0].EntityType != tc.typ {
				t.Fatalf("entity_type=%s", got.Matches[0].EntityType)
			}
			if kernel.MatchAction(got.Matches[0]) != kernel.ActionBlock {
				t.Fatalf("默认动作应为 block，得到 %s", got.Matches[0].Action)
			}
			for _, miss := range tc.miss {
				if p.InspectInput(context.Background(), miss).Hit {
					t.Fatalf("不得误拦 %q", miss)
				}
			}
		})
	}
}

func TestAC5_缺省动作为block(t *testing.T) {
	p, err := New(Rule{ID: "sec-default", Type: TypeAWSAccessKey})
	if err != nil {
		t.Fatal(err)
	}
	if p.action != kernel.ActionBlock {
		t.Fatalf("缺 action 应为 block，得到 %s", p.action)
	}
}

func TestAC9_未知type拒绝_空type拒绝(t *testing.T) {
	if _, err := New(Rule{ID: "x", Type: "email"}); err == nil {
		t.Fatal("未知 type 应拒绝")
	}
	if _, err := New(Rule{ID: "x", Type: ""}); err == nil {
		t.Fatal("空 type 应拒绝")
	}
}

func TestAC4_最长夹具不超过滑动窗口(t *testing.T) {
	if !MustFitWindow() {
		t.Fatalf("最长夹具 %d 字节，窗口=%d", LongestFixtureBytes(), kernel.DefaultWindowBytes)
	}
}

func TestAC12_正常路径不返回引擎错误(t *testing.T) {
	p, err := New(Rule{ID: "sec-ok", Type: TypeAWSAccessKey})
	if err != nil {
		t.Fatal(err)
	}
	got := p.InspectInput(context.Background(), FixtureS1Hit)
	if got.EngineError != "" {
		t.Fatalf("夹具命中不应是引擎错误：%s", got.EngineError)
	}
}
