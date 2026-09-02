import { describe, expect, it } from 'bun:test'
import { config } from '../state/store'
import { t } from './i18n'

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
})
