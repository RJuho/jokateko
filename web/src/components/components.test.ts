import { describe, expect, it } from 'bun:test'
import * as v from 'valibot'
import { sampleSnapshot } from '../fixtures/sampleData'
import { SnapshotSchema } from '../schemas/models'
import {
	columnTasks,
	config,
	filteredTasks,
	initFromSnapshot,
	milestones,
	mode,
	setMilestoneFilter,
	setSearchQuery,
	strategies,
	tasks,
	togglePriorityFilter,
	toggleTagFilter,
} from '../state/store'

describe('Sample Data & Component Logic Verification', () => {
	it('validates that sampleSnapshot conforms strictly to Valibot SnapshotSchema', () => {
		const result = v.safeParse(SnapshotSchema, sampleSnapshot)
		expect(result.success).toBe(true)
		if (result.success) {
			expect(result.output.tasks.length).toBe(6)
			expect(result.output.milestones.length).toBe(2)
			expect(result.output.strategies.length).toBe(3)
			expect(result.output.glossary.length).toBe(3)
		}
	})

	it('initializes store with sampleSnapshot and partitions tasks into Kanban columns', () => {
		initFromSnapshot(sampleSnapshot)

		expect(mode.value).toBe('static')
		expect(config.value.project.name).toBe('Jokateko Kanban')
		expect(tasks.value.length).toBe(6)
		expect(milestones.value.length).toBe(2)
		expect(strategies.value.length).toBe(3)

		const cols = columnTasks.value
		expect(cols.in_progress.length).toBe(2)
		expect(cols.ready.length).toBe(2)
		expect(cols.backlog.length).toBe(1)
		expect(cols.done.length).toBe(1)
	})

	it('filters sample data accurately by query, tag, priority, and milestone', () => {
		initFromSnapshot(sampleSnapshot)

		// 1. Text search
		setSearchQuery('oauth2')
		expect(filteredTasks.value.length).toBe(1)
		expect(filteredTasks.value[0].id).toBe('260901-user-auth')

		// Clear search
		setSearchQuery('')
		expect(filteredTasks.value.length).toBe(6)

		// 2. Tag filter
		toggleTagFilter('security')
		expect(filteredTasks.value.length).toBe(3) // user-auth, blocked-deploy, spec-audit

		// 3. Priority filter
		togglePriorityFilter('critical')
		expect(filteredTasks.value.length).toBe(2) // user-auth, spec-audit

		// 4. Milestone filter
		setMilestoneFilter('m1-mvp-release')
		expect(filteredTasks.value.length).toBe(2)
	})
})
