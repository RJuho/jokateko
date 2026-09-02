package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default("/tmp/my-project")

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected version %q, got %q", config.CurrentVersion, cfg.Version)
	}

	if cfg.Project.Name != "my-project" {
		t.Errorf("expected project name 'my-project', got %q", cfg.Project.Name)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected 5 default columns, got %d", len(cfg.Board.Columns))
	}

	if len(cfg.Tags.Allowed) != 11 {
		t.Errorf("expected 11 default tags, got %d", len(cfg.Tags.Allowed))
	}

	if !cfg.Tags.EnforceAllowed {
		t.Error("expected tags.enforce_allowed to be true by default")
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}

	if err := config.Validate(cfg, "/tmp/my-project"); err != nil {
		t.Fatalf("default config failed validation: %v", err)
	}

	if !cfg.HasColumn("backlog") {
		t.Error("expected HasColumn('backlog') to be true")
	}

	if cfg.HasColumn("nonexistent") {
		t.Error("expected HasColumn('nonexistent') to be false")
	}

	if len(cfg.Priorities) != 4 {
		t.Errorf("expected 4 default priorities, got %d", len(cfg.Priorities))
	}

	if !cfg.HasPriority("critical") || !cfg.HasPriority("low") {
		t.Error("expected HasPriority('critical') and HasPriority('low') to be true")
	}

	if cfg.HasPriority("nonexistent") {
		t.Error("expected HasPriority('nonexistent') to be false")
	}

	if !cfg.IsAllowedTag("backend") {
		t.Error("expected IsAllowedTag('backend') to be true")
	}

	if cfg.IsAllowedTag("invalid-tag-foo") {
		t.Error("expected IsAllowedTag('invalid-tag-foo') to be false")
	}

	if cfg.MCP.Timeout() != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.MCP.Timeout())
	}
}

func TestLoadMissingFileFallback(t *testing.T) {
	tempDir := t.TempDir()

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected graceful fallback on missing config, got error: %v", err)
	}

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected version %q, got %q", config.CurrentVersion, cfg.Version)
	}
}

func TestLoadValidTOMLFile(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = "0"

[project]
name = "Custom App"
description = "Custom description"

[server]
host = "0.0.0.0"
port = 9090

[[board.columns]]
id = "todo"
name = "To Do"
color = "#3b82f6"

[[board.columns]]
id = "done"
name = "Done"
color = "#10b981"

[tags]
allowed = ["backend", "frontend"]
enforce_allowed = true
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load valid config: %v", err)
	}

	if cfg.Project.Name != "Custom App" {
		t.Errorf("expected 'Custom App', got %q", cfg.Project.Name)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}

	if len(cfg.Board.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cfg.Board.Columns))
	}
}

func TestLoadPartialProjectNameOnly(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[project]
name = "Partial Project Name Only"
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load of partial config, got: %v", err)
	}

	if cfg.Project.Name != "Partial Project Name Only" {
		t.Errorf("expected project name 'Partial Project Name Only', got %q", cfg.Project.Name)
	}

	// Verify all omitted sections and fields retain standard defaults
	if cfg.Project.Description != "Markdown-driven task management and Kanban" {
		t.Errorf("expected default description, got %q", cfg.Project.Description)
	}

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected default version %q, got %q", config.CurrentVersion, cfg.Version)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected default 5 columns, got %d", len(cfg.Board.Columns))
	}

	if len(cfg.Tags.Allowed) != 11 {
		t.Errorf("expected default 11 tags, got %d", len(cfg.Tags.Allowed))
	}

	if cfg.Paths.Tasks != filepath.Join(".jokateko", "tasks") {
		t.Errorf("expected default tasks path, got %q", cfg.Paths.Tasks)
	}
}

func TestLoadPartialProjectDescriptionOnly(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[project]
description = "My Custom Description Only"
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load, got: %v", err)
	}

	if cfg.Project.Description != "My Custom Description Only" {
		t.Errorf("expected custom description, got %q", cfg.Project.Description)
	}

	// Project name defaults to directory basename
	expectedName := filepath.Base(tempDir)
	if cfg.Project.Name != expectedName {
		t.Errorf("expected default project name %q, got %q", expectedName, cfg.Project.Name)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected default 5 columns, got %d", len(cfg.Board.Columns))
	}
}

func TestLoadPartialTopLevelNameAlias(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
name = "Top Level Name App"
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load, got: %v", err)
	}

	if cfg.Project.Name != "Top Level Name App" {
		t.Errorf("expected 'Top Level Name App', got %q", cfg.Project.Name)
	}

	if cfg.Project.Description != "Markdown-driven task management and Kanban" {
		t.Errorf("expected default description, got %q", cfg.Project.Description)
	}
}

func TestLoadPartialServerPortOnly(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[server]
port = 9999
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load, got: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected custom port 9999, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected default host '127.0.0.1', got %q", cfg.Server.Host)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected default 5 columns, got %d", len(cfg.Board.Columns))
	}
}

func TestLoadEmptyTOMLFile(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected empty TOML to succeed with defaults, got: %v", err)
	}

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected default version %q, got %q", config.CurrentVersion, cfg.Version)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}


func TestEnvironmentOverrides(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv(config.EnvHost, "0.0.0.0")
	t.Setenv(config.EnvPort, "8888")

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with env overrides: %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got %q", cfg.Server.Host)
	}

	if cfg.Server.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Server.Port)
	}
}

func TestValidationRules(t *testing.T) {
	root := "/workspace"

	t.Run("CFG-000: Invalid Version", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Version = "999"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-000") {
			t.Errorf("expected CFG-000 error, got: %v", err)
		}
	})

	t.Run("CFG-002: Less than 2 columns", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.Columns = []config.ColumnConfig{
			{ID: "only-one", Name: "Only One"},
		}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-002") {
			t.Errorf("expected CFG-002 error, got: %v", err)
		}
	})

	t.Run("CFG-003: Duplicate column ID", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.Columns = []config.ColumnConfig{
			{ID: "col1", Name: "Column 1"},
			{ID: "col1", Name: "Column 1 Duplicate"},
		}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-003") {
			t.Errorf("expected CFG-003 error, got: %v", err)
		}
	})

	t.Run("CFG-004: Invalid port range", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Server.Port = 80 // Privileged port < 1024
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-004") {
			t.Errorf("expected CFG-004 error for port 80, got: %v", err)
		}

		cfg.Server.Port = 70000 // > 65535
		err = config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-004") {
			t.Errorf("expected CFG-004 error for port 70000, got: %v", err)
		}
	})

	t.Run("CFG-005: Path escaping root", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Paths.Tasks = "../../etc/passwd"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-005") {
			t.Errorf("expected CFG-005 error for escaping path, got: %v", err)
		}
	})

	t.Run("TAG-001: Invalid tag format", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Tags.Allowed = []string{"valid-tag", "INVALID_TAG", "has space"}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "TAG-001") {
			t.Errorf("expected TAG-001 error for uppercase/spaced tags, got: %v", err)
		}
	})

	t.Run("TAG-002: Duplicate tag", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Tags.Allowed = []string{"backend", "frontend", "backend"}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "TAG-002") {
			t.Errorf("expected TAG-002 error for duplicate tag, got: %v", err)
		}
	})

	t.Run("Invalid column color format", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.Columns[0].Color = "blue" // Not a hex color
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "invalid color hex") {
			t.Errorf("expected error for non-hex color, got: %v", err)
		}
	})
}

func TestLoadTranslationsTOML(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = "0"

[project]
name = "Localized App"

[translations]
board = "Taulu"
strategies = "Strategiat"
glossary = "Sanasto"
search = "Hae"
tasks = "tehtävää"
architectural_strategies = "Arkkitehtuuristrategiat"
strategies_subtitle = "Porrastetut ohjeet ja järjestelmäsuunnittelun säännöt"
tiers = "Tasot"
all_tiers = "Kaikki tasot"
project_glossary = "Projektin sanasto"
glossary_subtitle = "Standardoidut termimääritelmät ja sanakirja"
footer_text = "Rakennettu ❤️ 🇪🇺 kera 🤖"
arial_main_nav = "Päänavigointi"
arial_search = "Hae sivustolta"
arial_search_input = "Hae tehtäviä, virstanpylväitä, strategioita ja sanastoa"
`

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with translations: %v", err)
	}

	if cfg.Translations.Board != "Taulu" {
		t.Errorf("expected Translations.Board to be 'Taulu', got %q", cfg.Translations.Board)
	}
	if cfg.Translations.Strategies != "Strategiat" {
		t.Errorf("expected Translations.Strategies to be 'Strategiat', got %q", cfg.Translations.Strategies)
	}
	if cfg.Translations.Glossary != "Sanasto" {
		t.Errorf("expected Translations.Glossary to be 'Sanasto', got %q", cfg.Translations.Glossary)
	}
	if cfg.Translations.FooterText != "Rakennettu ❤️ 🇪🇺 kera 🤖" {
		t.Errorf("expected Translations.FooterText to be 'Rakennettu ❤️ 🇪🇺 kera 🤖', got %q", cfg.Translations.FooterText)
	}
	if cfg.Translations.ArialMainNav != "Päänavigointi" {
		t.Errorf("expected Translations.ArialMainNav to be 'Päänavigointi', got %q", cfg.Translations.ArialMainNav)
	}
}

func TestCustomPrioritiesTOML(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = "0"

[project]
name = "Custom Priorities App"

[[priorities]]
id = "p0"
name = "Blocker"
color = "#ef4444"

[[priorities]]
id = "p1"
name = "Critical"
color = "#f97316"

[[priorities]]
id = "p2"
name = "Normal"
color = "#3b82f6"
`

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with custom priorities: %v", err)
	}

	if len(cfg.Priorities) != 3 {
		t.Fatalf("expected 3 custom priorities, got %d", len(cfg.Priorities))
	}
	if cfg.Priorities[0].ID != "p0" || cfg.Priorities[0].Name != "Blocker" {
		t.Errorf("unexpected priority 0: %+v", cfg.Priorities[0])
	}
	if !cfg.HasPriority("p0") || !cfg.HasPriority("p1") || !cfg.HasPriority("p2") {
		t.Error("expected custom priorities p0, p1, p2 to exist")
	}
	if cfg.HasPriority("critical") {
		t.Error("expected default 'critical' priority to be replaced")
	}
}

func TestValidatePriorities(t *testing.T) {
	root := t.TempDir()

	t.Run("Empty priorities list error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Priorities = nil
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-006") {
			t.Errorf("expected CFG-006 error for empty priorities, got: %v", err)
		}
	})

	t.Run("Duplicate priority ID error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Priorities = append(cfg.Priorities, model.PriorityConfig{ID: "low", Name: "Duplicate Low"})
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-007") {
			t.Errorf("expected CFG-007 error for duplicate priority ID, got: %v", err)
		}
	})

	t.Run("Invalid priority hex color error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Priorities[0].Color = "not-a-hex"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "invalid color hex") {
			t.Errorf("expected invalid color hex error, got: %v", err)
		}
	})
}


