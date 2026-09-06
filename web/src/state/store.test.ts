import { describe, expect, it } from 'bun:test'
import {
	activeSortMode,
	allTags,
	columnSortModes,
	columnTasks,
	config,
	configuredPriorities,
	configuredTiers,
	filteredTasks,
	getColumnSortMode,
	initFromSnapshot,
	mode,
	removeTask,
	resetFilters,
	setColumnSortMode,
	setMilestoneFilter,
	setSearchQuery,
	setSortMode,
	tasks,
	togglePriorityFilter,
	toggleStateFilter,
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

		// 5. State Filter
		toggleStateFilter('col1')
		expect(filteredTasks.value.length).toBe(2)
		toggleStateFilter('non_existent')
		expect(filteredTasks.value.length).toBe(2)
		resetFilters()
		toggleStateFilter('non_existent')
		expect(filteredTasks.value.length).toBe(0)

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

	it('computes configuredTiers from config or falls back to default', () => {
		config.value = {
			...config.value,
			tiers: undefined,
		}
		expect(configuredTiers.value.map((t) => t.id)).toEqual(['1', '2', '3'])
		expect(configuredTiers.value[0].name).toBe('Tier 1')

		config.value = {
			...config.value,
			tiers: [
				{
					id: 'core',
					name: 'Core',
					title: 'Invariants',
					summary: 'Zero-CGO rules',
					color: '#3b82f6',
				},
				{
					id: 'extended',
					name: 'Extended',
					title: 'Details',
					summary: 'Implementation rules',
					color: '#10b981',
				},
			],
		}
		expect(configuredTiers.value.map((t) => t.id)).toEqual(['core', 'extended'])
		expect(configuredTiers.value[0].title).toBe('Invariants')
	})

	it('reactively sorts columnTasks based on activeSortMode and resets to default', () => {
		config.value = {
			...config.value,
			board: {
				...config.value.board,
				columns: [
					{ id: 'backlog', name: 'Backlog', color: '#94a3b8' },
					{ id: 'in_progress', name: 'In Progress', color: '#fbbf24' },
				],
			},
		}

		tasks.value = [
			{
				id: 't-1',
				title: 'Zebra',
				status: 'backlog',
				priority: 'low',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
				changed_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 't-2',
				title: 'Apple',
				status: 'backlog',
				priority: 'critical',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
				changed_at: '2026-09-02T00:00:00Z',
			},
			{
				id: 't-3',
				title: 'Mango',
				status: 'backlog',
				priority: 'medium',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
				changed_at: '2026-09-03T00:00:00Z',
			},
		]

		// 1. Default sort: Priority critical (t-2) > medium (t-3) > low (t-1)
		setSortMode('default')
		expect(columnTasks.value.backlog.map((t) => t.id)).toEqual([
			't-2',
			't-3',
			't-1',
		])

		// 2. Alphabetical sort: Apple (t-2) > Mango (t-3) > Zebra (t-1)
		setSortMode('title')
		expect(columnTasks.value.backlog.map((t) => t.id)).toEqual([
			't-2',
			't-3',
			't-1',
		])

		// 3. Recently changed sort: t-3 (Sep 03) > t-2 (Sep 02) > t-1 (Sep 01)
		setSortMode('changed_at')
		expect(columnTasks.value.backlog.map((t) => t.id)).toEqual([
			't-3',
			't-2',
			't-1',
		])

		// 4. Reset filters resets activeSortMode to default
		resetFilters()
		expect(activeSortMode.value).toBe('default')
		expect(columnTasks.value.backlog.map((t) => t.id)).toEqual([
			't-2',
			't-3',
			't-1',
		])
	})

	it('supports independent per-column sorting via columnSortModes', () => {
		config.value = {
			...config.value,
			board: {
				...config.value.board,
				columns: [
					{ id: 'backlog', name: 'Backlog', color: '#94a3b8' },
					{ id: 'ready', name: 'Ready', color: '#60a5fa' },
				],
			},
		}

		tasks.value = [
			{
				id: 'b-1',
				title: 'Zebra',
				status: 'backlog',
				priority: 'low',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
				changed_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 'b-2',
				title: 'Apple',
				status: 'backlog',
				priority: 'critical',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
				changed_at: '2026-09-02T00:00:00Z',
			},
			{
				id: 'r-1',
				title: 'Beta',
				status: 'ready',
				priority: 'medium',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
				created_at: '2026-09-01T00:00:00Z',
			},
			{
				id: 'r-2',
				title: 'Alpha',
				status: 'ready',
				priority: 'high',
				tags: [],
				summary: '',
				dependencies: [],
				body: '',
				total_criteria: 0,
				completed_criteria: 0,
				created_at: '2026-09-03T00:00:00Z',
			},
		]

		// Initially both default
		expect(getColumnSortMode('backlog')).toBe('default')
		expect(getColumnSortMode('ready')).toBe('default')

		// Set backlog to 'changed_at' and ready to 'title'
		setColumnSortMode('backlog', 'changed_at')
		setColumnSortMode('ready', 'title')

		expect(getColumnSortMode('backlog')).toBe('changed_at')
		expect(getColumnSortMode('ready')).toBe('title')

		// Backlog sorted by changed_at desc: b-2 (Sep 02) > b-1 (Sep 01)
		expect(columnTasks.value.backlog.map((t) => t.id)).toEqual(['b-2', 'b-1'])

		// Ready sorted by title A-Z: Alpha (r-2) > Beta (r-1)
		expect(columnTasks.value.ready.map((t) => t.id)).toEqual(['r-2', 'r-1'])

		// Reset filters resets all column sorts
		resetFilters()
		expect(columnSortModes.value).toEqual({})
		expect(getColumnSortMode('backlog')).toBe('default')
		expect(getColumnSortMode('ready')).toBe('default')
	})
})
