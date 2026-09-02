import { describe, expect, it } from 'bun:test'
import * as v from 'valibot'
import {
	BoardStateSchema,
	type SnapshotInput,
	SnapshotSchema,
	TaskSchema,
} from './models'

describe('Valibot Models', () => {
	it('validates a complete task and applies defaults', () => {
		const rawTask = {
			id: '260901-test-task',
			title: 'Test Task',
			status: 'in_progress',
			summary: 'A test summary',
		}

		const result = v.safeParse(TaskSchema, rawTask)
		expect(result.success).toBe(true)
		if (result.success) {
			expect(result.output.priority).toBe('medium')
			expect(result.output.tags).toEqual([])
			expect(result.output.dependencies).toEqual([])
			expect(result.output.total_criteria).toBe(0)
		}
	})

	it('fails validation on non-string priority', () => {
		const rawTask = {
			id: 'task-bad',
			title: 'Bad Task',
			status: 'backlog',
			priority: 12345, // invalid type
		}

		const result = v.safeParse(TaskSchema, rawTask)
		expect(result.success).toBe(false)
	})

	it('validates custom priority strings', () => {
		const rawTask = {
			id: 'task-custom',
			title: 'Custom Priority Task',
			status: 'backlog',
			priority: 'ultra-high',
		}

		const result = v.safeParse(TaskSchema, rawTask)
		expect(result.success).toBe(true)
		if (result.success) {
			expect(result.output.priority).toBe('ultra-high')
		}
	})

	it('validates an entire Snapshot payload', () => {
		const rawSnapshot: SnapshotInput = {
			config: {
				project: {
					name: 'Jokateko',
					description: 'Markdown Kanban',
				},
				board: {
					columns: [
						{ id: 'backlog', name: 'Backlog', color: '#64748b' },
						{ id: 'ready', name: 'Ready', color: '#3b82f6' },
					],
				},
			},
			tasks: [
				{
					id: 'task-1',
					title: 'First',
					status: 'ready',
					priority: 'high',
					tags: ['core'],
					summary: 'Initial',
					dependencies: [],
				},
			],
			milestones: [
				{
					id: 'm1',
					title: 'Milestone 1',
					status: 'open',
					is_archived: false,
					tags: [],
					summary: 'Target 1',
					total_tasks: 1,
					completed_tasks: 0,
					progress_percentage: 0,
				},
			],
			strategies: [
				{
					id: 's1',
					title: 'Architecture',
					tier: 1,
					tags: [],
					summary: 'Core design',
				},
			],
			glossary: [
				{
					id: 'g1',
					title: 'Task',
					tags: [],
					summary: 'A unit of work',
				},
			],
		}

		const result = v.safeParse(SnapshotSchema, rawSnapshot)
		expect(result.success).toBe(true)
	})

	it('validates live REST BoardState payload', () => {
		const rawBoard = {
			project_name: 'Jokateko Live',
			columns: [
				{
					id: 'ready',
					name: 'Ready',
					color: '#3b82f6',
					tasks: [
						{
							id: 'task-2',
							title: 'Second',
							status: 'ready',
							priority: 'low',
						},
					],
					count: 1,
				},
			],
		}

		const result = v.safeParse(BoardStateSchema, rawBoard)
		expect(result.success).toBe(true)
		if (result.success) {
			expect(result.output.columns[0].tasks[0].priority).toBe('low')
		}
	})

	it('validates custom strategy tiers in snapshot', () => {
		const snapshotWithTiers: SnapshotInput = {
			config: {
				project: { name: 'Tier Project', description: '' },
				board: { columns: [{ id: 'col1', name: 'Col 1', color: '#000' }] },
				tiers: [
					{
						id: '1',
						name: 'Tier 1',
						title: 'Core Invariants',
						summary: 'Zero CGO invariants',
						color: '#3b82f6',
					},
					{
						id: 'domain',
						name: 'Domain',
						title: 'Domain Patterns',
						summary: 'Architectural domain patterns',
						color: '#a855f7',
					},
				],
			},
			tasks: [],
			milestones: [],
			strategies: [
				{
					id: 'core-rule',
					title: 'Core Rule',
					tier: 'domain',
					summary: 'Rule summary',
				},
			],
			glossary: [],
		}

		const result = v.safeParse(SnapshotSchema, snapshotWithTiers)
		expect(result.success).toBe(true)
		if (result.success) {
			expect(result.output.config.tiers?.length).toBe(2)
			expect(result.output.strategies[0].tier).toBe('domain')
		}
	})
})
