import { describe, expect, it } from 'bun:test'
import * as v from 'valibot'
import { sampleSnapshot } from '../fixtures/sampleData'
import { SnapshotSchema } from '../schemas/models'
import {
	activeStrategyId,
	columnTasks,
	config,
	filteredTasks,
	initFromSnapshot,
	milestones,
	mode,
	resetFilters,
	setMilestoneFilter,
	setSearchQuery,
	strategies,
	tasks,
	togglePriorityFilter,
	toggleStateFilter,
	toggleTagFilter,
} from '../state/store'
import { formatBrowserDateTime } from '../utils/date'

describe('Sample Data & Component Logic Verification', () => {
	it('validates that sampleSnapshot conforms strictly to Valibot SnapshotSchema', () => {
		const result = v.safeParse(SnapshotSchema, sampleSnapshot)
		expect(result.success).toBe(true)
		if (result.success) {
			expect(result.output.tasks.length).toBeGreaterThanOrEqual(200)
			expect(result.output.milestones.length).toBeGreaterThanOrEqual(6)
			expect(result.output.strategies.length).toBeGreaterThanOrEqual(10)
			expect(result.output.glossary.length).toBeGreaterThanOrEqual(14)
		}
	})

	it('initializes store with sampleSnapshot and partitions tasks into Kanban columns', () => {
		initFromSnapshot(sampleSnapshot)

		expect(mode.value).toBe('static')
		expect(config.value.project.name).toBe('Jokateko Kanban')
		expect(tasks.value.length).toBeGreaterThanOrEqual(200)
		expect(milestones.value.length).toBeGreaterThanOrEqual(6)
		expect(strategies.value.length).toBeGreaterThanOrEqual(10)

		const cols = columnTasks.value
		expect(cols.in_progress.length).toBeGreaterThan(0)
		expect(cols.ready.length).toBeGreaterThan(0)
		expect(cols.backlog.length).toBeGreaterThan(0)
		expect(cols.done.length).toBeGreaterThan(0)
	})

	it('filters sample data accurately by query, tag, priority, and milestone', () => {
		initFromSnapshot(sampleSnapshot)

		// 1. Text search
		setSearchQuery('oauth2')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('260901-user-auth')

		// Clear search
		setSearchQuery('')
		expect(filteredTasks.value.length).toBe(tasks.value.length)

		// 2. Tag filter
		toggleTagFilter('security')
		expect(filteredTasks.value.length).toBe(17)

		// 3. Priority filter
		togglePriorityFilter('critical')
		expect(filteredTasks.value.length).toBe(5)

		// 4. Milestone filter
		setMilestoneFilter('m1-mvp-release')
		expect(filteredTasks.value.length).toBe(2)

		// 5. State filter
		toggleStateFilter('in_progress')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('260901-user-auth')
		toggleStateFilter('in_progress') // toggle off

		toggleStateFilter('backlog')
		expect(filteredTasks.value.length).toBe(0)

		resetFilters()
	})

	it('formats ISO datetime into browser localized format', () => {
		const formatted = formatBrowserDateTime('2026-09-02T05:30:00Z')
		expect(formatted.length).toBeGreaterThan(0)
		expect(formatted).toContain('2026')
		expect(formatBrowserDateTime('')).toBe('')
		expect(formatBrowserDateTime(null)).toBe('')
	})

	it('formats ISO datetime in an explicit locale', () => {
		expect(formatBrowserDateTime('2026-09-02T12:00:00Z', 'fi-FI')).toContain(
			'2.9.2026',
		)
		expect(formatBrowserDateTime('2026-09-02T12:00:00Z', 'en-US')).toContain(
			'Sep 2, 2026',
		)
	})

	it('clearing search query when milestone is selected shows all milestone tasks', () => {
		initFromSnapshot(sampleSnapshot)

		// Suppose user searched for a milestone name e.g. "Release" which doesn't match task titles
		setSearchQuery('Release')
		setMilestoneFilter('m1-mvp-release')
		expect(filteredTasks.value.length).toBe(0)

		// Clearing search query (as done when selecting milestone from search)
		setSearchQuery('')
		const expectedCount = sampleSnapshot.tasks.filter(
			(t) => t.milestone === 'm1-mvp-release',
		).length
		expect(filteredTasks.value.length).toBe(expectedCount)
		expect(
			filteredTasks.value.every((t) => t.milestone === 'm1-mvp-release'),
		).toBe(true)
		resetFilters()
	})

	it('locates URL targeted strategy accurately in sample data', () => {
		initFromSnapshot(sampleSnapshot)
		activeStrategyId.value = 'strat-mobile-first-layout'
		const target = strategies.value.find((s) => s.id === activeStrategyId.value)
		expect(target).toBeDefined()
		expect(target?.tier).toBe(2)
		expect(target?.title).toContain('Mobile-First')
	})
})
