// Package config provides configuration data structures, defaults, TOML loading,
// environment overrides, and schema validation for Jokateko.
package config

import (
	"time"

	"github.com/RJuho/jokateko/internal/model"
)

// CurrentVersion is the currently supported configuration schema version.
const CurrentVersion = "0"

// Config is the root configuration structure loaded from .jokateko/config.toml.
type Config struct {
	Version      string                   `toml:"version"`
	Project      ProjectConfig            `toml:"project"`
	Paths        PathsConfig              `toml:"paths"`
	Server       ServerConfig             `toml:"server"`
	Board        BoardConfig              `toml:"board"`
	Tags         TagsConfig               `toml:"tags"`
	MCP          MCPConfig                `toml:"mcp"`
	Translations model.TranslationsConfig `toml:"translations"`
}

// ProjectConfig specifies project-level descriptive metadata.
type ProjectConfig struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
}

// PathsConfig specifies directory paths for storage and static export.
type PathsConfig struct {
	Tasks      string `toml:"tasks"`
	Milestones string `toml:"milestones"`
	Strategies string `toml:"strategies"`
	Glossary   string `toml:"glossary"`
	Export     string `toml:"export"`
}

// ServerConfig specifies settings for the local HTTP daemon and Web UI.
type ServerConfig struct {
	Host        string               `toml:"host"`
	Port        int                  `toml:"port"`
	OpenBrowser bool                 `toml:"open_browser"`
	Security    ServerSecurityConfig `toml:"security"`
}

// ServerSecurityConfig configures CORS and Content-Security-Policy headers.
type ServerSecurityConfig struct {
	CORSEnabled        bool      `toml:"cors_enabled"`
	CORSAllowedOrigins []string  `toml:"cors_allowed_origins"`
	CSP                CSPConfig `toml:"csp"`
}

// CSPConfig specifies Content-Security-Policy header rules.
type CSPConfig struct {
	Enabled    bool     `toml:"enabled"`
	DefaultSrc []string `toml:"default_src"`
	ScriptSrc  []string `toml:"script_src"`
	StyleSrc   []string `toml:"style_src"`
	ImgSrc     []string `toml:"img_src"`
	ConnectSrc []string `toml:"connect_src"`
	FontSrc    []string `toml:"font_src"`
}

// BoardConfig defines the Kanban workflow columns.
type BoardConfig struct {
	Columns []ColumnConfig `toml:"columns"`
}

// ColumnConfig specifies a single workflow column.
type ColumnConfig struct {
	ID    string `toml:"id"`
	Name  string `toml:"name"`
	Color string `toml:"color"`
}

// TagsConfig specifies controlled tag vocabulary rules.
type TagsConfig struct {
	Allowed        []string `toml:"allowed"`
	EnforceAllowed bool     `toml:"enforce_allowed"`
}

// MCPConfig defines Model Context Protocol settings for AI agents.
type MCPConfig struct {
	Enabled        bool `toml:"enabled"`
	TimeoutSeconds int  `toml:"timeout_seconds"`
	AllowMutations bool `toml:"allow_mutations"`
}

// Timeout returns the configured timeout as a time.Duration.
func (m MCPConfig) Timeout() time.Duration {
	if m.TimeoutSeconds <= 0 {
		return 30 * time.Second
	}
	return time.Duration(m.TimeoutSeconds) * time.Second
}
