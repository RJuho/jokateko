import { config } from '../state/store'

/**
 * Standard English fallback dictionary for UI labels and aria attributes.
 * All aria labels are explicitly prefixed with 'arial' in their name.
 */
export const defaultTranslations = {
	// Navigation & Core Entities
	board: 'Board',
	strategies: 'Strategies',
	glossary: 'Glossary',
	milestones: 'Milestones',
	search: 'Search',
	tasks: 'tasks',

	// Architectural Strategies Page
	architectural_strategies: 'Architectural Strategies',
	strategies_subtitle:
		'Tiered guidelines and system design rules using progressive disclosure',
	tiers: 'Tiers',
	all_tiers: 'All Tiers',

	// Project Glossary Page
	project_glossary: 'Project Glossary',
	glossary_subtitle:
		'Standardized domain definitions and terminology dictionary',

	// Footer
	footer_text: 'Build with ❤️ in 🇪🇺 with 🤖',

	// Common UI Actions & Feedback
	reset: 'Reset',
	no_tasks: 'No tasks',
	no_matching_results: 'No matching results found',
	no_matches_current_page: 'No matches on current page',
	no_matches_other_pages: 'No matches on other pages',
	show_completed: 'Show completed',
	hide_completed: 'Hide completed',
	show_archived: 'Show Archived',

	// Task Sorting
	sort_by: 'Sort',
	sort_default: 'Default (Workflow)',
	sort_priority: 'Priority (Critical to Low)',
	sort_target_at: 'Target Date (Soonest First)',
	sort_changed_at: 'Recently Changed',
	sort_created_at: 'Created Date',
	sort_title: 'Alphabetical (A-Z)',

	// Aria Attributes (explicitly marked with 'arial' in name)
	arial_main_nav: 'Main Navigation',
	arial_mobile_nav: 'Mobile Navigation',
	arial_mobile_menu: 'Mobile navigation menu',
	arial_open_menu: 'Open menu',
	arial_close_menu: 'Close menu',
	arial_search: 'Site search',
	arial_search_input: 'Search tasks, milestones, strategies, and glossary',
	arial_search_results: 'Search results',
	arial_theme_toggle: 'Toggle light and dark theme',
	arial_theme_dark: 'Toggle dark theme',
	arial_theme_light_label: 'Light theme',
	arial_theme_dark_label: 'Dark theme',
	arial_filter_tasks: 'Filter tasks',
	arial_filter_by_priority: 'Filter by priority',
	arial_filter_by_tags: 'Filter by tags',
	arial_reset_all_filters: 'Reset all filters',
	arial_sort_tasks: 'Sort tasks',
	arial_sort_options: 'Sort options',
	arial_column_quick_nav: 'Column quick navigation',
	arial_kanban_columns: 'Kanban columns',
	arial_milestones_roadmap: 'Milestones roadmap',
	arial_footer: 'Application footer',
	arial_github_repo: 'Jokateko GitHub repository',
} as const

export type TranslationKey = keyof typeof defaultTranslations

/**
 * Returns the localized text for the given translation key.
 * If not translated in the active configuration, falls back to default English.
 */
export function t(key: TranslationKey | string, fallback?: string): string {
	const userTranslations = config.value.translations as
		| Record<string, string | undefined>
		| undefined
	const translated = userTranslations?.[key]
	if (
		translated !== undefined
		&& typeof translated === 'string'
		&& translated.trim() !== ''
	) {
		return translated
	}
	return fallback ?? (defaultTranslations as Record<string, string>)[key] ?? key
}
