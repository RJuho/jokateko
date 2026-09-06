import { describe, expect, it } from 'bun:test'
import type { Column } from '../schemas/models'
import { type SortableTask, compareTasks, getPriorityRank } from './sort'

describe('Multi-Criteria Task Sorting Utilities', () => {
	const defaultBacklogCol: Column = {
		id: 'backlog',
		name: 'Backlog',
		color: '#94a3b8',
	}

	const defaultInProgressCol: Column = {
		id: 'in_progress',
		name: 'In Progress',
		color: '#fbbf24',
	}

	const defaultDoneCol: Column = {
		id: 'done',
		name: 'Done',
		color: '#10b981',
	}

	const customCol: Column = {
		id: 'custom_col',
		name: 'Custom',
		color: '#a855f7',
		sort_by: 'created_at',
		sort_direction: 'desc',
	}

	it('ranks priorities in correct urgency order', () => {
		expect(getPriorityRank('critical')).toBe(0)
		expect(getPriorityRank('high')).toBe(1)
		expect(getPriorityRank('medium')).toBe(2)
		expect(getPriorityRank('low')).toBe(3)
		expect(getPriorityRank(undefined)).toBe(2)
	})

	it('sorts backlog by priority then changed_at descending', () => {
		const tasks: SortableTask[] = [
			{
				id: 't-low',
				title: 'Low priority',
				status: 'backlog',
				priority: 'low',
				changed_at: '2026-09-05T10:00:00Z',
			},
			{
				id: 't-crit',
				title: 'Critical priority',
				status: 'backlog',
				priority: 'critical',
				changed_at: '2026-09-01T10:00:00Z',
			},
			{
				id: 't-high-older',
				title: 'High older',
				status: 'backlog',
				priority: 'high',
				changed_at: '2026-09-02T10:00:00Z',
			},
			{
				id: 't-high-newer',
				title: 'High newer',
				status: 'backlog',
				priority: 'high',
				changed_at: '2026-09-04T10:00:00Z',
			},
		]

		tasks.sort((a, b) => compareTasks(a, b, defaultBacklogCol, 'default'))

		expect(tasks.map((t) => t.id)).toEqual([
			't-crit',
			't-high-newer',
			't-high-older',
			't-low',
		])
	})

	it('sorts active column by priority then target_at ascending with fallback to changed_at descending', () => {
		const tasks: SortableTask[] = [
			{
				id: 't-crit',
				title: 'Critical',
				status: 'in_progress',
				priority: 'critical',
				target_at: '2026-09-20T00:00:00Z',
				changed_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 't-no-target-old',
				title: 'No target old',
				status: 'in_progress',
				priority: 'high',
				target_at: '',
				changed_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 't-target-later',
				title: 'Target later',
				status: 'in_progress',
				priority: 'high',
				target_at: '2026-09-18T00:00:00Z',
				changed_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 't-target-sooner',
				title: 'Target sooner',
				status: 'in_progress',
				priority: 'high',
				target_at: '2026-09-10T00:00:00Z',
				changed_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 't-no-target-new',
				title: 'No target new',
				status: 'in_progress',
				priority: 'high',
				target_at: '',
				changed_at: '2026-09-05T00:00:00Z',
			},
		]

		tasks.sort((a, b) => compareTasks(a, b, defaultInProgressCol, 'default'))

		expect(tasks.map((t) => t.id)).toEqual([
			't-crit',
			't-target-sooner',
			't-target-later',
			't-no-target-new',
			't-no-target-old',
		])
	})

	it('sorts done column by changed_at descending regardless of priority', () => {
		const tasks: SortableTask[] = [
			{
				id: 't-crit-old',
				title: 'Critical old',
				status: 'done',
				priority: 'critical',
				changed_at: '2026-09-01T10:00:00Z',
			},
			{
				id: 't-low-new',
				title: 'Low new',
				status: 'done',
				priority: 'low',
				changed_at: '2026-09-06T10:00:00Z',
			},
			{
				id: 't-med-mid',
				title: 'Medium mid',
				status: 'done',
				priority: 'medium',
				changed_at: '2026-09-04T10:00:00Z',
			},
		]

		tasks.sort((a, b) => compareTasks(a, b, defaultDoneCol, 'default'))

		expect(tasks.map((t) => t.id)).toEqual([
			't-low-new',
			't-med-mid',
			't-crit-old',
		])
	})

	it('supports custom sort_by and sort_direction from column configuration in default mode', () => {
		const tasks: SortableTask[] = [
			{
				id: 't1',
				title: 'T1',
				status: 'custom_col',
				priority: 'medium',
				created_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 't2',
				title: 'T2',
				status: 'custom_col',
				priority: 'medium',
				created_at: '2026-09-05T00:00:00Z',
			},
			{
				id: 't3',
				title: 'T3',
				status: 'custom_col',
				priority: 'medium',
				created_at: '2026-09-03T00:00:00Z',
			},
		]

		tasks.sort((a, b) => compareTasks(a, b, customCol, 'default'))

		expect(tasks.map((t) => t.id)).toEqual(['t2', 't3', 't1'])
	})

	it('applies global sort mode overrides', () => {
		const tasks: SortableTask[] = [
			{
				id: 't-c',
				title: 'Charlie',
				status: 'backlog',
				priority: 'low',
				target_at: '2026-09-12T00:00:00Z',
				changed_at: '2026-09-05T00:00:00Z',
				created_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 't-a',
				title: 'Alpha',
				status: 'backlog',
				priority: 'critical',
				target_at: '2026-09-15T00:00:00Z',
				changed_at: '2026-09-01T00:00:00Z',
				created_at: '2026-09-03T00:00:00Z',
			},
			{
				id: 't-b',
				title: 'Bravo',
				status: 'backlog',
				priority: 'medium',
				target_at: '2026-09-08T00:00:00Z',
				changed_at: '2026-09-03T00:00:00Z',
				created_at: '2026-09-02T00:00:00Z',
			},
		]

		// Priority mode: critical (t-a) > medium (t-b) > low (t-c)
		const byPriority = [...tasks].sort((a, b) =>
			compareTasks(a, b, defaultBacklogCol, 'priority'),
		)
		expect(byPriority.map((t) => t.id)).toEqual(['t-a', 't-b', 't-c'])

		// Target Date mode: soonest first -> t-b (Sep 08) > t-c (Sep 12) > t-a (Sep 15)
		const byTarget = [...tasks].sort((a, b) =>
			compareTasks(a, b, defaultBacklogCol, 'target_at'),
		)
		expect(byTarget.map((t) => t.id)).toEqual(['t-b', 't-c', 't-a'])

		// Recently Changed mode: changed_at desc -> t-c (Sep 05) > t-b (Sep 03) > t-a (Sep 01)
		const byChanged = [...tasks].sort((a, b) =>
			compareTasks(a, b, defaultBacklogCol, 'changed_at'),
		)
		expect(byChanged.map((t) => t.id)).toEqual(['t-c', 't-b', 't-a'])

		// Created Date mode: created_at desc -> t-a (Sep 03) > t-b (Sep 02) > t-c (Sep 01)
		const byCreated = [...tasks].sort((a, b) =>
			compareTasks(a, b, defaultBacklogCol, 'created_at'),
		)
		expect(byCreated.map((t) => t.id)).toEqual(['t-a', 't-b', 't-c'])

		// Alphabetical mode: Alpha (t-a) > Bravo (t-b) > Charlie (t-c)
		const byTitle = [...tasks].sort((a, b) =>
			compareTasks(a, b, defaultBacklogCol, 'title'),
		)
		expect(byTitle.map((t) => t.id)).toEqual(['t-a', 't-b', 't-c'])
	})
})
