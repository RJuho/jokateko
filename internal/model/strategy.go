package model

import (
	"slices"
	"time"
)

// Tier represents the abstraction level of an architectural strategy
// under the Progressive Disclosure model.
type Tier int

const (
	// TierCore represents critical system-wide invariants (e.g. Zero CGO, tasks-as-code).
	TierCore Tier = 1

	// TierDomain represents domain-specific architectural rules (e.g. database schema, auth rules).
	TierDomain Tier = 2

	// TierImplementation represents deep technical details (e.g. state management, CSS styling).
	TierImplementation Tier = 3
)

// ValidTiers lists all allowed strategy tier values.
var ValidTiers = []Tier{
	TierCore,
	TierDomain,
	TierImplementation,
}

// IsValid reports whether the tier is 1, 2, or 3.
func (t Tier) IsValid() bool {
	return slices.Contains(ValidTiers, t)
}

// Name returns the human-readable designation of the tier level.
func (t Tier) Name() string {
	switch t {
	case TierCore:
		return "Core"
	case TierDomain:
		return "Domain"
	case TierImplementation:
		return "Implementation"
	default:
		return "Unknown"
	}
}

// StrategyFrontmatter represents TOML frontmatter for an architectural guideline file.
type StrategyFrontmatter struct {
	Title   string   `json:"title" toml:"title" yaml:"title"`
	Tier    Tier     `json:"tier" toml:"tier" yaml:"tier"`
	Tags    []string `json:"tags" toml:"tags" yaml:"tags"`
	Summary string   `json:"summary" toml:"summary" yaml:"summary"`
}

// Strategy represents an architectural rule or guideline with progressive disclosure support.
type Strategy struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Tier     Tier      `json:"tier"`
	Tags     []string  `json:"tags"`
	Summary  string    `json:"summary"`
	Body     string    `json:"body,omitempty"`
	FilePath string    `json:"file_path,omitempty"`
	ModTime  time.Time `json:"mod_time,omitzero"`
}

// Frontmatter returns the StrategyFrontmatter representation.
func (s Strategy) Frontmatter() StrategyFrontmatter {
	tier := s.Tier
	if !tier.IsValid() {
		tier = TierCore
	}

	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}

	return StrategyFrontmatter{
		Title:   s.Title,
		Tier:    tier,
		Tags:    tags,
		Summary: s.Summary,
	}
}

// HasTag returns true if the strategy contains the given tag.
func (s Strategy) HasTag(tag string) bool {
	return slices.Contains(s.Tags, tag)
}
