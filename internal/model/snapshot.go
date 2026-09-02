package model

// SnapshotConfig represents the project configuration embedded into static snapshots.
type SnapshotConfig struct {
	Project      ProjectConfig      `json:"project"`
	Board        BoardConfig        `json:"board"`
	Tags         TagsConfig         `json:"tags,omitzero"`
	Build        BuildConfig        `json:"build,omitzero"`
	Translations TranslationsConfig `json:"translations,omitzero"`
}

// TranslationsConfig represents customizable user-facing UI labels and aria attributes.
type TranslationsConfig struct {
	Board                   string `json:"board,omitempty" toml:"board"`
	Strategies              string `json:"strategies,omitempty" toml:"strategies"`
	Glossary                string `json:"glossary,omitempty" toml:"glossary"`
	Milestones              string `json:"milestones,omitempty" toml:"milestones"`
	Search                  string `json:"search,omitempty" toml:"search"`
	Tasks                   string `json:"tasks,omitempty" toml:"tasks"`
	ArchitecturalStrategies string `json:"architectural_strategies,omitempty" toml:"architectural_strategies"`
	StrategiesSubtitle      string `json:"strategies_subtitle,omitempty" toml:"strategies_subtitle"`
	Tiers                   string `json:"tiers,omitempty" toml:"tiers"`
	AllTiers                string `json:"all_tiers,omitempty" toml:"all_tiers"`
	ProjectGlossary         string `json:"project_glossary,omitempty" toml:"project_glossary"`
	GlossarySubtitle        string `json:"glossary_subtitle,omitempty" toml:"glossary_subtitle"`
	FooterText              string `json:"footer_text,omitempty" toml:"footer_text"`
	Reset                   string `json:"reset,omitempty" toml:"reset"`
	NoTasks                 string `json:"no_tasks,omitempty" toml:"no_tasks"`
	NoMatchingResults       string `json:"no_matching_results,omitempty" toml:"no_matching_results"`
	NoMatchesCurrentPage    string `json:"no_matches_current_page,omitempty" toml:"no_matches_current_page"`
	NoMatchesOtherPages     string `json:"no_matches_other_pages,omitempty" toml:"no_matches_other_pages"`
	ShowCompleted           string `json:"show_completed,omitempty" toml:"show_completed"`
	HideCompleted           string `json:"hide_completed,omitempty" toml:"hide_completed"`
	ShowArchived            string `json:"show_archived,omitempty" toml:"show_archived"`

	// Aria attributes marked with 'arial' in name
	ArialMainNav           string `json:"arial_main_nav,omitempty" toml:"arial_main_nav"`
	ArialMobileNav         string `json:"arial_mobile_nav,omitempty" toml:"arial_mobile_nav"`
	ArialMobileMenu        string `json:"arial_mobile_menu,omitempty" toml:"arial_mobile_menu"`
	ArialOpenMenu          string `json:"arial_open_menu,omitempty" toml:"arial_open_menu"`
	ArialCloseMenu         string `json:"arial_close_menu,omitempty" toml:"arial_close_menu"`
	ArialSearch            string `json:"arial_search,omitempty" toml:"arial_search"`
	ArialSearchInput       string `json:"arial_search_input,omitempty" toml:"arial_search_input"`
	ArialSearchResults     string `json:"arial_search_results,omitempty" toml:"arial_search_results"`
	ArialThemeToggle       string `json:"arial_theme_toggle,omitempty" toml:"arial_theme_toggle"`
	ArialThemeDark         string `json:"arial_theme_dark,omitempty" toml:"arial_theme_dark"`
	ArialThemeLightLabel   string `json:"arial_theme_light_label,omitempty" toml:"arial_theme_light_label"`
	ArialThemeDarkLabel    string `json:"arial_theme_dark_label,omitempty" toml:"arial_theme_dark_label"`
	ArialFilterTasks       string `json:"arial_filter_tasks,omitempty" toml:"arial_filter_tasks"`
	ArialFilterByPriority  string `json:"arial_filter_by_priority,omitempty" toml:"arial_filter_by_priority"`
	ArialFilterByTags      string `json:"arial_filter_by_tags,omitempty" toml:"arial_filter_by_tags"`
	ArialResetAllFilters   string `json:"arial_reset_all_filters,omitempty" toml:"arial_reset_all_filters"`
	ArialColumnQuickNav    string `json:"arial_column_quick_nav,omitempty" toml:"arial_column_quick_nav"`
	ArialKanbanColumns     string `json:"arial_kanban_columns,omitempty" toml:"arial_kanban_columns"`
	ArialMilestonesRoadmap string `json:"arial_milestones_roadmap,omitempty" toml:"arial_milestones_roadmap"`
	ArialFooter            string `json:"arial_footer,omitempty" toml:"arial_footer"`
	ArialGithubRepo        string `json:"arial_github_repo,omitempty" toml:"arial_github_repo"`
}

// BuildConfig represents build timestamp and VCS metadata embedded into static exports.
type BuildConfig struct {
	Time   string `json:"time,omitempty"`
	Branch string `json:"branch,omitempty"`
	Commit  string `json:"commit,omitempty"`
	Version string `json:"version,omitempty"`
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
