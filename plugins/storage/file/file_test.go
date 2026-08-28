package file

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/coragate/coragate/kernel"
)

func TestAC14_FileAdapterRoundTripsSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	want := kernel.Envelope{
		SchemaVersion: kernel.EnvelopeSchemaVersion,
		Time:          time.Now().UTC().Format(time.RFC3339Nano),
		DurationMS:    3,
		RuleID:        kernel.RuleNone,
		PromptHash:    "abc",
		PolicyMode:    kernel.PolicyEnforce,
		Outcome:       "forwarded",
	}
	if err := st.Append(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := st.List(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SchemaVersion != kernel.EnvelopeSchemaVersion || got[0].PromptHash != "abc" {
		t.Fatalf("List = %+v", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var env kernel.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("adapter should still unmarshal JSON schema: %v raw=%s", err, raw)
	}
	if env.SchemaVersion != kernel.EnvelopeSchemaVersion {
		t.Fatalf("schema_version=%d", env.SchemaVersion)
	}
}

func TestListReturnsLatestN(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.jsonl")
	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range []string{"old", "mid", "new"} {
		if err := st.Append(context.Background(), kernel.Envelope{
			SchemaVersion: kernel.EnvelopeSchemaVersion,
			Time:          time.Now().UTC().Format(time.RFC3339Nano),
			RuleID:        id,
			PromptHash:    "h",
			PolicyMode:    kernel.PolicyEnforce,
			DurationMS:    int64(i),
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := st.List(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].RuleID != "mid" || got[1].RuleID != "new" {
		t.Fatalf("want latest two: %+v", got)
	}
}
