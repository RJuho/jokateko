package main

import (
	"flag"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLowerFirst(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"A", "a"},
		{"ID", "iD"},
		{"TaskID", "taskID"},
		{"already", "already"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := lowerFirst(tt.in); got != tt.want {
				t.Errorf("lowerFirst(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGoTypeToTS(t *testing.T) {
	tests := []struct {
		goType string
		want   string
	}{
		{"string", "string"},
		{"int", "number"},
		{"int64", "number"},
		{"uint32", "number"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"Priority", "Priority"},
		{"MilestoneStatus", "MilestoneStatus"},
		{"Tier", "Tier"},
		{"Task", "Task"},
		{"[]string", "string[]"},
		{"[][]int", "number[][]"},
		{"[]Task", "Task[]"},
		{"map[string]int", "Record<string, number>"},
		{"map[string][]Task", "Record<string, Task[]>"},
		{"time.Time", "string"},
		{"time.Duration", "unknown"},
		{"json.RawMessage", "unknown"},
		{"any", "any"},
		{"interface{}", "unknown"},
		// Pointers are not unwrapped: *T maps to "unknown".
		{"*string", "unknown"},
		{"*time.Time", "unknown"},
		{"func()", "unknown"},
		{"chan int", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.goType)
			if err != nil {
				t.Fatalf("ParseExpr(%q): %v", tt.goType, err)
			}
			if got := goTypeToTS(expr); got != tt.want {
				t.Errorf("goTypeToTS(%s) = %q, want %q", tt.goType, got, tt.want)
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
	outFile := filepath.Join(t.TempDir(), "nested", "types.ts")

	runMain(t, "-models", modelDir, "-out", outFile)

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	got := string(data)

	wantAlpha := "export interface Alpha {\n" +
		"  name: string;\n" +
		"  noTag: number;\n" +
		"  commaTag?: string;\n" +
		"  count?: number;\n" +
		"  notes?: string[];\n" +
		"  meta: Record<string, string>;\n" +
		"  due?: unknown;\n" +
		"  at: string;\n" +
		"  other: string;\n" +
		"}\n"
	if !strings.Contains(got, wantAlpha) {
		t.Errorf("Alpha interface mismatch, want:\n%s\ngot:\n%s", wantAlpha, got)
	}

	// Structs are emitted alphabetically; unexported structs and non-struct types are skipped.
	iAlpha, iBase, iZeta := strings.Index(got, "interface Alpha"), strings.Index(got, "interface Base"), strings.Index(got, "interface Zeta")
	if iAlpha < 0 || iBase < 0 || iZeta < 0 || !(iAlpha < iBase && iBase < iZeta) {
		t.Errorf("expected Alpha < Base < Zeta ordering, got indices %d, %d, %d", iAlpha, iBase, iZeta)
	}
	for _, absent := range []string{"hidden", "interface Status", "Secret", "secret", "Version"} {
		if strings.Contains(got, absent) {
			t.Errorf("output should not contain %q:\n%s", absent, got)
		}
	}
}
