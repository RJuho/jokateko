// Package model is a sample model package for the gentypes golden test. It covers
// pointer types and embedded struct fields so the generated TypeScript can be
// checked against encoding/json output.
package model

import "time"

// Status is an exported non-struct type that Sample embeds.
type Status string

// base is unexported, but encoding/json still promotes its exported fields.
type base struct {
	ID     string `json:"id"`
	Hidden string `json:"hidden,omitempty"`
}

// Audit is embedded through a pointer, so its promoted fields are optional.
type Audit struct {
	CreatedBy  string     `json:"created_by"`
	ReviewedAt *time.Time `json:"reviewed_at"`
}

// Meta is embedded with a json tag in Sample and without one in Shadow.
type Meta struct {
	Label string `json:"label"`
	Score int    `json:"score,omitempty"`
}

// Sample exercises pointers and every kind of embedded field.
type Sample struct {
	base
	*Audit
	Meta `json:"meta"`
	Status
	Title    string `json:"title"`
	Untagged int
	internal string
	Skip     string `json:"-"`
	A, B     bool
	Note     *string          `json:"note"`
	Due      *time.Time       `json:"due,omitempty"`
	Count    *int             `json:"count,omitzero"`
	IDs      []*int           `json:"ids"`
	Index    map[string]*Meta `json:"index"`
	Raw      []byte           `json:"raw"`
}

// Shadow's own label field is shallower than Meta's, so it wins.
type Shadow struct {
	Meta
	Label int `json:"label"`
}

// Left and Right collide inside Conflict.
type Left struct {
	Shared string
	Name   string
}

// Right's tagged Name wins over Left's untagged one; both Shared fields are untagged, so the name is dropped.
type Right struct {
	Shared string
	Name   string `json:"Name"`
}

// Conflict embeds two structs whose fields collide at the same depth.
type Conflict struct {
	Left
	Right
	ID string `json:"id"`
}

// SelfRef embeds a pointer to itself; the recursion stops after one level.
type SelfRef struct {
	*SelfRef
	Value string `json:"value"`
}
