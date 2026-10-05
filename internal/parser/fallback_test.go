package parser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/parser"
)

// assertRecentRFC3339 checks that s is an RFC3339 UTC timestamp within [before, after].
func assertRecentRFC3339(t *testing.T, s string, before, after time.Time) {
	t.Helper()
	got, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("not RFC3339: %q (%v)", s, err)
	}
	if !strings.HasSuffix(s, "Z") {
		t.Errorf("expected UTC timestamp, got %q", s)
	}
	if got.Before(before.Truncate(time.Second)) || got.After(after) {
		t.Errorf("timestamp %q not within [%s, %s]", s, before, after)
	}
}

func TestDeriveFallbackChangedAt(t *testing.T) {
	helsinki := time.FixedZone("EEST", 3*60*60)
	mtime := time.Date(2026, 9, 10, 15, 4, 5, 999, helsinki)

	tests := []struct {
		name      string
		modTime   time.Time
		createdAt string
		want      string // empty means "now"
	}{
		{"mtime set wins over createdAt", mtime, "2026-01-01T00:00:00Z", "2026-09-10T12:04:05Z"},
		{"mtime set without createdAt", mtime, "", "2026-09-10T12:04:05Z"},
		{"createdAt only", time.Time{}, "2026-01-01T00:00:00Z", "2026-01-01T00:00:00Z"},
		{"createdAt returned verbatim", time.Time{}, "not-normalized", "not-normalized"},
		{"neither falls back to now", time.Time{}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UTC()
			got := parser.DeriveFallbackChangedAt(tt.modTime, tt.createdAt)
			after := time.Now().UTC()
			if tt.want == "" {
				assertRecentRFC3339(t, got, before, after)
				return
			}
			if got != tt.want {
				t.Errorf("DeriveFallbackChangedAt = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeriveFallbackCreatedAt(t *testing.T) {
	mtime := time.Date(2026, 3, 4, 5, 6, 7, 0, time.FixedZone("X", -2*60*60))

	tests := []struct {
		name    string
		id      string
		modTime time.Time
		want    string // empty means "now"
	}{
		{"date prefix wins over mtime", "260901-setup", mtime, "2026-09-01T00:00:00Z"},
		{"no date prefix uses mtime", "setup-database", mtime, "2026-03-04T07:06:07Z"},
		{"invalid date prefix uses mtime", "261399-bad-month", mtime, "2026-03-04T07:06:07Z"},
		{"short id uses mtime", "abc", mtime, "2026-03-04T07:06:07Z"},
		{"no date prefix and no mtime uses now", "setup-database", time.Time{}, ""},
		{"empty id and no mtime uses now", "", time.Time{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now().UTC()
			got := parser.DeriveFallbackCreatedAt(tt.id, tt.modTime)
			after := time.Now().UTC()
			if tt.want == "" {
				assertRecentRFC3339(t, got, before, after)
				return
			}
			if got != tt.want {
				t.Errorf("DeriveFallbackCreatedAt(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestParseEntityValidationErrors(t *testing.T) {
	doc := func(fm string) []byte { return []byte("+++\n" + fm + "\n+++\n\nbody\n") }

	type parseFn func([]byte, string) error
	milestone := func(c []byte, id string) error { _, err := parser.ParseMilestone(c, id); return err }
	strategy := func(c []byte, id string) error { _, err := parser.ParseStrategy(c, id); return err }
	glossary := func(c []byte, id string) error { _, err := parser.ParseGlossaryTerm(c, id); return err }

	tests := []struct {
		name    string
		parse   parseFn
		content []byte
		wantErr string
	}{
		{"milestone no frontmatter", milestone, []byte("just text"), ""},
		{"milestone bad toml", milestone, doc(`title = `), "failed to parse TOML"},
		{"milestone missing title", milestone, doc(`summary = "s"`), "missing required field: title"},
		{"milestone missing summary", milestone, doc(`title = "t"`), "missing required field: summary"},
		{"strategy no frontmatter", strategy, []byte("just text"), ""},
		{"strategy missing title", strategy, doc(`summary = "s"` + "\ntier = 1"), "missing required field: title"},
		{"strategy missing summary", strategy, doc(`title = "t"` + "\ntier = 1"), "missing required field: summary"},
		{"strategy invalid tier", strategy, doc("title = \"t\"\nsummary = \"s\"\ntier = 9"), "invalid tier 9"},
		{"glossary no frontmatter", glossary, []byte("just text"), ""},
		{"glossary missing title", glossary, doc(`summary = "s"`), "missing required field: title"},
		{"glossary missing summary", glossary, doc(`title = "t"`), "missing required field: summary"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse(tt.content, "id")
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestFormatMarshalError(t *testing.T) {
	// Channels cannot be encoded as TOML.
	if _, err := parser.Format(map[string]any{"c": make(chan int)}, "body"); err == nil {
		t.Error("expected marshal error for unsupported type")
	}
}
