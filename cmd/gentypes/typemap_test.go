package main

import (
	"bytes"
	"flag"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoTypeToTS(t *testing.T) {
	tests := []struct {
		goType string
		want   string
		warn   bool
	}{
		{"string", "string", false},
		{"int", "number", false},
		{"int64", "number", false},
		{"uint32", "number", false},
		{"uint8", "number", false},
		{"float64", "number", false},
		{"bool", "boolean", false},
		{"Priority", "Priority", false},
		{"MilestoneStatus", "MilestoneStatus", false},
		{"Tier", "Tier", false},
		{"Task", "Task", false},
		{"[]string", "string[]", false},
		{"[][]int", "number[][]", false},
		{"[]Task", "Task[]", false},
		{"[]byte", "string", false},
		{"map[string]int", "Record<string, number>", false},
		{"map[string][]Task", "Record<string, Task[]>", false},
		{"time.Time", "string", false},
		{"any", "any", false},
		{"interface{}", "unknown", false},
		// Pointers are nullable: a nil pointer encodes as null.
		{"*string", "string | null", false},
		{"**int", "number | null", false},
		{"*time.Time", "string | null", false},
		{"*Task", "Task | null", false},
		{"[]*int", "(number | null)[]", false},
		{"map[string]*Task", "Record<string, Task | null>", false},
		// Unsupported types are written as unknown with a warning.
		{"time.Duration", "unknown", true},
		{"json.RawMessage", "unknown", true},
		{"*json.RawMessage", "unknown | null", true},
		{"G[int]", "unknown", true},
		{"G[int, string]", "unknown", true},
		{"func()", "unknown", true},
		{"chan int", "unknown", true},
		{"struct{}", "unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.goType)
			if err != nil {
				t.Fatalf("ParseExpr(%q): %v", tt.goType, err)
			}
			var warnings []string
			if got := goTypeToTS(expr, func(msg string) { warnings = append(warnings, msg) }); got != tt.want {
				t.Errorf("goTypeToTS(%s) = %q, want %q", tt.goType, got, tt.want)
			}
			if got := len(warnings) > 0; got != tt.warn {
				t.Errorf("goTypeToTS(%s) warned = %v (%q), want %v", tt.goType, got, warnings, tt.warn)
			}
		})
	}
}

// runMain invokes main() with a fresh flag set so it can be called repeatedly.
func runMain(t *testing.T, args ...string) {
	t.Helper()
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() { os.Args, flag.CommandLine = oldArgs, oldFlags })
	os.Args = append([]string{"gentypes"}, args...)
	flag.CommandLine = flag.NewFlagSet("gentypes", flag.ExitOnError)
	main()
}

func TestGentypesFieldRules(t *testing.T) {
	modelDir := t.TempDir()
	src := "package model\n\n" +
		"import \"time\"\n\n" +
		"const Version = 1\n\n" +
		"type Status string\n\n" +
		"type hidden struct{ X int }\n\n" +
		"type Base struct{ ID string `json:\"id\"` }\n\n" +
		"type Zeta struct{ Z bool `json:\"z\"` }\n\n" +
		"type Alpha struct {\n" +
		"\tBase\n" +
		"\tName     string            `json:\"name\"`\n" +
		"\tSecret   string            `json:\"-\"`\n" +
		"\tNoTag    int\n" +
		"\tCommaTag string            `json:\",omitempty\"`\n" +
		"\tCount    int               `json:\"count,omitzero\"`\n" +
		"\tNotes    []string          `json:\"notes,omitempty\"`\n" +
		"\tMeta     map[string]string `json:\"meta\"`\n" +
		"\tDue      *time.Time        `json:\"due,omitempty\"`\n" +
		"\tAt       time.Time         `json:\"at\"`\n" +
		"\tOther    string            `yaml:\"other\"`\n" +
		"}\n"
	if err := os.WriteFile(filepath.Join(modelDir, "model.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	// Test files are not part of the model.
	testSrc := "package model\n\ntype FromTest struct{ X int }\n"
	if err := os.WriteFile(filepath.Join(modelDir, "model_test.go"), []byte(testSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(t.TempDir(), "nested", "types.ts")

	runMain(t, "-models", modelDir, "-out", outFile)

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(data)

	wantAlpha := "export interface Alpha {\n" +
		"  id: string;\n" +
		"  name: string;\n" +
		"  NoTag: number;\n" +
		"  CommaTag?: string;\n" +
		"  count?: number;\n" +
		"  notes?: string[];\n" +
		"  meta: Record<string, string>;\n" +
		"  due?: string;\n" +
		"  at: string;\n" +
		"  Other: string;\n" +
		"}\n"
	if !strings.Contains(got, wantAlpha) {
		t.Errorf("Alpha interface mismatch, want:\n%s\ngot:\n%s", wantAlpha, got)
	}

	// Structs are emitted alphabetically; unexported structs and non-struct types are skipped.
	iAlpha, iBase, iZeta := strings.Index(got, "interface Alpha"), strings.Index(got, "interface Base"), strings.Index(got, "interface Zeta")
	if iAlpha < 0 || iBase < 0 || iZeta < 0 || !(iAlpha < iBase && iBase < iZeta) {
		t.Errorf("expected Alpha < Base < Zeta ordering, got indices %d, %d, %d", iAlpha, iBase, iZeta)
	}
	for _, absent := range []string{"hidden", "interface Status", "Secret", "secret", "Version", "FromTest"} {
		if strings.Contains(got, absent) {
			t.Errorf("output should not contain %q:\n%s", absent, got)
		}
	}
}

func TestGenerateFields(t *testing.T) {
	tests := []struct {
		name     string
		decls    string // declarations added to package model
		iface    string // interface to check
		want     string // expected interface body
		warnings []string
	}{
		{
			name:  "pointer without omitempty is nullable",
			decls: "type A struct {\n\tP *string `json:\"p\"`\n\tQ *int\n}",
			iface: "A", want: "  p: string | null;\n  Q: number | null;\n",
		},
		{
			name:  "pointer with omitempty or omitzero is optional",
			decls: "type A struct {\n\tP *string `json:\"p,omitempty\"`\n\tQ *int `json:\"q,omitzero\"`\n}",
			iface: "A", want: "  p?: string;\n  q?: number;\n",
		},
		{
			name:  "pointer to time.Time",
			decls: "type A struct {\n\tT *time.Time `json:\"t\"`\n\tU *time.Time `json:\"u,omitempty\"`\n}",
			iface: "A", want: "  t: string | null;\n  u?: string;\n",
		},
		{
			name:  "pointer to model struct",
			decls: "type B struct{ X int `json:\"x\"` }\ntype A struct {\n\tB *B `json:\"b\"`\n}",
			iface: "A", want: "  b: B | null;\n",
		},
		{
			name:  "multiple names and unexported fields",
			decls: "type A struct {\n\tX, Y int\n\tz, w string\n}",
			iface: "A", want: "  X: number;\n  Y: number;\n",
		},
		{
			name:  "embedded struct is flattened in place",
			decls: "type B struct {\n\tX int `json:\"x\"`\n\tY string `json:\"y,omitempty\"`\n}\ntype A struct {\n\tFirst string `json:\"first\"`\n\tB\n\tLast string `json:\"last\"`\n}",
			iface: "A", want: "  first: string;\n  x: number;\n  y?: string;\n  last: string;\n",
		},
		{
			name:  "embedded pointer struct makes promoted fields optional",
			decls: "type B struct {\n\tX int `json:\"x\"`\n\tP *int `json:\"p\"`\n}\ntype A struct {\n\t*B\n}",
			iface: "A", want: "  x?: number;\n  p?: number | null;\n",
		},
		{
			name:  "embedded unexported struct is promoted",
			decls: "type b struct{ X int `json:\"x\"` }\ntype c struct{ Y int `json:\"y\"` }\ntype A struct {\n\tb\n\t*c\n}",
			iface: "A", want: "  x: number;\n  y?: number;\n",
		},
		{
			name:  "nested embedding",
			decls: "type C struct{ Z int `json:\"z\"` }\ntype B struct {\n\tC\n\tY int `json:\"y\"`\n}\ntype A struct {\n\t*B\n}",
			iface: "A", want: "  z?: number;\n  y?: number;\n",
		},
		{
			name:  "tagged embedded struct is a named field",
			decls: "type B struct{ X int `json:\"x\"` }\ntype A struct {\n\tB `json:\"b\"`\n\t*B2 `json:\"b2,omitempty\"`\n}\ntype B2 struct{ Y int `json:\"y\"` }",
			iface: "A", want: "  b: B;\n  b2?: B2;\n",
		},
		{
			name:  "tagged embedded struct without a name is still flattened",
			decls: "type B struct{ X int `json:\"x\"` }\ntype A struct {\n\tB `json:\",omitempty\"`\n}",
			iface: "A", want: "  x: number;\n",
		},
		{
			name:  "embedded non-struct types",
			decls: "type Status string\ntype level int\ntype A struct {\n\tStatus\n\t*Code\n\tlevel\n}\ntype Code int",
			iface: "A", want: "  Status: Status;\n  Code: Code | null;\n",
		},
		{
			name:  "shallower field shadows promoted field",
			decls: "type B struct {\n\tX int `json:\"x\"`\n\tY int `json:\"y\"`\n}\ntype A struct {\n\tB\n\tX string `json:\"x\"`\n}",
			iface: "A", want: "  y: number;\n  x: string;\n",
		},
		{
			name:  "tagged field wins at the same depth",
			decls: "type B struct{ X int `json:\"X\"` }\ntype C struct{ X string }\ntype A struct {\n\tC\n\tB\n}",
			iface: "A", want: "  X: number;\n",
		},
		{
			name:     "ambiguous fields at the same depth are dropped",
			decls:    "type B struct{ X int `json:\"x\"`; K int `json:\"k\"` }\ntype C struct{ X string `json:\"x\"` }\ntype A struct {\n\tB\n\tC\n}",
			iface:    "A",
			want:     "  k: number;\n",
			warnings: []string{"A.x: ambiguous field name at the same embedding depth; encoding/json drops it"},
		},
		{
			name:  "recursive embedding stops",
			decls: "type A struct {\n\t*A\n\t*B\n\tV int `json:\"v\"`\n}\ntype B struct {\n\t*A\n\tW int `json:\"w\"`\n}",
			iface: "A", want: "  w?: number;\n  v: number;\n",
		},
		{
			name:     "embedded type from another package",
			decls:    "type A struct {\n\ttime.Time\n\t*json.Decoder\n\tX int `json:\"x\"`\n\tT time.Location `json:\"t\"`\n}",
			iface:    "A",
			want:     "  x: number;\n  t: unknown;\n",
			warnings: []string{"A.time.Time: embedded type from another package is not supported; its fields are skipped", "A.json.Decoder: embedded type from another package is not supported; its fields are skipped", "A.T: type time.Location from another package is not supported; written as unknown"},
		},
		{
			name:     "generic types",
			decls:    "type G[T any] struct{ V T `json:\"v\"` }\ntype A struct {\n\tG[int]\n\tH G[string] `json:\"h\"`\n}",
			iface:    "A",
			want:     "  h: unknown;\n",
			warnings: []string{"G: generic struct types are not supported; skipped", "A.G: embedded generic type is not supported; its fields are skipped", "A.H: generic type G is not supported; written as unknown"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			src := "package model\n\nimport (\n\t\"encoding/json\"\n\t\"time\"\n)\n\nvar _ json.Decoder\nvar _ time.Time\n\n" + tt.decls + "\n"
			if err := os.WriteFile(filepath.Join(dir, "model.go"), []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
			var warn bytes.Buffer
			out, err := generate(dir, &warn)
			if err != nil {
				t.Fatal(err)
			}
			want := "export interface " + tt.iface + " {\n" + tt.want + "}\n"
			if !strings.Contains(string(out), want) {
				t.Errorf("want:\n%s\ngot:\n%s", want, out)
			}
			var wantWarn string
			for _, w := range tt.warnings {
				wantWarn += "gentypes: warning: " + w + "\n"
			}
			if warn.String() != wantWarn {
				t.Errorf("warnings:\n%s\nwant:\n%s", warn.String(), wantWarn)
			}
		})
	}
}

func TestGenerateErrors(t *testing.T) {
	if _, err := generate(filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Error("expected error for missing directory")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte("package other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := generate(dir, nil); err == nil || !strings.Contains(err.Error(), "package 'model' not found") {
		t.Errorf("expected package-not-found error, got %v", err)
	}
}
