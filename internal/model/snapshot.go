package model

// SnapshotConfig represents the project configuration embedded into static snapshots.
type SnapshotConfig struct {
	Project      ProjectConfig     `json:"project"`
	Board        BoardConfig       `json:"board"`
	Priorities   []PriorityConfig  `json:"priorities,omitzero"`
	Tiers        []TierConfig      `json:"tiers,omitzero"`
	Tags         TagsConfig        `json:"tags,omitzero"`
	Build        BuildConfig       `json:"build,omitzero"`
	Translations map[string]string `json:"translations,omitzero"`
	MCP          MCPConfig         `json:"mcp,omitzero"`
}

// MCPConfig specifies Model Context Protocol settings for snapshots.
type MCPConfig struct {
	Instructions string `json:"instructions,omitempty"`
}

// PriorityConfig specifies a task priority level definition.
type PriorityConfig struct {
	ID    string `json:"id" toml:"id"`
	Name  string `json:"name" toml:"name"`
	Color string `json:"color,omitempty" toml:"color"`
}

// TierConfig specifies an architectural strategy tier level definition.
type TierConfig struct {
	ID      string `json:"id" toml:"id"`
	Name    string `json:"name" toml:"name"`
	Title   string `json:"title" toml:"title"`
	Summary string `json:"summary" toml:"summary"`
	Color   string `json:"color,omitempty" toml:"color"`
}

// BuildConfig represents build timestamp and VCS metadata embedded into static exports.
type BuildConfig struct {
	Time      string `json:"time,omitempty"`
	Branch    string `json:"branch,omitempty"`
	Commit    string `json:"commit,omitempty"`
	Version   string `json:"version,omitempty"`
	GoVersion string `json:"go_version,omitempty"`
	Platform  string `json:"platform,omitempty"`
}

// ProjectConfig represents the project name and description for static exports.
type ProjectConfig struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Locale      string `json:"locale,omitempty"`
}

// BoardConfig represents the column layout for static exports.
type BoardConfig struct {
	Columns            []Column `json:"columns"`
	EditableStates     []string `json:"editable_states,omitempty"`
	CreatableStates    []string `json:"creatable_states,omitempty"`
	DefaultCreateState string   `json:"default_create_state,omitempty"`
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
