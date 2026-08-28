package pii

import (
	"context"
	"strings"
	"testing"

	"github.com/coragate/coragate/kernel"
)

func TestAC1_Fixtures(t *testing.T) {
	cases := []struct {
		typ    string
		hit    string
		misses []string
	}{
		{TypeEmail, "alice@example.com", []string{"alice at example"}},
		{TypePhoneCN, "13812345678", []string{"12345", "138123456789"}},
		{TypeIDCardCN, "440524188001010014", []string{"123456"}},
		{TypeBankCard, "4111111111111111", []string{"411111111111"}},
	}
	for _, tc := range cases {
		p, err := New(Rule{ID: "t-" + tc.typ, Type: tc.typ})
		if err != nil {
			t.Fatal(err)
		}
		got := p.InspectInput(context.Background(), "prefix "+tc.hit+" suffix")
		if !got.Hit {
			t.Fatalf("%s must hit %q: %+v", tc.typ, tc.hit, got)
		}
		if got.Matches[0].EntityType != tc.typ {
			t.Fatalf("%s entity_type=%s", tc.typ, got.Matches[0].EntityType)
		}
		if kernel.MatchAction(got.Matches[0]) != kernel.ActionRedact {
			t.Fatalf("%s default action=%s", tc.typ, got.Matches[0].Action)
		}
		for _, miss := range tc.misses {
			r := p.InspectInput(context.Background(), miss)
			if r.Hit {
				t.Fatalf("%s must not hit %q", tc.typ, miss)
			}
		}
	}
}

func TestAC1_RedactAllEntitiesInOneString(t *testing.T) {
	text := "mail alice@example.com phone 13812345678 id 440524188001010014 card 4111111111111111"
	var inspectors []kernel.Inspector
	for _, typ := range []string{TypeEmail, TypePhoneCN, TypeIDCardCN, TypeBankCard} {
		p, err := New(Rule{ID: "pii-" + typ, Type: typ})
		if err != nil {
			t.Fatal(err)
		}
		inspectors = append(inspectors, p)
	}
	r := kernel.InspectInput(context.Background(), inspectors, text)
	out := kernel.ApplyRedact(text, r)
	for _, raw := range []string{"alice@example.com", "13812345678", "440524188001010014", "4111111111111111"} {
		if strings.Contains(out, raw) {
			t.Fatalf("plaintext still present %q in %q", raw, out)
		}
	}
	for _, ph := range []string{"[REDACTED:email]", "[REDACTED:phone_cn]", "[REDACTED:id_card_cn]", "[REDACTED:bank_card]"} {
		if !strings.Contains(out, ph) {
			t.Fatalf("missing placeholder %s in %q", ph, out)
		}
	}
}

func TestIDChecksumFixture(t *testing.T) {
	if !idChecksumOK("440524188001010014") {
		t.Fatal("fixture id card checksum should be valid")
	}
}
