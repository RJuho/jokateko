package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
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
	dotPath := filepath.Join(rootPath, ".jokateko", "config.toml")
	if _, err := os.Stat(dotPath); err == nil {
		return dotPath
	}
	rootConfig := filepath.Join(rootPath, "config.toml")
	if _, err := os.Stat(rootConfig); err == nil {
		return rootConfig
	}
	return dotPath
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
	Board        *rawBoardConfig          `toml:"board"`
	Priorities   []model.PriorityConfig   `toml:"priorities"`
	Strategies   *rawStrategiesConfig     `toml:"strategies"`
	Tiers        []model.TierConfig       `toml:"tiers"`
	Tags         *rawTagsConfig           `toml:"tags"`
	MCP          *rawMCPConfig            `toml:"mcp"`
	Translations *model.TranslationsConfig `toml:"translations"`
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
	StyleSrc     []string `toml:"style_src"`
	StyleSrcElem []string `toml:"style_src_elem"`
	StyleSrcAttr []string `toml:"style_src_attr"`
	ImgSrc       []string `toml:"img_src"`
	ConnectSrc   []string `toml:"connect_src"`
	FontSrc      []string `toml:"font_src"`
}

type rawBoardConfig struct {
	Columns            []ColumnConfig         `toml:"columns"`
	EditableStates     []string               `toml:"editable_states"`
	CreatableStates    []string               `toml:"creatable_states"`
	DefaultCreateState *string                `toml:"default_create_state"`
	Priorities         []model.PriorityConfig `toml:"priorities"`
}

type rawStrategiesConfig struct {
	Tiers []model.TierConfig `toml:"tiers"`
}

type rawTagsConfig struct {
	Allowed        []string `toml:"allowed"`
	EnforceAllowed *bool    `toml:"enforce_allowed"`
}

type rawMCPConfig struct {
	Enabled        *bool   `toml:"enabled"`
	TimeoutSeconds *int    `toml:"timeout_seconds"`
	AllowMutations *bool   `toml:"allow_mutations"`
	Instructions   *string `toml:"instructions"`
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
				if len(raw.Server.Security.CSP.StyleSrcElem) > 0 {
					target.Server.Security.CSP.StyleSrcElem = raw.Server.Security.CSP.StyleSrcElem
				}
				if len(raw.Server.Security.CSP.StyleSrcAttr) > 0 {
					target.Server.Security.CSP.StyleSrcAttr = raw.Server.Security.CSP.StyleSrcAttr
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

	// Board columns & editable/creatable states: if custom values provided, replace defaults
	if raw.Board != nil {
		if len(raw.Board.Columns) > 0 {
			target.Board.Columns = raw.Board.Columns
			if len(raw.Board.EditableStates) == 0 && !target.HasColumn("backlog") {
				target.Board.EditableStates = []string{raw.Board.Columns[0].ID}
			}
			if len(raw.Board.CreatableStates) == 0 && !target.HasColumn("backlog") {
				target.Board.CreatableStates = []string{raw.Board.Columns[0].ID}
			}
			if (raw.Board.DefaultCreateState == nil || *raw.Board.DefaultCreateState == "") && !target.HasColumn("backlog") {
				target.Board.DefaultCreateState = raw.Board.Columns[0].ID
			}
		}
		if len(raw.Board.EditableStates) > 0 {
			target.Board.EditableStates = raw.Board.EditableStates
		}
		if len(raw.Board.CreatableStates) > 0 {
			target.Board.CreatableStates = raw.Board.CreatableStates
		}
		if raw.Board.DefaultCreateState != nil && *raw.Board.DefaultCreateState != "" {
			target.Board.DefaultCreateState = *raw.Board.DefaultCreateState
		}
	}

	// Priorities: if custom priorities provided (top-level or under board), replace defaults
	if len(raw.Priorities) > 0 {
		target.Priorities = raw.Priorities
	} else if raw.Board != nil && len(raw.Board.Priorities) > 0 {
		target.Priorities = raw.Board.Priorities
	}

	// Strategies tiers: if custom tiers provided (under strategies or top-level), replace defaults
	if raw.Strategies != nil && len(raw.Strategies.Tiers) > 0 {
		target.Strategies.Tiers = raw.Strategies.Tiers
	} else if len(raw.Tiers) > 0 {
		target.Strategies.Tiers = raw.Tiers
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
		if raw.MCP.Instructions != nil {
			target.MCP.Instructions = *raw.MCP.Instructions
		}
	}

	// Translations section
	if raw.Translations != nil {
		mergeTranslations(&target.Translations, raw.Translations)
	}
}

func mergeTranslations(target *model.TranslationsConfig, raw *model.TranslationsConfig) {
	if raw.Board != "" { target.Board = raw.Board }
	if raw.Strategies != "" { target.Strategies = raw.Strategies }
	if raw.Glossary != "" { target.Glossary = raw.Glossary }
	if raw.Milestones != "" { target.Milestones = raw.Milestones }
	if raw.Search != "" { target.Search = raw.Search }
	if raw.Tasks != "" { target.Tasks = raw.Tasks }
	if raw.ArchitecturalStrategies != "" { target.ArchitecturalStrategies = raw.ArchitecturalStrategies }
	if raw.StrategiesSubtitle != "" { target.StrategiesSubtitle = raw.StrategiesSubtitle }
	if raw.Tiers != "" { target.Tiers = raw.Tiers }
	if raw.AllTiers != "" { target.AllTiers = raw.AllTiers }
	if raw.ProjectGlossary != "" { target.ProjectGlossary = raw.ProjectGlossary }
	if raw.GlossarySubtitle != "" { target.GlossarySubtitle = raw.GlossarySubtitle }
	if raw.FooterText != "" { target.FooterText = raw.FooterText }
	if raw.Reset != "" { target.Reset = raw.Reset }
	if raw.NoTasks != "" { target.NoTasks = raw.NoTasks }
	if raw.NoMatchingResults != "" { target.NoMatchingResults = raw.NoMatchingResults }
	if raw.NoMatchesCurrentPage != "" { target.NoMatchesCurrentPage = raw.NoMatchesCurrentPage }
	if raw.NoMatchesOtherPages != "" { target.NoMatchesOtherPages = raw.NoMatchesOtherPages }
	if raw.ShowCompleted != "" { target.ShowCompleted = raw.ShowCompleted }
	if raw.HideCompleted != "" { target.HideCompleted = raw.HideCompleted }
	if raw.ShowArchived != "" { target.ShowArchived = raw.ShowArchived }

	if raw.SortBy != "" { target.SortBy = raw.SortBy }
	if raw.SortDefault != "" { target.SortDefault = raw.SortDefault }
	if raw.SortPriority != "" { target.SortPriority = raw.SortPriority }
	if raw.SortTargetAt != "" { target.SortTargetAt = raw.SortTargetAt }
	if raw.SortChangedAt != "" { target.SortChangedAt = raw.SortChangedAt }
	if raw.SortCreatedAt != "" { target.SortCreatedAt = raw.SortCreatedAt }
	if raw.SortTitle != "" { target.SortTitle = raw.SortTitle }

	if raw.ArialMainNav != "" { target.ArialMainNav = raw.ArialMainNav }
	if raw.ArialMobileNav != "" { target.ArialMobileNav = raw.ArialMobileNav }
	if raw.ArialMobileMenu != "" { target.ArialMobileMenu = raw.ArialMobileMenu }
	if raw.ArialOpenMenu != "" { target.ArialOpenMenu = raw.ArialOpenMenu }
	if raw.ArialCloseMenu != "" { target.ArialCloseMenu = raw.ArialCloseMenu }
	if raw.ArialSearch != "" { target.ArialSearch = raw.ArialSearch }
	if raw.ArialSearchInput != "" { target.ArialSearchInput = raw.ArialSearchInput }
	if raw.ArialSearchResults != "" { target.ArialSearchResults = raw.ArialSearchResults }
	if raw.ArialThemeToggle != "" { target.ArialThemeToggle = raw.ArialThemeToggle }
	if raw.ArialThemeDark != "" { target.ArialThemeDark = raw.ArialThemeDark }
	if raw.ArialThemeLightLabel != "" { target.ArialThemeLightLabel = raw.ArialThemeLightLabel }
	if raw.ArialThemeDarkLabel != "" { target.ArialThemeDarkLabel = raw.ArialThemeDarkLabel }
	if raw.ArialFilterTasks != "" { target.ArialFilterTasks = raw.ArialFilterTasks }
	if raw.ArialFilterByPriority != "" { target.ArialFilterByPriority = raw.ArialFilterByPriority }
	if raw.ArialFilterByTags != "" { target.ArialFilterByTags = raw.ArialFilterByTags }
	if raw.ArialResetAllFilters != "" { target.ArialResetAllFilters = raw.ArialResetAllFilters }
	if raw.ArialSortTasks != "" { target.ArialSortTasks = raw.ArialSortTasks }
	if raw.ArialSortOptions != "" { target.ArialSortOptions = raw.ArialSortOptions }
	if raw.ArialColumnQuickNav != "" { target.ArialColumnQuickNav = raw.ArialColumnQuickNav }
	if raw.ArialKanbanColumns != "" { target.ArialKanbanColumns = raw.ArialKanbanColumns }
	if raw.ArialMilestonesRoadmap != "" { target.ArialMilestonesRoadmap = raw.ArialMilestonesRoadmap }
	if raw.ArialFooter != "" { target.ArialFooter = raw.ArialFooter }
	if raw.ArialGithubRepo != "" { target.ArialGithubRepo = raw.ArialGithubRepo }
}
