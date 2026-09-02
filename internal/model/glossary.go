package model

import (
	"slices"
	"time"
)

// GlossaryFrontmatter represents TOML frontmatter for a glossary term file.
type GlossaryFrontmatter struct {
	Title   string   `json:"title" toml:"title" yaml:"title"`
	Tags    []string `json:"tags" toml:"tags" yaml:"tags"`
	Summary string   `json:"summary" toml:"summary" yaml:"summary"`
}

// GlossaryTerm represents a standardized project terminology definition.
type GlossaryTerm struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Tags     []string  `json:"tags"`
	Summary  string    `json:"summary"`
	Body     string    `json:"body,omitempty"`
	BodyHTML string    `json:"body_html,omitempty"`
	FilePath string    `json:"file_path,omitempty"`
	ModTime  time.Time `json:"mod_time,omitzero"`
}

// Frontmatter returns the GlossaryFrontmatter representation.
func (g GlossaryTerm) Frontmatter() GlossaryFrontmatter {
	tags := g.Tags
	if tags == nil {
		tags = []string{}
	}

	return GlossaryFrontmatter{
		Title:   g.Title,
		Tags:    tags,
		Summary: g.Summary,
	}
}

// HasTag returns true if the term contains the given tag.
func (g GlossaryTerm) HasTag(tag string) bool {
	return slices.Contains(g.Tags, tag)
}
