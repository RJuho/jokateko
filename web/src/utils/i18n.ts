import { effect } from '@preact/signals'
import { config } from '../state/store'

/**
 * Standard English fallback dictionary for UI labels and aria attributes.
 * All aria labels and tooltip titles are explicitly prefixed with 'arial' in
 * their name. Values may contain {name} placeholders, filled in by tf() and
 * tParts(). Every key must also be listed in internal/config/default.toml.
 */
export const defaultTranslations = {
	// Navigation & Core Entities
	board: 'Board',
	calendar: 'Calendar',
	strategies: 'Strategies',
	glossary: 'Glossary',
	milestones: 'Milestones',
	search: 'Search',
	tasks: 'tasks',
	today: 'Today',
	month: 'Month',
	week: 'Week',
	week_num: 'Week',
	more: 'more',
	add_task_date: 'Add task for this date',
	view_week_tasks: 'View all tasks in week view',
	no_tasks_for_day: 'No tasks scheduled',
	state: 'State',

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
	no_strategies_found: 'No strategies found',
	no_terms_found: 'No glossary terms found',
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

	// Shared Modal Actions
	cancel: 'Cancel',
	close: 'Close',
	saving: 'Saving...',
	copied: 'Copied!',
	copy: 'Copy',
	id_copied: 'copied!',
	none: 'None',

	// Task Detail Modal
	task_column_badge: 'Column: {column}',
	task_criteria_done: '{completed} / {total} done',
	task_created_label: 'Created:',
	task_updated_label: 'Updated:',
	task_target_label: 'Target:',
	task_specification: 'Specification',
	task_edit_spec: 'Edit Spec',
	task_save_spec: 'Save Spec',
	task_spec_placeholder: 'Task markdown body and acceptance criteria...',
	task_no_body: 'No body specification provided.',
	task_add_note: 'Add Note',
	task_adding_note: 'Adding...',
	task_note_hint: 'Markdown supported, use - [ ] for checklists',
	task_note_placeholder:
		'Add progress notes or follow-up criteria (e.g. - [ ] Check edge cases)...',
	task_dependencies_count: 'Dependencies ({count})',
	task_dependency_missing: 'missing',
	task_confirm_delete: 'Are you sure you want to delete task "{title}"?',
	task_delete_failed: 'Failed to delete task: {error}',
	task_delete: 'Delete',
	task_deleting: 'Deleting...',
	task_edit: 'Edit Task',

	// Task Edit & Create Modals
	task_create_new: 'Create New Task',
	task_create: 'Create Task',
	task_creating: 'Creating...',
	task_save_changes: 'Save Changes',
	task_title_required: 'Title is required',
	task_summary_required: 'Summary is required',
	task_field_title: 'Title',
	task_field_summary: 'Summary',
	task_field_column: 'Column',
	task_field_priority: 'Priority',
	task_field_milestone: 'Milestone',
	task_field_target_at: 'Target Date & Time',
	task_field_tags: 'Tags (comma separated)',
	task_field_dependencies: 'Dependencies (IDs comma separated)',
	task_field_body: 'Body & Acceptance Criteria (Markdown)',
	task_body_locked: 'Locked in {status} (use Notes for updates)',
	task_title_placeholder: 'e.g., Implement OAuth2 flow',
	task_summary_placeholder: 'Concise summary of work and scope',
	task_create_body_placeholder:
		'Enter task description or acceptance criteria (Markdown)...',
	task_edit_body_placeholder:
		'## Acceptance Criteria\n- [ ] Criterion 1\n- [ ] Criterion 2',

	// Column Detail Modal
	column_status_id: 'status: {id}',
	column_handled_by: 'Assigned Role / Handled By',
	column_assigned: 'Assigned',
	column_assigned_text:
		'Tasks in this column are designated for handling by {handler}.',
	column_unassigned: 'Unassigned',
	column_unassigned_text: 'Open to any team member or AI agent persona.',
	column_instructions: 'Workflow Guidance & Instructions',
	column_no_instructions:
		'No column-specific workflow instructions configured. You can set {instructions} under {section} in config.',
	column_policies: 'Column Policies',
	column_task_creation: 'Task Creation',
	column_creation_allowed: 'Allowed',
	column_creation_restricted: 'Restricted',
	column_creation_default: 'Default',
	column_creation_default_text: 'Initial column for newly created tasks.',
	column_creation_allowed_text:
		'New tasks can be created directly in this column.',
	column_creation_restricted_text: 'Tasks cannot be created directly here.',
	column_spec_editing: 'Specification Editing',
	column_editable: 'Editable',
	column_locked: 'Locked',
	column_editable_text: 'Full task specification body is editable.',
	column_locked_text: 'Body is locked; only append notes are allowed.',
	column_mcp_prompt: 'Global MCP System Prompt',

	// About Modal
	about_tagline: 'Local, Markdown-driven Kanban & Task Management',
	about_tab_project: 'About Project',
	about_tab_licenses: 'Open Source Licenses',
	about_overview_title: 'Zero-Dependency, Tasks-as-Code Architecture',
	about_overview_text:
		'Jokateko operates on a "Spec-First" philosophy, where version-controlled Markdown files inside your repository are the single source of truth. It compiles down to a single auditable, zero-CGO binary with embedded Preact and Model Context Protocol (MCP) support.',
	about_github_repo: 'GitHub Repository',
	about_platform: 'Platform: {platform}',
	about_cli_title: 'Terminal CLI Commands',
	about_cli_text:
		'You can also explore build metadata and licenses directly from your command line:',
	about_project_license: 'Project License (MIT)',
	about_osi_approved: 'OSI Approved',
	about_copy_license: 'Copy License',
	about_license_search_placeholder: 'Search package name, license type...',
	about_filter_all: 'All ({count})',
	about_filter_go: 'Go ({count})',
	about_filter_web: 'Web ({count})',
	about_view_license: 'View License',
	about_hide_license: 'Hide License',
	about_full_license_text: 'Full License Text',
	about_no_dependencies_match: 'No dependencies match "{query}"',

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
	arial_filter_by_state: 'Filter by state',
	arial_filter_by_priority: 'Filter by priority',
	arial_filter_by_tags: 'Filter by tags',
	arial_reset_all_filters: 'Reset all filters',
	arial_sort_tasks: 'Sort tasks',
	arial_sort_options: 'Sort options',
	arial_column_quick_nav: 'Column quick navigation',
	arial_kanban_columns: 'Kanban columns',
	arial_milestones_roadmap: 'Milestones roadmap',
	arial_calendar_nav: 'Calendar navigation',
	arial_week_number: 'Week number',
	arial_prev_month: 'Previous period',
	arial_next_month: 'Next period',
	arial_today_btn: 'Jump to current day',
	arial_month_view: 'Switch to month view',
	arial_week_view: 'Switch to week view',
	arial_week_day_nav: 'Week days navigation',
	arial_footer: 'Application footer',
	arial_github_repo: 'Jokateko GitHub repository',
	arial_close_modal: 'Close modal',

	// Task Detail Modal aria & titles
	arial_task_details: 'Task Details: {title}',
	arial_criteria_progress: 'Acceptance criteria progress',
	arial_filter_by_milestone: 'Filter board by milestone: {milestone}',
	arial_go_to_milestone: 'Go to milestone {milestone}',
	arial_id_copied: 'Copied to clipboard!',
	arial_click_to_copy_id: 'Click to copy ID',
	arial_copy_task_id: 'Copy task ID {id}',
	arial_restore_size: 'Restore size',
	arial_expand_modal: 'Expand modal',
	arial_maximize_modal: 'Maximize modal',
	arial_close_task_details: 'Close task details',
	arial_cancel_edit_spec: 'Cancel editing specification',
	arial_edit_spec: 'Edit specification',
	arial_spec_editor: 'Task specification markdown editor',
	arial_add_note_input: 'Add note text input',
	arial_open_dependency: 'Open dependent task {title}',
	arial_delete_task: 'Delete this task',
	arial_edit_task: 'Edit this task',

	// Task Edit Modal aria
	arial_edit_task_dialog: 'Edit Task: {title}',
	arial_close_edit_modal: 'Close edit modal',
	arial_task_title: 'Task title',
	arial_task_summary: 'Task summary',
	arial_select_column: 'Select column',
	arial_select_priority: 'Select priority',
	arial_select_milestone: 'Select milestone',
	arial_target_at: 'Target date and time',
	arial_task_tags: 'Task tags',
	arial_task_dependencies: 'Task dependencies',
	arial_task_body: 'Task markdown body',
	arial_cancel_editing: 'Cancel editing',
	arial_save_changes: 'Save changes',

	// Create Task Modal aria
	arial_create_task_dialog: 'Create New Task',
	arial_close_create_modal: 'Close create modal',
	arial_new_task_title: 'New task title',
	arial_new_task_summary: 'New task summary',
	arial_select_initial_column: 'Select initial column',
	arial_select_initial_priority: 'Select initial priority',
	arial_select_target_at: 'Select target date and time',
	arial_new_task_tags: 'New task tags',
	arial_new_task_dependencies: 'New task dependencies',
	arial_new_task_body: 'New task markdown body',
	arial_cancel_create: 'Cancel task creation',
	arial_create_task: 'Create task',

	// Column Detail Modal aria
	arial_column_details: 'Column details: {name}',

	// About Modal aria & titles
	arial_about_dialog: 'About Jokateko and Open Source Licenses',
	arial_open_repository: 'Open repository',
	arial_open_repository_for: 'Open repository for {name}',

	// Board, Column & Task Card aria & titles
	arial_open_task: 'Open task: {title}, Priority: {priority}',
	arial_blocked: 'Blocked',
	arial_blocked_by: 'Blocked by unfinished dependencies: {dependencies}',
	arial_checklist_progress: 'Checklist progress',
	arial_milestone_badge: 'Milestone: {milestone}',
	arial_filter_by_milestone_card: 'Filter tasks by milestone {title}',
	arial_milestone_ready: 'Milestone ready',
	arial_scroll_to_column: 'Scroll to column {name} ({count} tasks)',
	arial_kanban_column: 'Kanban column: {name}',
	arial_column_info: 'Column info: {name}',
	arial_column_info_title:
		'Click to view workflow guidance and prompt for {name}',
	arial_column_task_count: '{count} tasks in {name}',
	arial_add_task_to_column: 'Add new task to {name}',
	arial_sort_column: 'Sort tasks: {name}',
	arial_sort_by_title: 'Sort: {sort}',
	arial_column_tasks: 'Tasks in {name}',

	// Calendar aria & titles
	arial_calendar_day: '{day} {date} ({count} tasks)',
	arial_week_title: 'Week {week}',
	arial_milestone_progress: '{title} ({percent}%)',
	arial_view_week_tasks_date: 'View all tasks in week view {date}',
	arial_add_task_date: 'Add task for this date {date}',
	arial_task_status_title: '{title} ({status})',
	arial_more_tasks: '+{count} more',

	// Strategies & Glossary aria & titles
	arial_filter_strategies_by: 'Filter strategies by {tier}',
	arial_copy_strategy_id: 'Copy strategy ID {id}',
	arial_copy_strategy_link_title: 'Copy link to strategy',
	arial_copy_strategy_link: 'Copy link to strategy {id}',
	arial_link_copied: 'Link copied to clipboard!',
	arial_strategy: 'Strategy: {title}',
	arial_copy_term_link_title: 'Copy link to term',
	arial_copy_term_link: 'Copy link to glossary term {title}',
	arial_copy_term_id: 'Copy glossary ID {id}',
	arial_glossary_term: 'Glossary term: {title}',

	// Badges, Filters & Status aria & titles
	arial_priority: 'Priority: {priority}',
	arial_filter_by_tag: 'Filter by tag {tag}',
	arial_filter_state: 'Filter state {name}',
	arial_filter_priority: 'Filter priority {name}',
	arial_target_date: 'Target date',
	arial_target_title: 'Target: {date}',
	arial_target_overdue_title: 'Target: {date} (Overdue)',
	arial_target_approaching_title: 'Target: {date} (Approaching)',
	arial_about_title: 'About & Licenses',
	arial_live_connected: 'Live daemon connected',
	arial_live_reconnecting: 'Reconnecting to daemon...',
	arial_show_issue_details: 'Show issue details',
	arial_hide_issue_details: 'Hide issue details',
	arial_dismiss_schema_warning: 'Dismiss schema warning banner',
} as const

export type TranslationKey = keyof typeof defaultTranslations

/**
 * Returns the configured project locale as a canonical BCP 47 tag for Intl
 * date formatting, or undefined to use the browser locale when the project
 * locale is unset or not a valid tag.
 */
export function uiLocale(): string | undefined {
	const tag = config.value.project?.locale?.trim()
	if (!tag) return undefined
	try {
		return Intl.getCanonicalLocales(tag)[0]
	} catch {
		return undefined
	}
}

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

/**
 * Splits the localized text for key around {name} placeholders and puts the
 * matching vars in between, so callers can interpolate strings or JSX nodes in
 * whatever word order the translation uses. Placeholders without a var are
 * kept as written.
 */
export function tParts<T>(
	key: TranslationKey,
	vars: Record<string, T>,
): (string | T)[] {
	const text = t(key)
	const parts: (string | T)[] = []
	let last = 0
	for (const m of text.matchAll(/\{(\w+)\}/g)) {
		if (!(m[1] in vars)) continue
		if (m.index > last) parts.push(text.slice(last, m.index))
		parts.push(vars[m[1]])
		last = m.index + m[0].length
	}
	if (last < text.length) parts.push(text.slice(last))
	return parts
}

/**
 * Returns the localized text for key with {name} placeholders replaced by vars.
 */
export function tf(
	key: TranslationKey,
	vars: Record<string, string | number>,
): string {
	return tParts(key, vars).join('')
}

/**
 * Keeps <html lang> in sync with the project locale, so screen readers and
 * the browser use the same language as the dates. Without a project locale
 * the page keeps its original lang. Returns a function that stops syncing.
 */
export function initDocumentLang(): () => void {
	const root = document.documentElement
	const initialLang = root.lang
	return effect(() => {
		root.lang = uiLocale() ?? initialLang
	})
}
