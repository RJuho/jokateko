import { describe, expect, it } from 'bun:test'
import { readdirSync, readFileSync } from 'node:fs'
import { config } from '../state/store'
import { defaultTranslations, t, tf, tParts, uiLocale } from './i18n'

describe('i18n Translations System', () => {
	it('returns default English translation when translations config is undefined or empty', () => {
		config.value = {
			...config.value,
			translations: undefined,
		}

		expect(t('board')).toBe('Board')
		expect(t('strategies')).toBe('Strategies')
		expect(t('glossary')).toBe('Glossary')
		expect(t('search')).toBe('Search')
		expect(t('tasks')).toBe('tasks')
		expect(t('architectural_strategies')).toBe('Architectural Strategies')
		expect(t('strategies_subtitle')).toBe(
			'Tiered guidelines and system design rules using progressive disclosure',
		)
		expect(t('tiers')).toBe('Tiers')
		expect(t('all_tiers')).toBe('All Tiers')
		expect(t('project_glossary')).toBe('Project Glossary')
		expect(t('glossary_subtitle')).toBe(
			'Standardized domain definitions and terminology dictionary',
		)
		expect(t('footer_text')).toBe('Build with ❤️ in 🇪🇺 with 🤖')
	})

	it('returns configured translation when provided in config.translations', () => {
		config.value = {
			...config.value,
			translations: {
				board: 'Taulu',
				strategies: 'Strategiat',
				glossary: 'Sanasto',
				search: 'Etsi',
				tasks: 'askaretta',
				footer_text: 'Rakennettu rakkaudella',
				arial_main_nav: 'Päänavigaatio',
			},
		}

		expect(t('board')).toBe('Taulu')
		expect(t('strategies')).toBe('Strategiat')
		expect(t('glossary')).toBe('Sanasto')
		expect(t('search')).toBe('Etsi')
		expect(t('tasks')).toBe('askaretta')
		expect(t('footer_text')).toBe('Rakennettu rakkaudella')
		expect(t('arial_main_nav')).toBe('Päänavigaatio')

		// Untranslated keys should still fall back to English
		expect(t('architectural_strategies')).toBe('Architectural Strategies')
		expect(t('tiers')).toBe('Tiers')
		expect(t('all_tiers')).toBe('All Tiers')
	})

	it('supports all arial aria keys with English defaults', () => {
		config.value = {
			...config.value,
			translations: undefined,
		}

		expect(t('arial_main_nav')).toBe('Main Navigation')
		expect(t('arial_mobile_nav')).toBe('Mobile Navigation')
		expect(t('arial_mobile_menu')).toBe('Mobile navigation menu')
		expect(t('arial_open_menu')).toBe('Open menu')
		expect(t('arial_close_menu')).toBe('Close menu')
		expect(t('arial_search')).toBe('Site search')
		expect(t('arial_search_input')).toBe(
			'Search tasks, milestones, strategies, and glossary',
		)
		expect(t('arial_search_results')).toBe('Search results')
		expect(t('arial_theme_toggle')).toBe('Toggle light and dark theme')
		expect(t('arial_theme_dark')).toBe('Toggle dark theme')
		expect(t('arial_theme_light_label')).toBe('Light theme')
		expect(t('arial_theme_dark_label')).toBe('Dark theme')
		expect(t('arial_filter_tasks')).toBe('Filter tasks')
		expect(t('arial_filter_by_priority')).toBe('Filter by priority')
		expect(t('arial_filter_by_tags')).toBe('Filter by tags')
		expect(t('arial_reset_all_filters')).toBe('Reset all filters')
		expect(t('arial_column_quick_nav')).toBe('Column quick navigation')
		expect(t('arial_kanban_columns')).toBe('Kanban columns')
		expect(t('arial_milestones_roadmap')).toBe('Milestones roadmap')
		expect(t('arial_footer')).toBe('Application footer')
		expect(t('arial_github_repo')).toBe('Jokateko GitHub repository')
	})

	it('returns fallback parameter if key is unknown and fallback is supplied', () => {
		expect(t('non_existent_key', 'Custom Fallback')).toBe('Custom Fallback')
	})

	it('provides English defaults for modal strings', () => {
		config.value = { ...config.value, translations: undefined }

		expect(t('task_edit')).toBe('Edit Task')
		expect(t('task_create_new')).toBe('Create New Task')
		expect(t('task_title_required')).toBe('Title is required')
		expect(t('task_edit_body_placeholder')).toBe(
			'## Acceptance Criteria\n- [ ] Criterion 1\n- [ ] Criterion 2',
		)
		expect(t('column_policies')).toBe('Column Policies')
		expect(t('about_tab_licenses')).toBe('Open Source Licenses')
		expect(t('arial_close_modal')).toBe('Close modal')
		expect(t('cancel')).toBe('Cancel')
	})

	it('lets config.translations override modal strings', () => {
		config.value = {
			...config.value,
			translations: {
				task_edit: 'Muokkaa tehtävää',
				arial_close_modal: 'Sulje',
				task_dependencies_count: 'Riippuvuudet: {count}',
			},
		}

		expect(t('task_edit')).toBe('Muokkaa tehtävää')
		expect(t('arial_close_modal')).toBe('Sulje')
		expect(tf('task_dependencies_count', { count: 3 })).toBe('Riippuvuudet: 3')
		expect(t('task_create')).toBe('Create Task')
	})
})

describe('i18n interpolation', () => {
	it('tf fills {name} placeholders in the default text', () => {
		config.value = { ...config.value, translations: undefined }

		expect(tf('arial_task_details', { title: 'Fix login' })).toBe(
			'Task Details: Fix login',
		)
		expect(tf('task_criteria_done', { completed: 2, total: 5 })).toBe(
			'2 / 5 done',
		)
	})

	it('tf follows the word order of a configured translation', () => {
		config.value = {
			...config.value,
			translations: { task_column_badge: '{column} -sarake' },
		}

		expect(tf('task_column_badge', { column: 'Valmis' })).toBe('Valmis -sarake')
	})

	it('tf keeps placeholders that have no value', () => {
		config.value = { ...config.value, translations: undefined }

		expect(tf('task_criteria_done', { completed: 1 })).toBe('1 / {total} done')
	})

	it('tParts places non-string values between text segments', () => {
		config.value = { ...config.value, translations: undefined }
		const node = { node: true }

		expect(tParts('column_assigned_text', { handler: node })).toEqual([
			'Tasks in this column are designated for handling by ',
			node,
			'.',
		])
	})
})

describe('screen-reader labels', () => {
	it('fills placeholders in the English defaults', () => {
		config.value = { ...config.value, translations: undefined }

		expect(tf('arial_calendar_day', { day: 'Mon', date: 5, count: 2 })).toBe(
			'Mon 5 (2 tasks)',
		)
		expect(tf('arial_open_task', { title: 'Fix', priority: 'high' })).toBe(
			'Open task: Fix, Priority: high',
		)
		expect(tf('arial_target_overdue_title', { date: '2026-09-01' })).toBe(
			'Target: 2026-09-01 (Overdue)',
		)
		expect(t('arial_dismiss_schema_warning')).toBe(
			'Dismiss schema warning banner',
		)
	})

	it('can be translated with a different word order', () => {
		config.value = {
			...config.value,
			translations: { arial_scroll_to_column: '{count} tehtävää: {name}' },
		}

		expect(tf('arial_scroll_to_column', { name: 'Valmis', count: 3 })).toBe(
			'3 tehtävää: Valmis',
		)
	})

	it('has no hard-coded English labels left in components', () => {
		const dir = new URL('../components/', import.meta.url)
		const files = readdirSync(dir, {
			recursive: true,
			encoding: 'utf8',
		}).filter((f) => f.endsWith('.tsx'))
		// aria-label/title/alt given as a string literal or template literal,
		// or an SVG <title> with literal text
		const literal = /(aria-label|title|alt)=('|"|\{`)|<title>[A-Za-z]/
		const offenders = files.flatMap((f) =>
			readFileSync(new URL(f, dir), 'utf8')
				.split('\n')
				.map((line, i) => ({ line: line.trim(), at: `${f}:${i + 1}` }))
				.filter(({ line }) => literal.test(line))
				.map(({ at, line }) => `${at} ${line}`),
		)
		expect(offenders).toEqual([])
	})
})

describe('uiLocale', () => {
	const withLocale = (locale?: string) => {
		config.value = {
			...config.value,
			project: { ...config.value.project, locale },
		}
	}

	it('is undefined (browser locale) when the project locale is unset or blank', () => {
		withLocale(undefined)
		expect(uiLocale()).toBeUndefined()
		withLocale('  ')
		expect(uiLocale()).toBeUndefined()
	})

	it('returns the canonical form of a valid tag', () => {
		withLocale('fi-fi')
		expect(uiLocale()).toBe('fi-FI')
		withLocale(' en-GB ')
		expect(uiLocale()).toBe('en-GB')
	})

	it('falls back to the browser locale for an invalid tag', () => {
		withLocale('fi_FI')
		expect(uiLocale()).toBeUndefined()
		withLocale(undefined)
	})
})

describe('default.toml parity', () => {
	const toml = Bun.TOML.parse(
		readFileSync(
			new URL('../../../internal/config/default.toml', import.meta.url),
			'utf8',
		),
	) as { translations: Record<string, string> }

	it('declares exactly the keys of defaultTranslations', () => {
		expect(Object.keys(toml.translations).sort()).toEqual(
			Object.keys(defaultTranslations).sort(),
		)
	})

	it('uses the same English defaults', () => {
		expect(toml.translations).toEqual({ ...defaultTranslations })
	})
})
