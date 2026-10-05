package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	model "github.com/RJuho/jokateko/cmd/gentypes/internal/samplemodel"
)

var update = flag.Bool("update", false, "rewrite testdata/samplemodel.ts.golden")

const sampleDir = "internal/samplemodel"

// TestGoldenSampleModel compares the generated TypeScript for the sample model with
// testdata/samplemodel.ts.golden. Run `go test ./cmd/gentypes -update` to refresh it.
func TestGoldenSampleModel(t *testing.T) {
	var warn bytes.Buffer
	got, err := generate(sampleDir, &warn)
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "samplemodel.ts.golden")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run with -update to create it): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("generated output differs from %s:\n%s", golden, got)
	}

	wantWarn := "gentypes: warning: Conflict.Shared: ambiguous field name at the same embedding depth; encoding/json drops it\n"
	if warn.String() != wantWarn {
		t.Errorf("warnings = %q, want %q", warn.String(), wantWarn)
	}
}

// tsProp is one property of a generated TypeScript interface.
type tsProp struct {
	optional bool
	nullable bool
	tsType   string // without the "| null" suffix
}

var (
	interfaceRe = regexp.MustCompile(`(?ms)^export interface (\w+) \{\n(.*?)^\}`)
	propRe      = regexp.MustCompile(`^  (\w+)(\??): (.+);$`)
)

// parseInterfaces extracts the properties of every interface in src.
func parseInterfaces(t *testing.T, src string) map[string]map[string]tsProp {
	t.Helper()
	out := make(map[string]map[string]tsProp)
	for _, m := range interfaceRe.FindAllStringSubmatch(src, -1) {
		props := make(map[string]tsProp)
		for line := range strings.Lines(m[2]) {
			p := propRe.FindStringSubmatch(strings.TrimSuffix(line, "\n"))
			if p == nil {
				t.Fatalf("unparsable line in interface %s: %q", m[1], line)
			}
			typ, nullable := strings.CutSuffix(p[3], " | null")
			props[p[1]] = tsProp{optional: p[2] == "?", nullable: nullable, tsType: typ}
		}
		out[m[1]] = props
	}
	return out
}

// kindMatches reports whether the decoded JSON value v fits the TypeScript type.
func kindMatches(tsType string, v any, interfaces map[string]map[string]tsProp) bool {
	switch {
	case tsType == "string":
		_, ok := v.(string)
		return ok
	case tsType == "number":
		_, ok := v.(float64)
		return ok
	case tsType == "boolean":
		_, ok := v.(bool)
		return ok
	case strings.HasSuffix(tsType, "[]"):
		_, ok := v.([]any)
		return ok
	case strings.HasPrefix(tsType, "Record<"):
		_, ok := v.(map[string]any)
		return ok
	case interfaces[tsType] != nil:
		_, ok := v.(map[string]any)
		return ok
	default:
		return true // named non-struct types such as Status are not checked
	}
}

// TestGoldenMatchesJSON checks the generated interfaces against json.Marshal of
// zero and fully populated sample values.
func TestGoldenMatchesJSON(t *testing.T) {
	src, err := generate(sampleDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	interfaces := parseInterfaces(t, string(src))

	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	full := &model.Sample{
		ID:       "s1",
		Hidden:   "h",
		Audit:    &model.Audit{CreatedBy: "me", ReviewedAt: &now},
		Meta:     model.Meta{Label: "l", Score: 3},
		Status:   "open",
		Title:    "t",
		Untagged: 1,
		A:        true,
		B:        true,
		Note:     new("n"),
		Due:      &now,
		Count:    new(2),
		IDs:      []*int{new(1), nil},
		Index:    map[string]*model.Meta{"a": {Label: "x"}, "b": nil},
		Raw:      []byte("raw"),
	}

	cases := []struct {
		name       string
		zero, full any
	}{
		{"Sample", model.Sample{}, full},
		{"Audit", model.Audit{}, model.Audit{CreatedBy: "me", ReviewedAt: &now}},
		{"Meta", model.Meta{}, model.Meta{Label: "l", Score: 1}},
		{"Shadow", model.Shadow{}, model.Shadow{Meta: model.Meta{Label: "inner", Score: 1}, Label: 2}},
		{"Conflict", model.Conflict{}, model.Conflict{Left: model.Left{Shared: "l", Name: "l"}, Right: model.Right{Shared: "r", Name: "r"}, ID: "c"}},
		{"Left", model.Left{}, model.Left{Shared: "s", Name: "n"}},
		{"Right", model.Right{}, model.Right{Shared: "s", Name: "n"}},
		{"SelfRef", model.SelfRef{}, model.SelfRef{Value: "v"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			props := interfaces[tc.name]
			if props == nil {
				t.Fatalf("interface %s not generated", tc.name)
			}
			zero, full := marshalMap(t, tc.zero), marshalMap(t, tc.full)

			for _, obj := range []map[string]any{zero, full} {
				for key, v := range obj {
					p, ok := props[key]
					switch {
					case !ok:
						t.Errorf("JSON key %q is missing from interface %s", key, tc.name)
					case v == nil:
						// Nil slices and maps encode as null too; gentypes keeps
						// them non-nullable and the backend normalises them.
						isCollection := strings.HasSuffix(p.tsType, "[]") || strings.HasPrefix(p.tsType, "Record<") || key == "raw"
						if !p.nullable && !isCollection {
							t.Errorf("%s.%s is null in JSON but not nullable in TS", tc.name, key)
						}
					case !kindMatches(p.tsType, v, interfaces):
						t.Errorf("%s.%s = %#v does not match TS type %s", tc.name, key, v, p.tsType)
					}
				}
			}
			for key, p := range props {
				if _, ok := zero[key]; !ok && !p.optional {
					t.Errorf("%s.%s is required in TS but missing from zero-value JSON", tc.name, key)
				}
				if _, ok := full[key]; !ok {
					t.Errorf("%s.%s is missing from populated JSON", tc.name, key)
				}
			}
		})
	}
}

func marshalMap(t *testing.T, v any) map[string]any {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal %T: %v", v, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	return m
}
