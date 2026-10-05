package config

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/model"
)

func TestMergeConfigNilRaw(t *testing.T) {
	cfg := Default("/tmp/proj")
	want := Default("/tmp/proj")
	mergeConfig(cfg, nil)
	if !reflect.DeepEqual(cfg, want) {
		t.Error("mergeConfig(nil) must not change the config")
	}
}

func TestMergeConfigEmptyRawKeepsDefaults(t *testing.T) {
	cfg := Default("/tmp/proj")
	want := Default("/tmp/proj")
	empty := ""
	raw := &rawConfig{
		Version:     &empty,
		Name:        &empty,
		Description: &empty,
		Project:     &rawProjectConfig{Name: &empty, Description: &empty},
		Paths: &rawPathsConfig{
			Tasks: &empty, Milestones: &empty, Strategies: &empty, Glossary: &empty, Export: &empty,
		},
		Server: &rawServerConfig{
			Host:     &empty,
			Security: &rawServerSecurityConfig{CSP: &rawCSPConfig{}},
		},
		Board:        &rawBoardConfig{DefaultCreateState: &empty},
		Strategies:   &rawStrategiesConfig{},
		Tags:         &rawTagsConfig{},
		MCP:          &rawMCPConfig{},
		Translations: &model.TranslationsConfig{},
	}
	mergeConfig(cfg, raw)
	if !reflect.DeepEqual(cfg, want) {
		t.Errorf("empty overrides changed defaults:\n got %+v\nwant %+v", cfg, want)
	}
}

func TestMergeConfigOverrideMatrix(t *testing.T) {
	tests := []struct {
		name  string
		raw   rawConfig
		check func(t *testing.T, c *Config)
	}{
		{
			name: "version",
			raw:  rawConfig{Version: new("7")},
			check: func(t *testing.T, c *Config) {
				if c.Version != "7" {
					t.Errorf("version = %q", c.Version)
				}
			},
		},
		{
			name: "project section wins over top-level aliases",
			raw: rawConfig{
				Name:        new("alias-name"),
				Description: new("alias-desc"),
				Project:     &rawProjectConfig{Name: new("proj-name"), Description: new("proj-desc")},
			},
			check: func(t *testing.T, c *Config) {
				if c.Project.Name != "proj-name" || c.Project.Description != "proj-desc" {
					t.Errorf("project = %+v", c.Project)
				}
			},
		},
		{
			name: "top-level aliases used when project section omits the key",
			raw: rawConfig{
				Name:        new("alias-name"),
				Description: new("alias-desc"),
				Project:     &rawProjectConfig{},
			},
			check: func(t *testing.T, c *Config) {
				if c.Project.Name != "alias-name" || c.Project.Description != "alias-desc" {
					t.Errorf("project = %+v", c.Project)
				}
			},
		},
		{
			name: "top-level description alias without project section",
			raw:  rawConfig{Description: new("only-desc")},
			check: func(t *testing.T, c *Config) {
				if c.Project.Description != "only-desc" {
					t.Errorf("description = %q", c.Project.Description)
				}
			},
		},
		{
			name: "all paths",
			raw: rawConfig{Paths: &rawPathsConfig{
				Tasks: new("t"), Milestones: new("m"), Strategies: new("s"), Glossary: new("g"), Export: new("e.html"),
			}},
			check: func(t *testing.T, c *Config) {
				want := PathsConfig{Tasks: "t", Milestones: "m", Strategies: "s", Glossary: "g", Export: "e.html"}
				if c.Paths != want {
					t.Errorf("paths = %+v, want %+v", c.Paths, want)
				}
			},
		},
		{
			name: "server and security",
			raw: rawConfig{Server: &rawServerConfig{
				Host:        new("0.0.0.0"),
				Port:        new(9999),
				OpenBrowser: new(true),
				Security: &rawServerSecurityConfig{
					CORSEnabled:        new(false),
					CORSAllowedOrigins: []string{"https://example.test"},
					CSP: &rawCSPConfig{
						Enabled:      new(false),
						DefaultSrc:   []string{"d"},
						ScriptSrc:    []string{"s"},
						StyleSrc:     []string{"st"},
						StyleSrcElem: []string{"se"},
						StyleSrcAttr: []string{"sa"},
						ImgSrc:       []string{"i"},
						ConnectSrc:   []string{"c"},
						FontSrc:      []string{"f"},
					},
				},
			}},
			check: func(t *testing.T, c *Config) {
				s := c.Server
				if s.Host != "0.0.0.0" || s.Port != 9999 || !s.OpenBrowser {
					t.Errorf("server = %+v", s)
				}
				if s.Security.CORSEnabled || !slices.Equal(s.Security.CORSAllowedOrigins, []string{"https://example.test"}) {
					t.Errorf("security = %+v", s.Security)
				}
				want := CSPConfig{
					DefaultSrc: []string{"d"}, ScriptSrc: []string{"s"}, StyleSrc: []string{"st"},
					StyleSrcElem: []string{"se"}, StyleSrcAttr: []string{"sa"}, ImgSrc: []string{"i"},
					ConnectSrc: []string{"c"}, FontSrc: []string{"f"},
				}
				if !reflect.DeepEqual(s.Security.CSP, want) {
					t.Errorf("csp = %+v, want %+v", s.Security.CSP, want)
				}
			},
		},
		{
			name: "custom columns without backlog derive policies from first column",
			raw:  rawConfig{Board: &rawBoardConfig{Columns: []ColumnConfig{{ID: "todo"}, {ID: "doing"}}}},
			check: func(t *testing.T, c *Config) {
				if !slices.Equal(c.Board.EditableStates, []string{"todo"}) ||
					!slices.Equal(c.Board.CreatableStates, []string{"todo"}) ||
					c.Board.DefaultCreateState != "todo" {
					t.Errorf("board = %+v", c.Board)
				}
			},
		},
		{
			name: "custom columns with backlog keep default policies",
			raw:  rawConfig{Board: &rawBoardConfig{Columns: []ColumnConfig{{ID: "backlog"}, {ID: "done"}}}},
			check: func(t *testing.T, c *Config) {
				if !slices.Equal(c.Board.EditableStates, []string{"backlog"}) || c.Board.DefaultCreateState != "backlog" {
					t.Errorf("board = %+v", c.Board)
				}
				if len(c.Board.Columns) != 2 {
					t.Errorf("columns = %+v", c.Board.Columns)
				}
			},
		},
		{
			name: "explicit board policies win over derived ones",
			raw: rawConfig{Board: &rawBoardConfig{
				Columns:            []ColumnConfig{{ID: "todo"}, {ID: "doing"}},
				EditableStates:     []string{"todo", "doing"},
				CreatableStates:    []string{"doing"},
				DefaultCreateState: new("doing"),
			}},
			check: func(t *testing.T, c *Config) {
				if !slices.Equal(c.Board.EditableStates, []string{"todo", "doing"}) ||
					!slices.Equal(c.Board.CreatableStates, []string{"doing"}) ||
					c.Board.DefaultCreateState != "doing" {
					t.Errorf("board = %+v", c.Board)
				}
			},
		},
		{
			name: "top-level priorities win over board priorities",
			raw: rawConfig{
				Priorities: []model.PriorityConfig{{ID: "top"}},
				Board:      &rawBoardConfig{Priorities: []model.PriorityConfig{{ID: "board"}}},
			},
			check: func(t *testing.T, c *Config) {
				if len(c.Priorities) != 1 || c.Priorities[0].ID != "top" {
					t.Errorf("priorities = %+v", c.Priorities)
				}
			},
		},
		{
			name: "board priorities used when no top-level priorities",
			raw:  rawConfig{Board: &rawBoardConfig{Priorities: []model.PriorityConfig{{ID: "board"}}}},
			check: func(t *testing.T, c *Config) {
				if len(c.Priorities) != 1 || c.Priorities[0].ID != "board" {
					t.Errorf("priorities = %+v", c.Priorities)
				}
			},
		},
		{
			name: "strategies tiers win over top-level tiers",
			raw: rawConfig{
				Strategies: &rawStrategiesConfig{Tiers: []model.TierConfig{{ID: "s"}}},
				Tiers:      []model.TierConfig{{ID: "top"}},
			},
			check: func(t *testing.T, c *Config) {
				if len(c.Strategies.Tiers) != 1 || c.Strategies.Tiers[0].ID != "s" {
					t.Errorf("tiers = %+v", c.Strategies.Tiers)
				}
			},
		},
		{
			name: "top-level tiers used when strategies section has none",
			raw: rawConfig{
				Strategies: &rawStrategiesConfig{},
				Tiers:      []model.TierConfig{{ID: "top"}},
			},
			check: func(t *testing.T, c *Config) {
				if len(c.Strategies.Tiers) != 1 || c.Strategies.Tiers[0].ID != "top" {
					t.Errorf("tiers = %+v", c.Strategies.Tiers)
				}
			},
		},
		{
			name: "tags",
			raw:  rawConfig{Tags: &rawTagsConfig{Allowed: []string{"x"}, EnforceAllowed: new(false)}},
			check: func(t *testing.T, c *Config) {
				if !slices.Equal(c.Tags.Allowed, []string{"x"}) || c.Tags.EnforceAllowed {
					t.Errorf("tags = %+v", c.Tags)
				}
			},
		},
		{
			name: "mcp",
			raw: rawConfig{MCP: &rawMCPConfig{
				Enabled: new(false), TimeoutSeconds: new(5), AllowMutations: new(false), Instructions: new("be nice"),
			}},
			check: func(t *testing.T, c *Config) {
				want := MCPConfig{Enabled: false, TimeoutSeconds: 5, AllowMutations: false, Instructions: "be nice"}
				if c.MCP != want {
					t.Errorf("mcp = %+v, want %+v", c.MCP, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Default("/tmp/proj")
			mergeConfig(cfg, &tt.raw)
			tt.check(t, cfg)
		})
	}
}

// stringFields returns pointers to every string field of a TranslationsConfig.
func stringFields(tc *model.TranslationsConfig) map[string]*string {
	v := reflect.ValueOf(tc).Elem()
	out := make(map[string]*string, v.NumField())
	for i := range v.NumField() {
		f := v.Field(i)
		if f.Kind() == reflect.String {
			out[v.Type().Field(i).Name] = f.Addr().Interface().(*string)
		}
	}
	return out
}

func TestMergeTranslations(t *testing.T) {
	t.Run("every non-empty field overrides", func(t *testing.T) {
		var target, raw model.TranslationsConfig
		for name, p := range stringFields(&target) {
			*p = "default-" + name
		}
		for name, p := range stringFields(&raw) {
			*p = "custom-" + name
		}
		mergeTranslations(&target, &raw)
		for name, p := range stringFields(&target) {
			if *p != "custom-"+name {
				t.Errorf("%s = %q, want override", name, *p)
			}
		}
	})

	t.Run("empty fields keep defaults", func(t *testing.T) {
		var target model.TranslationsConfig
		for name, p := range stringFields(&target) {
			*p = "default-" + name
		}
		want := target
		mergeTranslations(&target, &model.TranslationsConfig{})
		if target != want {
			t.Error("empty translations must not override defaults")
		}
	})

	t.Run("partial override", func(t *testing.T) {
		target := model.TranslationsConfig{Board: "Board", Search: "Search"}
		mergeTranslations(&target, &model.TranslationsConfig{Search: "Haku"})
		if target.Board != "Board" || target.Search != "Haku" {
			t.Errorf("got %+v", target)
		}
	})
}

func TestValidatePathWithinRoot(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"relative inside", ".jokateko/tasks", false},
		{"root itself", ".", false},
		{"absolute inside", filepath.Join(root, "sub", "dir"), false},
		{"relative escape", "../outside", true},
		{"nested escape", "a/../../outside", true},
		{"absolute outside", filepath.Dir(root), true},
		{"absolute elsewhere", string(filepath.Separator) + "etc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePathWithinRoot(root, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePathWithinRoot(%q) error = %v, wantErr %v", tt.target, err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "escapes workspace root") {
				t.Errorf("unexpected error text: %v", err)
			}
		})
	}

	t.Run("relative root", func(t *testing.T) {
		if err := validatePathWithinRoot(".", "sub"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if err := validatePathWithinRoot(".", ".."); err == nil {
			t.Error("expected escape error for '..' with relative root")
		}
	})

	t.Run("dot-dot prefixed directory name inside root", func(t *testing.T) {
		for _, p := range []string{"..cache", "..cache/tasks", "sub/..hidden"} {
			if err := validatePathWithinRoot(root, p); err != nil {
				t.Errorf("validatePathWithinRoot(%q): unexpected error: %v", p, err)
			}
		}
		for _, p := range []string{"..", "../x", "sub/../../x"} {
			if err := validatePathWithinRoot(root, p); err == nil {
				t.Errorf("validatePathWithinRoot(%q): expected escape error", p)
			}
		}
	})
}

func TestResolveConfigPathRootFallback(t *testing.T) {
	t.Setenv(EnvConfigPath, "")
	root := t.TempDir()
	rootCfg := filepath.Join(root, "config.toml")
	if err := os.WriteFile(rootCfg, []byte("version = \"0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := resolveConfigPath(root); got != rootCfg {
		t.Errorf("resolveConfigPath = %q, want %q", got, rootCfg)
	}
}
