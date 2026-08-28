package keyword

import (
	"context"
	"testing"
)

func TestHitKeyword(t *testing.T) {
	p, err := New(Rule{ID: "t1", Pattern: `(?i)secret`})
	if err != nil {
		t.Fatal(err)
	}
	r := p.InspectInput(context.Background(), "this has SECRET inside")
	if !r.Hit || r.RuleID != "t1" {
		t.Fatalf("result = %+v", r)
	}
}

func TestMissAllows(t *testing.T) {
	p, err := New(Rule{Pattern: `secret`})
	if err != nil {
		t.Fatal(err)
	}
	r := p.InspectInput(context.Background(), "hello world")
	if r.Hit {
		t.Fatalf("should not hit: %+v", r)
	}
}
