package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const (
	// EnvConfigPath overrides the default configuration file location.
	EnvConfigPath = "JOKATEKO_CONFIG"

	// EnvHost overrides the server binding host.
	EnvHost = "JOKATEKO_HOST"

	// EnvPort overrides the server listening port.
	EnvPort = "JOKATEKO_PORT"
)

// Load reads and parses the Jokateko configuration from disk.
// If the configuration file does not exist, it returns the compiled-in default configuration.
// If a configuration file only specifies a subset of settings (e.g. only project name or port),
// all omitted fields automatically retain their standard compiled-in default values.
// Runtime environment variables (JOKATEKO_CONFIG, JOKATEKO_HOST, JOKATEKO_PORT) are applied last.
func Load(rootPath string) (*Config, error) {
	configPath := resolveConfigPath(rootPath)

	cfg := Default(rootPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Graceful fallback: return defaults with environment overrides applied
			applyEnvOverrides(cfg)
			if valErr := Validate(cfg, rootPath); valErr != nil {
				return nil, fmt.Errorf("default configuration failed validation: %w", valErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read configuration file %q: %w", configPath, err)
	}

	var raw rawConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("CFG-001: failed to parse TOML configuration from %q: %w", configPath, err)
	}

	// Merge provided non-nil fields on top of defaults
	mergeConfig(cfg, &raw)

	// Apply runtime environment variable overrides
	applyEnvOverrides(cfg)

	if err := Validate(cfg, rootPath); err != nil {
		return nil, err
	}

	return cfg, nil
}

// resolveConfigPath determines the active configuration file path.
func resolveConfigPath(rootPath string) string {
	if envPath := strings.TrimSpace(os.Getenv(EnvConfigPath)); envPath != "" {
		return envPath
	}
	return filepath.Join(rootPath, ".jokateko", "config.toml")
}

// applyEnvOverrides applies runtime environment variable overrides.
func applyEnvOverrides(cfg *Config) {
	if host := strings.TrimSpace(os.Getenv(EnvHost)); host != "" {
		cfg.Server.Host = host
	}

	if portStr := strings.TrimSpace(os.Getenv(EnvPort)); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = port
		}
	}
}

// rawConfig represents the optional unmarshaled TOML structure where fields
// are pointers or slices to distinguish between omitted fields vs explicit values.
type rawConfig struct {
	Version     *string            `toml:"version"`
	Name        *string            `toml:"name"`        // Optional top-level alias for project.name
	Description *string            `toml:"description"` // Optional top-level alias for project.description
	Project     *rawProjectConfig  `toml:"project"`
	Paths       *rawPathsConfig    `toml:"paths"`
	Server      *rawServerConfig   `toml:"server"`
	Board       *rawBoardConfig    `toml:"board"`
	Tags        *rawTagsConfig     `toml:"tags"`
	MCP         *rawMCPConfig      `toml:"mcp"`
}

type rawProjectConfig struct {
	Name        *string `toml:"name"`
	Description *string `toml:"description"`
}

type rawPathsConfig struct {
	Tasks      *string `toml:"tasks"`
	Milestones *string `toml:"milestones"`
	Strategies *string `toml:"strategies"`
	Glossary   *string `toml:"glossary"`
	Export     *string `toml:"export"`
}

type rawServerConfig struct {
	Host        *string                  `toml:"host"`
	Port        *int                     `toml:"port"`
	OpenBrowser *bool                    `toml:"open_browser"`
	Security    *rawServerSecurityConfig `toml:"security"`
}

type rawServerSecurityConfig struct {
	CORSEnabled        *bool         `toml:"cors_enabled"`
	CORSAllowedOrigins []string      `toml:"cors_allowed_origins"`
	CSP                *rawCSPConfig `toml:"csp"`
}

type rawCSPConfig struct {
	Enabled    *bool    `toml:"enabled"`
	DefaultSrc []string `toml:"default_src"`
	ScriptSrc  []string `toml:"script_src"`
	StyleSrc   []string `toml:"style_src"`
	ImgSrc     []string `toml:"img_src"`
	ConnectSrc []string `toml:"connect_src"`
	FontSrc    []string `toml:"font_src"`
}

type rawBoardConfig struct {
	Columns []ColumnConfig `toml:"columns"`
}

type rawTagsConfig struct {
	Allowed        []string `toml:"allowed"`
	EnforceAllowed *bool    `toml:"enforce_allowed"`
}

type rawMCPConfig struct {
	Enabled        *bool `toml:"enabled"`
	TimeoutSeconds *int  `toml:"timeout_seconds"`
	AllowMutations *bool `toml:"allow_mutations"`
}

// mergeConfig overlays explicit values from rawConfig onto target,
// ensuring any omitted fields preserve their default values.
func mergeConfig(target *Config, raw *rawConfig) {
	if raw == nil {
		return
	}

	if raw.Version != nil && *raw.Version != "" {
		target.Version = *raw.Version
	}

	// Project section & top-level name/description aliases
	if raw.Project != nil {
		if raw.Project.Name != nil && *raw.Project.Name != "" {
			target.Project.Name = *raw.Project.Name
		}
		if raw.Project.Description != nil && *raw.Project.Description != "" {
			target.Project.Description = *raw.Project.Description
		}
	}
	if raw.Name != nil && *raw.Name != "" && (raw.Project == nil || raw.Project.Name == nil) {
		target.Project.Name = *raw.Name
	}
	if raw.Description != nil && *raw.Description != "" && (raw.Project == nil || raw.Project.Description == nil) {
		target.Project.Description = *raw.Description
	}

	// Paths section
	if raw.Paths != nil {
		if raw.Paths.Tasks != nil && *raw.Paths.Tasks != "" {
			target.Paths.Tasks = *raw.Paths.Tasks
		}
		if raw.Paths.Milestones != nil && *raw.Paths.Milestones != "" {
			target.Paths.Milestones = *raw.Paths.Milestones
		}
		if raw.Paths.Strategies != nil && *raw.Paths.Strategies != "" {
			target.Paths.Strategies = *raw.Paths.Strategies
		}
		if raw.Paths.Glossary != nil && *raw.Paths.Glossary != "" {
			target.Paths.Glossary = *raw.Paths.Glossary
		}
		if raw.Paths.Export != nil && *raw.Paths.Export != "" {
			target.Paths.Export = *raw.Paths.Export
		}
	}

	// Server section
	if raw.Server != nil {
		if raw.Server.Host != nil && *raw.Server.Host != "" {
			target.Server.Host = *raw.Server.Host
		}
		if raw.Server.Port != nil {
			target.Server.Port = *raw.Server.Port
		}
		if raw.Server.OpenBrowser != nil {
			target.Server.OpenBrowser = *raw.Server.OpenBrowser
		}
		if raw.Server.Security != nil {
			if raw.Server.Security.CORSEnabled != nil {
				target.Server.Security.CORSEnabled = *raw.Server.Security.CORSEnabled
			}
			if len(raw.Server.Security.CORSAllowedOrigins) > 0 {
				target.Server.Security.CORSAllowedOrigins = raw.Server.Security.CORSAllowedOrigins
			}
			if raw.Server.Security.CSP != nil {
				if raw.Server.Security.CSP.Enabled != nil {
					target.Server.Security.CSP.Enabled = *raw.Server.Security.CSP.Enabled
				}
				if len(raw.Server.Security.CSP.DefaultSrc) > 0 {
					target.Server.Security.CSP.DefaultSrc = raw.Server.Security.CSP.DefaultSrc
				}
				if len(raw.Server.Security.CSP.ScriptSrc) > 0 {
					target.Server.Security.CSP.ScriptSrc = raw.Server.Security.CSP.ScriptSrc
				}
				if len(raw.Server.Security.CSP.StyleSrc) > 0 {
					target.Server.Security.CSP.StyleSrc = raw.Server.Security.CSP.StyleSrc
				}
				if len(raw.Server.Security.CSP.ImgSrc) > 0 {
					target.Server.Security.CSP.ImgSrc = raw.Server.Security.CSP.ImgSrc
				}
				if len(raw.Server.Security.CSP.ConnectSrc) > 0 {
					target.Server.Security.CSP.ConnectSrc = raw.Server.Security.CSP.ConnectSrc
				}
				if len(raw.Server.Security.CSP.FontSrc) > 0 {
					target.Server.Security.CSP.FontSrc = raw.Server.Security.CSP.FontSrc
				}
			}
		}
	}

	// Board columns: if custom columns provided, replace defaults
	if raw.Board != nil && len(raw.Board.Columns) > 0 {
		target.Board.Columns = raw.Board.Columns
	}

	// Tags section
	if raw.Tags != nil {
		if len(raw.Tags.Allowed) > 0 {
			target.Tags.Allowed = raw.Tags.Allowed
		}
		if raw.Tags.EnforceAllowed != nil {
			target.Tags.EnforceAllowed = *raw.Tags.EnforceAllowed
		}
	}

	// MCP section
	if raw.MCP != nil {
		if raw.MCP.Enabled != nil {
			target.MCP.Enabled = *raw.MCP.Enabled
		}
		if raw.MCP.TimeoutSeconds != nil {
			target.MCP.TimeoutSeconds = *raw.MCP.TimeoutSeconds
		}
		if raw.MCP.AllowMutations != nil {
			target.MCP.AllowMutations = *raw.MCP.AllowMutations
		}
	}
}
