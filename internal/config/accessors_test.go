package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
)

func TestDirsAllAndResolveDirs(t *testing.T) {
	absTasks := filepath.Join(t.TempDir(), "abs", "..", "tasks")
	cfg := &config.Config{Paths: config.PathsConfig{
		Tasks:      absTasks,
		Milestones: "ms",
		Strategies: filepath.Join("docs", "strategies"),
		Glossary:   "",
	}}
	ws := filepath.Join(string(filepath.Separator), "workspace")

	dirs := cfg.ResolveDirs(ws)
	want := config.Dirs{
		Tasks:      filepath.Clean(absTasks),
		Milestones: filepath.Join(ws, "ms"),
		Strategies: filepath.Join(ws, "docs", "strategies"),
		Glossary:   ws,
	}
	if dirs != want {
		t.Errorf("ResolveDirs = %+v, want %+v", dirs, want)
	}

	all := dirs.All()
	if !slices.Equal(all, []string{want.Tasks, want.Milestones, want.Strategies, want.Glossary}) {
		t.Errorf("All() = %v, not in stable Tasks/Milestones/Strategies/Glossary order", all)
	}
}

func TestColumns(t *testing.T) {
	t.Run("defaults keep order", func(t *testing.T) {
		cfg := config.Default("")
		cols := cfg.Columns()
		if len(cols) != len(cfg.Board.Columns) {
			t.Fatalf("got %d columns, want %d", len(cols), len(cfg.Board.Columns))
		}
		for i, c := range cfg.Board.Columns {
			if cols[i].ID != c.ID || cols[i].Name != c.Name {
				t.Errorf("column %d = %+v, want id %q", i, cols[i], c.ID)
			}
		}
	})

	t.Run("all fields copied", func(t *testing.T) {
		cfg := &config.Config{Board: config.BoardConfig{Columns: []config.ColumnConfig{{
			ID: "x", Name: "X", Color: "#000", HandledBy: "human", Instructions: "do it", SortBy: "title", SortDirection: "desc",
		}}}}
		want := model.Column{ID: "x", Name: "X", Color: "#000", HandledBy: "human", Instructions: "do it", SortBy: "title", SortDirection: "desc"}
		if got := cfg.Columns(); len(got) != 1 || got[0] != want {
			t.Errorf("Columns() = %+v, want [%+v]", got, want)
		}
	})

	t.Run("no columns", func(t *testing.T) {
		got := (&config.Config{}).Columns()
		if got == nil || len(got) != 0 {
			t.Errorf("Columns() = %#v, want empty non-nil slice", got)
		}
	})
}

func TestNilConfigPolicies(t *testing.T) {
	var cfg *config.Config
	if !cfg.IsTaskEditable(" backlog ") || cfg.IsTaskEditable("done") {
		t.Error("nil config: only backlog should be editable")
	}
	if !cfg.IsTaskCreatable("backlog") || cfg.IsTaskCreatable("ready") {
		t.Error("nil config: only backlog should be creatable")
	}
	if got := cfg.DefaultCreateState(); got != "backlog" {
		t.Errorf("nil config DefaultCreateState = %q", got)
	}
}

func TestDefaultCreateState(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty", "", "backlog"},
		{"whitespace only", "   ", "backlog"},
		{"trimmed", "  ready  ", "ready"},
		{"plain", "todo", "todo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Board: config.BoardConfig{DefaultCreateState: tt.value}}
			if got := cfg.DefaultCreateState(); got != tt.want {
				t.Errorf("DefaultCreateState() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEmptyPolicyListsFallBackToBacklog(t *testing.T) {
	cfg := &config.Config{}
	if !cfg.IsTaskEditable("backlog") || cfg.IsTaskEditable("ready") {
		t.Error("empty editable_states should allow only backlog")
	}
	if !cfg.IsTaskCreatable("backlog") || cfg.IsTaskCreatable("ready") {
		t.Error("empty creatable_states should allow only backlog")
	}
}

func TestIsAllowedTag(t *testing.T) {
	tests := []struct {
		name    string
		tags    config.TagsConfig
		tag     string
		allowed bool
	}{
		{"not enforced allows anything", config.TagsConfig{EnforceAllowed: false, Allowed: []string{"a"}}, "zzz", true},
		{"not enforced allows empty tag", config.TagsConfig{}, "", true},
		{"enforced listed", config.TagsConfig{EnforceAllowed: true, Allowed: []string{"a", "b"}}, "b", true},
		{"enforced unlisted", config.TagsConfig{EnforceAllowed: true, Allowed: []string{"a"}}, "c", false},
		{"enforced is case-sensitive", config.TagsConfig{EnforceAllowed: true, Allowed: []string{"a"}}, "A", false},
		{"enforced with empty list rejects all", config.TagsConfig{EnforceAllowed: true}, "a", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{Tags: tt.tags}
			if got := cfg.IsAllowedTag(tt.tag); got != tt.allowed {
				t.Errorf("IsAllowedTag(%q) = %v, want %v", tt.tag, got, tt.allowed)
			}
		})
	}
}

func TestGetTier(t *testing.T) {
	cfg := config.Default("")
	tests := []struct {
		id     string
		found  bool
		wantNm string
	}{
		{"1", true, "Tier 1"},
		{"3", true, "Tier 3"},
		{"4", false, ""},
		{"", false, ""},
		{" 1", false, ""},
	}
	for _, tt := range tests {
		t.Run("id="+tt.id, func(t *testing.T) {
			tier, ok := cfg.GetTier(tt.id)
			if ok != tt.found {
				t.Fatalf("GetTier(%q) found = %v, want %v", tt.id, ok, tt.found)
			}
			if tier.Name != tt.wantNm {
				t.Errorf("GetTier(%q).Name = %q, want %q", tt.id, tier.Name, tt.wantNm)
			}
			if !ok && tier != (model.TierConfig{}) {
				t.Errorf("GetTier(%q) returned non-zero tier when not found: %+v", tt.id, tier)
			}
		})
	}

	if _, ok := (&config.Config{}).GetTier("1"); ok {
		t.Error("config without tiers must not find tier 1")
	}
}

func TestLoadErrors(t *testing.T) {
	t.Run("unreadable config path", func(t *testing.T) {
		root := t.TempDir()
		// A directory where the file is expected makes ReadFile fail with a non-ErrNotExist error.
		dir := filepath.Join(root, "cfgdir")
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv(config.EnvConfigPath, dir)
		_, err := config.Load(root)
		if err == nil || !strings.Contains(err.Error(), "failed to read configuration file") {
			t.Errorf("expected read error, got %v", err)
		}
	})

	t.Run("invalid TOML", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "custom.toml")
		if err := os.WriteFile(path, []byte("this is = = not toml"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv(config.EnvConfigPath, path)
		_, err := config.Load(root)
		if err == nil || !strings.Contains(err.Error(), "CFG-001") {
			t.Errorf("expected CFG-001 parse error, got %v", err)
		}
	})

	t.Run("default config fails validation via env override", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv(config.EnvConfigPath, filepath.Join(root, "missing.toml"))
		t.Setenv(config.EnvPort, "70000")
		_, err := config.Load(root)
		if err == nil || !strings.Contains(err.Error(), "default configuration failed validation") {
			t.Errorf("expected validation error for out-of-range port, got %v", err)
		}
	})
}
