package model

// SnapshotConfig represents the project configuration embedded into static snapshots.
type SnapshotConfig struct {
	Project ProjectConfig `json:"project"`
	Board   BoardConfig   `json:"board"`
	Tags    TagsConfig    `json:"tags,omitzero"`
	Build   BuildConfig   `json:"build,omitzero"`
}

// BuildConfig represents build timestamp and VCS metadata embedded into static exports.
type BuildConfig struct {
	Time   string `json:"time,omitempty"`
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit,omitempty"`
}

// ProjectConfig represents the project name and description for static exports.
type ProjectConfig struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// BoardConfig represents the column layout for static exports.
type BoardConfig struct {
	Columns []Column `json:"columns"`
}

// TagsConfig represents tag constraints for static exports.
type TagsConfig struct {
	Allowed        []string `json:"allowed,omitempty"`
	EnforceAllowed bool     `json:"enforce_allowed"`
}

// Snapshot represents the complete project offline state embedded into self-contained HTML.
type Snapshot struct {
	Config     SnapshotConfig `json:"config"`
	Tasks      []Task         `json:"tasks"`
	Milestones []Milestone    `json:"milestones"`
	Strategies []Strategy     `json:"strategies"`
	Glossary   []GlossaryTerm `json:"glossary"`
}
