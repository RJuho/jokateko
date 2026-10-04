import type { Column, Priority } from '../schemas/models'
import { DONE_STATUS, isDoneStatus } from './status'

export type SortMode =
	| 'default'
	| 'priority'
	| 'target_at'
	| 'changed_at'
	| 'created_at'
	| 'title'

export type SortableTask = {
	id: string
	title?: string
	status: string
	priority?: Priority
	target_at?: string
	changed_at?: string
	created_at?: string
}

export function getPriorityRank(priority?: Priority): number {
	switch (priority) {
		case 'critical':
			return 0
		case 'high':
			return 1
		case 'medium':
			return 2
		case 'low':
			return 3
		default:
			return 2
	}
}

/**
 * Compares two tasks for a specific column, taking into account either
 * the column's configured sort rules / default workflow state rules (mode === 'default')
 * or an active global sort mode override.
 */
export function compareTasks(
	a: SortableTask,
	b: SortableTask,
	col: Column,
	mode: SortMode = 'default',
): number {
	if (mode === 'default') {
		const customSortBy = col.sort_by?.trim().toLowerCase()
		const customSortDir = col.sort_direction?.trim().toLowerCase()

		// For 'done' state, default sorting is Recently changed (changed_at desc),
		// since priority does not matter anymore when task is done.
		if (col.id === DONE_STATUS || isDoneStatus(a.status)) {
			if (customSortBy && customSortBy !== 'default') {
				const res = compareField(a, b, customSortBy, customSortDir)
				if (res !== 0) {
					return res
				}
			} else {
				const diff = (b.changed_at || '').localeCompare(a.changed_at || '')
				if (diff !== 0) {
					return diff
				}
			}
			return a.id.localeCompare(b.id)
		}

		// Primary Tier for active and backlog states: Priority (critical > high > medium > low)
		const pA = getPriorityRank(a.priority)
		const pB = getPriorityRank(b.priority)
		if (pA !== pB) {
			return pA - pB
		}

		// Secondary Tier: Custom column configuration or default workflow rules
		if (customSortBy && customSortBy !== 'default') {
			const res = compareField(a, b, customSortBy, customSortDir)
			if (res !== 0) {
				return res
			}
		} else {
			const res = compareDefaultWorkflowSecondary(a, b, col.id)
			if (res !== 0) {
				return res
			}
		}

		// Tertiary Tier: ID ascending
		return a.id.localeCompare(b.id)
	}

	// Global sort mode overrides
	switch (mode) {
		case 'priority': {
			const pA = getPriorityRank(a.priority)
			const pB = getPriorityRank(b.priority)
			if (pA !== pB) {
				return pA - pB
			}
			const diff = (b.changed_at || '').localeCompare(a.changed_at || '')
			if (diff !== 0) {
				return diff
			}
			return a.id.localeCompare(b.id)
		}

		case 'target_at': {
			const res = compareTargetDates(a, b, 'asc')
			if (res !== 0) {
				return res
			}
			return a.id.localeCompare(b.id)
		}

		case 'changed_at': {
			const diff = (b.changed_at || '').localeCompare(a.changed_at || '')
			if (diff !== 0) {
				return diff
			}
			return a.id.localeCompare(b.id)
		}

		case 'created_at': {
			const diff = (b.created_at || '').localeCompare(a.created_at || '')
			if (diff !== 0) {
				return diff
			}
			return a.id.localeCompare(b.id)
		}

		case 'title': {
			const diff = (a.title || '').localeCompare(b.title || '', undefined, {
				sensitivity: 'base',
			})
			if (diff !== 0) {
				return diff
			}
			return a.id.localeCompare(b.id)
		}

		default:
			return a.id.localeCompare(b.id)
	}
}

function compareDefaultWorkflowSecondary(
	a: SortableTask,
	b: SortableTask,
	colId: string,
): number {
	switch (colId) {
		case 'ready':
		case 'in_progress':
		case 'in_review': {
			// Soonest target_at ascending. Empty target_at placed after, fallback to changed_at desc.
			return compareTargetDates(a, b, 'asc')
		}

		default: {
			// backlog, done, or other columns: changed_at descending (newest to oldest)
			return (b.changed_at || '').localeCompare(a.changed_at || '')
		}
	}
}

function compareTargetDates(
	a: SortableTask,
	b: SortableTask,
	direction: 'asc' | 'desc',
): number {
	const hasTargetA = Boolean(a.target_at && a.target_at.trim() !== '')
	const hasTargetB = Boolean(b.target_at && b.target_at.trim() !== '')

	if (hasTargetA && hasTargetB) {
		const targetA = a.target_at || ''
		const targetB = b.target_at || ''
		let cmp = targetA.localeCompare(targetB)
		if (direction === 'desc') {
			cmp = -cmp
		}
		if (cmp !== 0) {
			return cmp
		}
		// Same target date -> fallback to changed_at descending
		return (b.changed_at || '').localeCompare(a.changed_at || '')
	}

	if (hasTargetA && !hasTargetB) {
		return -1 // A has target, comes first
	}
	if (!hasTargetA && hasTargetB) {
		return 1 // B has target, comes first
	}

	// Both empty target_at -> fallback to changed_at descending
	return (b.changed_at || '').localeCompare(a.changed_at || '')
}

function compareField(
	a: SortableTask,
	b: SortableTask,
	field: string,
	direction?: string,
): number {
	const isDesc = direction === 'desc'

	switch (field) {
		case 'target_at':
			return compareTargetDates(a, b, isDesc ? 'desc' : 'asc')

		case 'changed_at': {
			const diff = (a.changed_at || '').localeCompare(b.changed_at || '')
			return isDesc ? -diff : diff
		}

		case 'created_at': {
			const diff = (a.created_at || '').localeCompare(b.created_at || '')
			return isDesc ? -diff : diff
		}

		case 'title': {
			const diff = (a.title || '').localeCompare(b.title || '', undefined, {
				sensitivity: 'base',
			})
			return isDesc ? -diff : diff
		}

		case 'id': {
			const diff = a.id.localeCompare(b.id)
			return isDesc ? -diff : diff
		}

		case 'priority': {
			return (b.changed_at || '').localeCompare(a.changed_at || '')
		}

		default:
			return 0
	}
}
