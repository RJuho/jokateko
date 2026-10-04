package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
)

var (
	tagRegex            = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	hexColorRe          = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)
	validSortByFields   = []string{"default", "priority", "target_at", "changed_at", "created_at", "title", "id"}
	validSortDirections = []string{"asc", "desc"}
)

// Validate validates the configuration against all invariants and specification rules.
// rootPath is used to ensure configured directory paths do not escape the workspace.
func Validate(cfg *Config, rootPath string) error {
	if cfg == nil {
		return errors.New("configuration is nil")
	}

	var errs []error

	// CFG-000: Configuration Version
	if cfg.Version != CurrentVersion {
		errs = append(errs, fmt.Errorf("CFG-000: unsupported config version %q; expected %q", cfg.Version, CurrentVersion))
	}

	// CFG-004: Port range
	if cfg.Server.Port < 1024 || cfg.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("CFG-004: server.port must be between 1024 and 65535; got %d", cfg.Server.Port))
	}

	// CFG-002: Minimum columns
	if len(cfg.Board.Columns) < 2 {
		errs = append(errs, errors.New("CFG-002: at least two board columns must be defined"))
	}

	// CFG-003: Unique and non-empty column IDs
	seenColumns := make(map[string]bool, len(cfg.Board.Columns))
	for i, col := range cfg.Board.Columns {
		trimmedID := strings.TrimSpace(col.ID)
		if trimmedID == "" {
			errs = append(errs, fmt.Errorf("CFG-003: board column at index %d has an empty id", i))
			continue
		}
		if seenColumns[trimmedID] {
			errs = append(errs, fmt.Errorf("CFG-003: duplicate board column id %q", trimmedID))
		}
		seenColumns[trimmedID] = true

		if col.Color != "" && !hexColorRe.MatchString(col.Color) {
			errs = append(errs, fmt.Errorf("invalid color hex %q for column %q; expected format #rgb or #rrggbb", col.Color, trimmedID))
		}

		if col.SortBy != "" {
			trimmedSortBy := strings.ToLower(strings.TrimSpace(col.SortBy))
			if !slices.Contains(validSortByFields, trimmedSortBy) {
				errs = append(errs, fmt.Errorf("invalid sort_by %q for column %q; expected one of %v", col.SortBy, trimmedID, validSortByFields))
			}
		}

		if col.SortDirection != "" {
			trimmedSortDir := strings.ToLower(strings.TrimSpace(col.SortDirection))
			if !slices.Contains(validSortDirections, trimmedSortDir) {
				errs = append(errs, fmt.Errorf("invalid sort_direction %q for column %q; expected 'asc' or 'desc'", col.SortDirection, trimmedID))
			}
		}
	}

	// Board creatable_states & default_create_state validation
	for _, state := range cfg.Board.CreatableStates {
		trimmed := strings.TrimSpace(state)
		if trimmed != "" && !seenColumns[trimmed] {
			errs = append(errs, fmt.Errorf("board.creatable_states contains unknown column %q", trimmed))
		}
	}
	if dcs := strings.TrimSpace(cfg.Board.DefaultCreateState); dcs != "" {
		if !seenColumns[dcs] {
			errs = append(errs, fmt.Errorf("board.default_create_state %q is not a defined column", dcs))
		}
	}

	// Priorities validation
	if len(cfg.Priorities) == 0 {
		errs = append(errs, errors.New("CFG-006: at least one priority must be defined"))
	}
	seenPriorities := make(map[string]bool, len(cfg.Priorities))
	for i, p := range cfg.Priorities {
		trimmedID := strings.TrimSpace(p.ID)
		if trimmedID == "" {
			errs = append(errs, fmt.Errorf("CFG-007: priority at index %d has an empty id", i))
			continue
		}
		if seenPriorities[trimmedID] {
			errs = append(errs, fmt.Errorf("CFG-007: duplicate priority id %q", trimmedID))
		}
		seenPriorities[trimmedID] = true

		if p.Color != "" && !hexColorRe.MatchString(p.Color) {
			errs = append(errs, fmt.Errorf("invalid color hex %q for priority %q; expected format #rgb or #rrggbb", p.Color, trimmedID))
		}
	}

	// Strategies tiers validation
	if len(cfg.Strategies.Tiers) == 0 {
		errs = append(errs, errors.New("CFG-008: at least one strategy tier must be defined"))
	}
	seenTiers := make(map[string]bool, len(cfg.Strategies.Tiers))
	for i, tr := range cfg.Strategies.Tiers {
		trimmedID := strings.TrimSpace(tr.ID)
		if trimmedID == "" {
			errs = append(errs, fmt.Errorf("CFG-009: strategy tier at index %d has an empty id", i))
			continue
		}
		if seenTiers[trimmedID] {
			errs = append(errs, fmt.Errorf("CFG-009: duplicate strategy tier id %q", trimmedID))
		}
		seenTiers[trimmedID] = true

		if tr.Color != "" && !hexColorRe.MatchString(tr.Color) {
			errs = append(errs, fmt.Errorf("invalid color hex %q for strategy tier %q; expected format #rgb or #rrggbb", tr.Color, trimmedID))
		}
	}

	// CFG-005: Path traversal verification
	cleanRoot := filepath.Clean(rootPath)
	pathChecks := []struct {
		name string
		path string
	}{
		{"paths.tasks", cfg.Paths.Tasks},
		{"paths.milestones", cfg.Paths.Milestones},
		{"paths.strategies", cfg.Paths.Strategies},
		{"paths.glossary", cfg.Paths.Glossary},
		{"paths.export", cfg.Paths.Export},
	}

	for _, pc := range pathChecks {
		if strings.TrimSpace(pc.path) == "" {
			errs = append(errs, fmt.Errorf("%s must not be empty", pc.name))
			continue
		}
		if err := validatePathWithinRoot(cleanRoot, pc.path); err != nil {
			errs = append(errs, fmt.Errorf("CFG-005: %s (%q) invalid: %w", pc.name, pc.path, err))
		}
	}

	// TAG-001 & TAG-002: Tag vocabulary rules
	if cfg.Tags.EnforceAllowed {
		if len(cfg.Tags.Allowed) == 0 {
			errs = append(errs, errors.New("TAG-001: tags.allowed must contain at least one tag when enforce_allowed is true"))
		}

		seenTags := make(map[string]bool, len(cfg.Tags.Allowed))
		for _, tag := range cfg.Tags.Allowed {
			if !tagRegex.MatchString(tag) {
				errs = append(errs, fmt.Errorf("TAG-001: tag %q must be lowercase kebab-case (^[a-z0-9]+(-[a-z0-9]+)*$)", tag))
			}
			if seenTags[tag] {
				errs = append(errs, fmt.Errorf("TAG-002: duplicate tag %q in tags.allowed", tag))
			}
			seenTags[tag] = true
		}
	}

	// Security / CSP checks
	if cfg.Server.Security.CSP.Enabled {
		if len(cfg.Server.Security.CSP.DefaultSrc) == 0 {
			errs = append(errs, errors.New("server.security.csp.default_src must not be empty when CSP is enabled"))
		}
	}

	return errors.Join(errs...)
}

// validatePathWithinRoot checks that a given path does not escape the workspace root.
func validatePathWithinRoot(root, target string) error {
	var fullPath string
	if filepath.IsAbs(target) {
		fullPath = filepath.Clean(target)
	} else {
		fullPath = filepath.Clean(filepath.Join(root, target))
	}

	// If root is empty or relative current dir, resolve absolute comparison
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(fullPath)
	if err != nil {
		return err
	}

	// Target must have absRoot as prefix (or be equal)
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil {
		return err
	}

	if strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "/..") {
		return fmt.Errorf("path escapes workspace root %q", root)
	}

	return nil
}

// IsAllowedTag checks whether a given tag is permitted under the configuration rules.
func (c *Config) IsAllowedTag(tag string) bool {
	if !c.Tags.EnforceAllowed {
		return true
	}
	return slices.Contains(c.Tags.Allowed, tag)
}

// HasColumn checks if a given column ID is defined in the board configuration.
func (c *Config) HasColumn(columnID string) bool {
	return slices.ContainsFunc(c.Board.Columns, func(col ColumnConfig) bool { return col.ID == columnID })
}

// HasPriority checks if a given priority ID is defined in the priorities configuration.
func (c *Config) HasPriority(priorityID string) bool {
	for _, p := range c.Priorities {
		if p.ID == priorityID {
			return true
		}
	}
	return false
}

// HasTier checks if a given tier ID is defined in the strategy tiers configuration.
func (c *Config) HasTier(tierID string) bool {
	for _, tr := range c.Strategies.Tiers {
		if tr.ID == tierID {
			return true
		}
	}
	return false
}

// GetTier returns the tier configuration for a given tier ID, if defined.
func (c *Config) GetTier(tierID string) (model.TierConfig, bool) {
	for _, tr := range c.Strategies.Tiers {
		if tr.ID == tierID {
			return tr, true
		}
	}
	return model.TierConfig{}, false
}
