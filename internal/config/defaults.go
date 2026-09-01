package config

import (
	"cmp"
	"path/filepath"
)

// Default returns a new Config initialized with standard compiled-in defaults.
// If baseDir is provided, the project name defaults to the base directory's name.
func Default(baseDir string) *Config {
	projectName := "My Project"
	if baseDir != "" && baseDir != "." {
		clean := filepath.Clean(baseDir)
		base := filepath.Base(clean)
		if base != "" && base != "." && base != "/" {
			projectName = base
		}
	}

	return &Config{
		Version: CurrentVersion,
		Project: ProjectConfig{
			Name:        projectName,
			Description: "Markdown-driven task management and Kanban",
		},
		Paths: PathsConfig{
			Tasks:      filepath.Join(".jokateko", "tasks"),
			Milestones: filepath.Join(".jokateko", "milestones"),
			Strategies: filepath.Join(".jokateko", "strategies"),
			Glossary:   filepath.Join(".jokateko", "glossary"),
			Export:     "dist-kanban",
		},
		Server: ServerConfig{
			Host:        "127.0.0.1",
			Port:        8080,
			OpenBrowser: false,
			Security: ServerSecurityConfig{
				CORSEnabled: false,
				CORSAllowedOrigins: []string{
					"http://localhost:3000",
					"http://127.0.0.1:3000",
				},
				CSP: CSPConfig{
					Enabled:    true,
					DefaultSrc: []string{"'self'"},
					ScriptSrc:  []string{"'self'"},
					StyleSrc:   []string{"'self'", "'unsafe-inline'"},
					ImgSrc:     []string{"'self'", "data:"},
					ConnectSrc: []string{"'self'"},
					FontSrc:    []string{"'self'"},
				},
			},
		},
		Board: BoardConfig{
			Columns: []ColumnConfig{
				{ID: "backlog", Name: "Backlog", Color: "#94a3b8"},
				{ID: "ready", Name: "Ready", Color: "#60a5fa"},
				{ID: "in_progress", Name: "In Progress", Color: "#f59e0b"},
				{ID: "in_review", Name: "In Review", Color: "#a855f7"},
				{ID: "done", Name: "Done", Color: "#10b981"},
			},
		},
		Tags: TagsConfig{
			Allowed: []string{
				"backend",
				"frontend",
				"database",
				"security",
				"ui",
				"auth",
				"api",
				"docs",
				"testing",
				"release",
				"infra",
			},
			EnforceAllowed: true,
		},
		MCP: MCPConfig{
			Enabled:        true,
			TimeoutSeconds: cmp.Or(30, 30),
			AllowMutations: true,
		},
	}
}
