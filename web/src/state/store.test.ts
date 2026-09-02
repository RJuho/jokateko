import { describe, expect, it } from 'bun:test'
import {
	allTags,
	columnTasks,
	config,
	configuredPriorities,
	filteredTasks,
	initFromSnapshot,
	mode,
	removeTask,
	resetFilters,
	setMilestoneFilter,
	setSearchQuery,
	tasks,
	togglePriorityFilter,
	toggleTagFilter,
	upsertTask,
} from './store'

describe('Preact Signals State Store', () => {
	it('initializes cleanly from snapshot', () => {
		initFromSnapshot({
			config: {
				project: { name: 'Snapshot Board', description: '' },
				board: {
					columns: [
						{ id: 'col1', name: 'Col 1', color: '#111' },
						{ id: 'col2', name: 'Col 2', color: '#222' },
					],
				},
			},
			tasks: [
				{
					id: 't-1',
					title: 'First Task',
					status: 'col1',
					priority: 'high',
					tags: ['frontend', 'ui'],
					summary: 'Test summary',
					dependencies: [],
					body: '',
					total_criteria: 0,
					completed_criteria: 0,
				},
			],
			milestones: [],
			strategies: [],
			glossary: [],
		})

		expect(mode.value).toBe('static')
		expect(config.value.project.name).toBe('Snapshot Board')
		expect(tasks.value.length).toBe(1)
		expect(tasks.value[0].title).toBe('First Task')
	})

	it('upserts and removes tasks reactively', () => {
		upsertTask({
			id: 't-2',
			title: 'Second Task',
			status: 'col2',
			priority: 'medium',
			tags: ['backend'],
			summary: 'Backend task',
			dependencies: [],
			body: '',
			total_criteria: 0,
			completed_criteria: 0,
		})

		expect(tasks.value.length).toBe(2)
		expect(tasks.value.find((t) => t.id === 't-2')?.title).toBe('Second Task')

		// Update existing
		upsertTask({
			id: 't-2',
			title: 'Second Task Updated',
			status: 'col2',
			priority: 'critical',
			tags: ['backend', 'urgent'],
			summary: 'Backend task updated',
			dependencies: [],
			body: '',
			total_criteria: 0,
			completed_criteria: 0,
		})

		expect(tasks.value.length).toBe(2)
		expect(tasks.value.find((t) => t.id === 't-2')?.title).toBe(
			'Second Task Updated',
		)
		expect(tasks.value.find((t) => t.id === 't-2')?.priority).toBe('critical')

		// Remove task
		removeTask('t-2')
		expect(tasks.value.length).toBe(1)
		expect(tasks.value.find((t) => t.id === 't-2')).toBeUndefined()
	})

	it('filters tasks by search query, tags, priority, and milestone', () => {
		resetFilters()
		tasks.value = [
			{
				id: 't-alpha',
				title: 'Alpha Login',
				status: 'col1',
				priority: 'high',
				tags: ['auth', 'security'],
				milestone: 'm1',
				summary: 'User auth',
				dependencies: [],
				body: 'OAuth2 and session cookies specification',
				total_criteria: 0,
				completed_criteria: 0,
			},
			{
				id: 't-beta',
				title: 'Beta Dashboard',
				status: 'col1',
				priority: 'low',
				tags: ['ui'],
				milestone: 'm2',
				summary: 'Admin dashboard',
				dependencies: [],
				body: 'React and Preact components',
				total_criteria: 0,
				completed_criteria: 0,
			},
		]

		// 1. Search Query across title, body, and tags
		setSearchQuery('login')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('t-alpha')

		setSearchQuery('cookies')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('t-alpha')

		setSearchQuery('security')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('t-alpha')

		resetFilters()

		// 2. Tag Filter
		toggleTagFilter('ui')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('t-beta')

		resetFilters()

		// 3. Priority Filter
		togglePriorityFilter('high')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('t-alpha')

		resetFilters()

		// 4. Milestone Filter
		setMilestoneFilter('m2')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('t-beta')

		resetFilters()
		expect(filteredTasks.value.length).toBe(2)
	})

	it('computes columnTasks groupings correctly', () => {
		resetFilters()
		config.value = {
			...config.value,
			board: {
				columns: [
					{ id: 'col1', name: 'Col 1', color: '#111' },
					{ id: 'col2', name: 'Col 2', color: '#222' },
				],
			},
		}

		tasks.value = [
			{
				id: 't1',
				title: 'Task 1',
				status: 'col1',
				priority: 'medium',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
			},
			{
				id: 't2',
				title: 'Task 2',
				status: 'col2',
				priority: 'medium',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
			},
		]

		const grouped = columnTasks.value
		expect(grouped.col1.length).toBe(1)
		expect(grouped.col2.length).toBe(1)
		expect(grouped.col1[0].id).toBe('t1')
		expect(grouped.col2[0].id).toBe('t2')
	})

	it('computes unique sorted allTags list', () => {
		tasks.value = [
			{
				id: 't1',
				title: 'T1',
				status: 'col1',
				priority: 'low',
				tags: ['zebra', 'apple'],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
			},
			{
				id: 't2',
				title: 'T2',
				status: 'col1',
				priority: 'low',
				tags: ['banana', 'apple'],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
			},
		]

		expect(allTags.value).toEqual(['apple', 'banana', 'zebra'])
	})

	it('computes configuredPriorities from config or falls back to default', () => {
		config.value = {
			...config.value,
			priorities: undefined,
		}
		expect(configuredPriorities.value.map((p) => p.id)).toEqual([
			'critical',
			'high',
			'medium',
			'low',
		])

		config.value = {
			...config.value,
			priorities: [
				{ id: 'p0', name: 'P0 - Blocker', color: '#ff0000' },
				{ id: 'p1', name: 'P1 - High', color: '#ff8800' },
			],
		}
		expect(configuredPriorities.value).toEqual([
			{ id: 'p0', name: 'P0 - Blocker', color: '#ff0000' },
			{ id: 'p1', name: 'P1 - High', color: '#ff8800' },
		])
	})
})
