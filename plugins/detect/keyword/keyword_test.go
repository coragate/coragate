package keyword

import (
	"context"
	"testing"
)

func Test命中关键字(t *testing.T) {
	p, err := New(Rule{ID: "t1", Pattern: `(?i)secret`})
	if err != nil {
		t.Fatal(err)
	}
	r := p.InspectInput(context.Background(), "this has SECRET inside")
	if !r.Hit || r.RuleID != "t1" {
		t.Fatalf("结果 = %+v", r)
	}
}

func Test未命中放行(t *testing.T) {
	p, err := New(Rule{Pattern: `secret`})
	if err != nil {
		t.Fatal(err)
	}
	r := p.InspectInput(context.Background(), "hello world")
	if r.Hit {
		t.Fatalf("不应命中: %+v", r)
	}
}
