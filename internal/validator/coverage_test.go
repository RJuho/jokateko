package validator_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/validator"
)

func ruleIDs(diags []validator.Diagnostic) []string {
	ids := make([]string, 0, len(diags))
	for _, d := range diags {
		ids = append(ids, d.RuleID)
	}
	return ids
}

func TestDiagnosticString(t *testing.T) {
	tests := []struct {
		name string
		d    validator.Diagnostic
		want string
	}{
		{
			name: "file only",
			d:    validator.Diagnostic{RuleID: "TSK-001", Severity: validator.SeverityWarning, File: "a.md", Message: "msg"},
			want: "WARNING [TSK-001] a.md\n  msg",
		},
		{
			name: "with line",
			d:    validator.Diagnostic{RuleID: "TSK-002", Severity: validator.SeverityError, File: "a.md", Line: 7, Message: "bad"},
			want: "ERROR [TSK-002] a.md:7\n  bad",
		},
		{
			name: "context and fix",
			d: validator.Diagnostic{
				RuleID: "DAG-001", Severity: validator.SeverityError, File: "b.md", Message: "cycle",
				Context: []string{"a -> b", "b -> a"}, Fix: "remove one",
			},
			want: "ERROR [DAG-001] b.md\n  cycle\n  a -> b\n  b -> a\n  Fix: remove one",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.d.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidationResultCountsAndReport(t *testing.T) {
	warn := func(id string) validator.Diagnostic {
		return validator.Diagnostic{RuleID: id, Severity: validator.SeverityWarning, File: id + ".md", Message: "w " + id}
	}
	errD := func(id string) validator.Diagnostic {
		return validator.Diagnostic{RuleID: id, Severity: validator.SeverityError, File: id + ".md", Message: "e " + id}
	}

	tests := []struct {
		name         string
		diags        []validator.Diagnostic
		wantErrs     int
		wantWarns    int
		wantContains []string
		wantOrder    []string // substrings that must appear in this order
	}{
		{
			name:         "empty",
			wantContains: []string{"[OK] Validation successful:", "Configuration: cfg.toml (3 columns defined)", "Tasks: 4 files parsed", "Milestones: 5 files", "Strategies: 6 guidelines", "Glossary: 7 terms", "Project is healthy. Exiting with code 0."},
		},
		{
			name:         "single warning",
			diags:        []validator.Diagnostic{warn("W1")},
			wantWarns:    1,
			wantContains: []string{"[WARNING] Found 1 warning during validation:", "Project passed with warnings. Exiting with code 0."},
		},
		{
			name:         "warnings only",
			diags:        []validator.Diagnostic{warn("W1"), warn("W2")},
			wantWarns:    2,
			wantContains: []string{"[WARNING] Found 2 warnings during validation:"},
			wantOrder:    []string{"[W1]", "[W2]"},
		},
		{
			name:         "single error no warnings",
			diags:        []validator.Diagnostic{errD("E1")},
			wantErrs:     1,
			wantContains: []string{"[FAIL] Found 1 error and 0 warnings during validation:", "Validation failed with 1 error. Exiting with code 1."},
		},
		{
			name:         "mixed keeps diagnostic order",
			diags:        []validator.Diagnostic{warn("W1"), errD("E1"), errD("E2"), warn("W2")},
			wantErrs:     2,
			wantWarns:    2,
			wantContains: []string{"[FAIL] Found 2 errors and 2 warnings during validation:", "Validation failed with 2 errors."},
			wantOrder:    []string{"[W1]", "[E1]", "[E2]", "[W2]", "Validation failed"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := &validator.ValidationResult{
				Diagnostics: tc.diags, ConfigFile: "cfg.toml", ColumnCount: 3,
				TaskCount: 4, MilestoneCount: 5, StrategyCount: 6, GlossaryCount: 7,
			}
			if got := r.ErrorCount(); got != tc.wantErrs {
				t.Errorf("ErrorCount() = %d, want %d", got, tc.wantErrs)
			}
			if got := r.WarningCount(); got != tc.wantWarns {
				t.Errorf("WarningCount() = %d, want %d", got, tc.wantWarns)
			}
			if got := r.HasErrors(); got != (tc.wantErrs > 0) {
				t.Errorf("HasErrors() = %v, want %v", got, tc.wantErrs > 0)
			}
			report := r.FormatReport()
			for _, s := range tc.wantContains {
				if !strings.Contains(report, s) {
					t.Errorf("report missing %q:\n%s", s, report)
				}
			}
			pos := -1
			for _, s := range tc.wantOrder {
				idx := strings.Index(report, s)
				if idx <= pos {
					t.Errorf("expected %q after previous entries (idx %d, prev %d):\n%s", s, idx, pos, report)
				}
				pos = idx
			}
		})
	}
}

func enforcingConfig() (*config.Config, map[string]bool) {
	cfg := config.Default(".")
	cfg.Tags = config.TagsConfig{Allowed: []string{"ok"}, EnforceAllowed: true}
	return cfg, map[string]bool{"ok": true}
}

func TestValidateMilestoneRules(t *testing.T) {
	cfg, allowed := enforcingConfig()
	tests := []struct {
		name      string
		file      string
		content   string
		cfg       *config.Config
		wantRules []string
		wantNil   bool
	}{
		{
			name:    "valid",
			file:    "260101-ms.md",
			content: "+++\ntitle = \"M\"\nsummary = \"S\"\ntarget_date = \"2026-01-31\"\ntags = [\"ok\"]\n+++\n",
			cfg:     cfg,
		},
		{
			name:      "bad filename",
			file:      "Milestone.md",
			content:   "+++\ntitle = \"M\"\nsummary = \"S\"\n+++\n",
			cfg:       cfg,
			wantRules: []string{"MLS-001"},
		},
		{
			name:      "no frontmatter",
			file:      "260101-ms.md",
			content:   "# just markdown\n",
			cfg:       cfg,
			wantRules: []string{"MLS-002"},
			wantNil:   true,
		},
		{
			name:      "missing summary surfaces as frontmatter error",
			file:      "260101-ms.md",
			content:   "+++\ntitle = \"M\"\n+++\n",
			cfg:       cfg,
			wantRules: []string{"MLS-002"},
			wantNil:   true,
		},
		{
			name:      "bad target date",
			file:      "260101-ms.md",
			content:   "+++\ntitle = \"M\"\nsummary = \"S\"\ntarget_date = \"31.1.2026\"\n+++\n",
			cfg:       cfg,
			wantRules: []string{"MLS-004"},
		},
		{
			name:      "disallowed tags",
			file:      "260101-ms.md",
			content:   "+++\ntitle = \"M\"\nsummary = \"S\"\ntags = [\"ok\", \"nope\", \"also-nope\"]\n+++\n",
			cfg:       cfg,
			wantRules: []string{"MLS-005", "MLS-005"},
		},
		{
			name:    "nil config skips tag enforcement",
			file:    "260101-ms.md",
			content: "+++\ntitle = \"M\"\nsummary = \"S\"\ntags = [\"nope\"]\n+++\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ms, diags := validator.ValidateMilestone(filepath.Join("ms", tc.file), []byte(tc.content), tc.cfg, allowed)
			if got := ruleIDs(diags); !slices.Equal(got, tc.wantRules) {
				t.Errorf("rules = %v, want %v (%+v)", got, tc.wantRules, diags)
			}
			if (ms == nil) != tc.wantNil {
				t.Errorf("milestone nil = %v, want %v", ms == nil, tc.wantNil)
			}
		})
	}
}

func TestValidateGlossaryTermRules(t *testing.T) {
	cfg, allowed := enforcingConfig()
	tests := []struct {
		name      string
		content   string
		cfg       *config.Config
		wantRules []string
		wantNil   bool
	}{
		{name: "valid", content: "+++\ntitle = \"T\"\nsummary = \"S\"\ntags = [\"ok\"]\n+++\nBody\n", cfg: cfg},
		{name: "unclosed frontmatter", content: "+++\ntitle = \"T\"\n", cfg: cfg, wantRules: []string{"GLS-001"}, wantNil: true},
		{name: "invalid toml", content: "+++\ntitle = \n+++\n", cfg: cfg, wantRules: []string{"GLS-001"}, wantNil: true},
		{name: "missing title surfaces as frontmatter error", content: "+++\nsummary = \"S\"\n+++\n", cfg: cfg, wantRules: []string{"GLS-001"}, wantNil: true},
		{name: "disallowed tag", content: "+++\ntitle = \"T\"\nsummary = \"S\"\ntags = [\"bad\"]\n+++\n", cfg: cfg, wantRules: []string{"GLS-004"}},
		{name: "enforcement disabled", content: "+++\ntitle = \"T\"\nsummary = \"S\"\ntags = [\"bad\"]\n+++\n", cfg: &config.Config{Tags: config.TagsConfig{Allowed: []string{"ok"}}}},
		{name: "nil config", content: "+++\ntitle = \"T\"\nsummary = \"S\"\ntags = [\"bad\"]\n+++\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term, diags := validator.ValidateGlossaryTerm("gloss/term.md", []byte(tc.content), tc.cfg, allowed)
			if got := ruleIDs(diags); !slices.Equal(got, tc.wantRules) {
				t.Errorf("rules = %v, want %v (%+v)", got, tc.wantRules, diags)
			}
			if (term == nil) != tc.wantNil {
				t.Errorf("term nil = %v, want %v", term == nil, tc.wantNil)
			}
		})
	}
}

func TestValidateStrategyRules(t *testing.T) {
	cfg, allowed := enforcingConfig()
	tests := []struct {
		name      string
		content   string
		cfg       *config.Config
		wantRules []string
		wantNil   bool
	}{
		{name: "valid default tiers", content: "+++\ntitle = \"S\"\nsummary = \"x\"\ntier = 2\n+++\nbody\n"},
		{name: "bad frontmatter", content: "nope", wantRules: []string{"STR-001"}, wantNil: true},
		{name: "missing title and summary", content: "+++\ntier = 1\n+++\n", wantRules: []string{"STR-002", "STR-002"}},
		{name: "invalid default tier", content: "+++\ntitle = \"S\"\nsummary = \"x\"\ntier = 9\n+++\n", wantRules: []string{"STR-003"}},
		{name: "disallowed tag", content: "+++\ntitle = \"S\"\nsummary = \"x\"\ntier = 1\ntags = [\"bad\"]\n+++\n", cfg: cfg, wantRules: []string{"STR-004"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, diags := validator.ValidateStrategy("strat/s.md", []byte(tc.content), tc.cfg, allowed)
			if got := ruleIDs(diags); !slices.Equal(got, tc.wantRules) {
				t.Errorf("rules = %v, want %v (%+v)", got, tc.wantRules, diags)
			}
			if (s == nil) != tc.wantNil {
				t.Errorf("strategy nil = %v, want %v", s == nil, tc.wantNil)
			}
		})
	}
}

func TestValidateTaskParseErrors(t *testing.T) {
	ctx := validator.TaskContext{ValidStatuses: map[string]bool{"ready": true}}
	tests := []struct {
		name      string
		content   string
		wantRules []string
		wantLine  int
		wantNil   bool
	}{
		{name: "no frontmatter", content: "# Title\n", wantRules: []string{"TSK-002"}, wantNil: true},
		{name: "invalid toml", content: "+++\ntitle = = 1\n+++\n", wantRules: []string{"TSK-002"}, wantLine: 4, wantNil: true},
		{name: "invalid default priority", content: "+++\ntitle = \"T\"\nsummary = \"S\"\nstatus = \"ready\"\npriority = \"meh\"\n+++\n", wantRules: []string{"TSK-005"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task, diags := validator.ValidateTask("tasks/260101-t.md", []byte(tc.content), ctx)
			if got := ruleIDs(diags); !slices.Equal(got, tc.wantRules) {
				t.Fatalf("rules = %v, want %v (%+v)", got, tc.wantRules, diags)
			}
			if tc.wantLine != 0 && diags[0].Line != tc.wantLine {
				t.Errorf("line = %d, want %d", diags[0].Line, tc.wantLine)
			}
			if (task == nil) != tc.wantNil {
				t.Errorf("task nil = %v, want %v", task == nil, tc.wantNil)
			}
		})
	}
}

func TestDetectCyclesCanonicalisation(t *testing.T) {
	// DFS starts at "a" and enters the cycle at "c", so the cycle is discovered
	// as c -> b -> c and must be rotated to start from the smallest node. The
	// missing taskFiles entry makes the diagnostic fall back to the task ID.
	deps := map[string][]string{
		"a": {"c"},
		"b": {"c"},
		"c": {"b"},
	}
	diags := validator.DetectCycles(deps, nil)
	if len(diags) != 1 {
		t.Fatalf("expected 1 cycle, got %d: %+v", len(diags), diags)
	}
	if diags[0].File == "" {
		t.Errorf("expected task ID fallback as file, got empty")
	}
	if !strings.Contains(strings.Join(diags[0].Context, "\n"), "c -> b -> c") {
		t.Errorf("unexpected cycle context: %v", diags[0].Context)
	}
}

func TestValidateWorkspaceConfigLoadError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("this is = = not toml"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := validator.ValidateWorkspace(dir)
	if err != nil {
		t.Fatalf("ValidateWorkspace: %v", err)
	}
	if !slices.Contains(ruleIDs(res.Diagnostics), "CFG-001") {
		t.Errorf("expected CFG-001, got %+v", res.Diagnostics)
	}
	if res.ColumnCount != 0 {
		t.Errorf("ColumnCount = %d, want 0 when config fails to load", res.ColumnCount)
	}
}

func TestValidateWorkspaceUnreadableFiles(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"tasks":      "TSK-002",
		"milestones": "MLS-002",
		"strategies": "STR-001",
		"glossary":   "GLS-001",
	}
	for sub := range cases {
		d := filepath.Join(dir, ".jokateko", sub)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		// A dangling symlink is listed as a non-directory .md entry but cannot be read.
		if err := os.Symlink(filepath.Join(dir, "missing-target"), filepath.Join(d, "260101-broken.md")); err != nil {
			t.Skipf("symlinks not supported: %v", err)
		}
	}
	res, err := validator.ValidateWorkspace(dir)
	if err != nil {
		t.Fatalf("ValidateWorkspace: %v", err)
	}
	for sub, rule := range cases {
		found := false
		for _, d := range res.Diagnostics {
			if d.RuleID == rule && strings.Contains(d.Message, "Failed to read file") && strings.Contains(d.File, sub) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: expected %s read failure, got %+v", sub, rule, res.Diagnostics)
		}
	}
	if res.TaskCount != 1 || res.MilestoneCount != 1 || res.StrategyCount != 1 || res.GlossaryCount != 1 {
		t.Errorf("counts = %d/%d/%d/%d, want 1 each", res.TaskCount, res.MilestoneCount, res.StrategyCount, res.GlossaryCount)
	}
}
