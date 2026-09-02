import { computed, signal } from '@preact/signals'
import * as v from 'valibot'
import {
	BoardStateSchema,
	GlossaryTermSchema,
	MilestoneSchema,
	type Priority,
	type PriorityConfig,
	type Snapshot,
	type SnapshotConfig,
	StrategySchema,
	type Task,
} from '../schemas/models'
import type { AppMode } from './bootstrap'
import { saveStateToStorage } from './storage'

export type Tab = 'board' | 'milestones' | 'strategies' | 'glossary'

export interface FilterState {
	searchQuery: string
	selectedTags: string[]
	selectedMilestone: string | null
	selectedPriorities: Priority[]
}

export const defaultPriorities: PriorityConfig[] = [
	{ id: 'critical', name: 'Critical', color: '#ef4444' },
	{ id: 'high', name: 'High', color: '#f97316' },
	{ id: 'medium', name: 'Medium', color: '#eab308' },
	{ id: 'low', name: 'Low', color: '#3b82f6' },
]

const initialConfig: SnapshotConfig = {
	project: {
		name: 'Jokateko',
		description: '',
	},
	board: {
		columns: [
			{ id: 'backlog', name: 'Backlog', color: '#94a3b8' },
			{ id: 'ready', name: 'Ready', color: '#38bdf8' },
			{ id: 'in_progress', name: 'In Progress', color: '#fbbf24' },
			{ id: 'in_review', name: 'In Review', color: '#c084fc' },
			{ id: 'done', name: 'Done', color: '#4ade80' },
		],
	},
	priorities: defaultPriorities,
	tags: {
		allowed: [],
		enforce_allowed: false,
	},
	build: {
		time: '',
		branch: '',
		commit: '',
		version: '0.1.0',
	},
}

// Signals
export const mode = signal<AppMode>('client')
export const activeTab = signal<Tab>('board')
export const config = signal<SnapshotConfig>(initialConfig)
export const configuredPriorities = computed<PriorityConfig[]>(() => {
	const p = config.value.priorities
	if (p && p.length > 0) {
		return p
	}
	return defaultPriorities
})
export const tasks = signal<Task[]>([])
export const milestones = signal<Snapshot['milestones']>([])
export const strategies = signal<Snapshot['strategies']>([])
export const glossary = signal<Snapshot['glossary']>([])
export const validationWarnings = signal<string[]>([])
export const connectionStatus = signal<
	'connected' | 'connecting' | 'disconnected'
>('connecting')

export const filters = signal<FilterState>({
	searchQuery: '',
	selectedTags: [],
	selectedMilestone: null,
	selectedPriorities: [],
})

export const activeTaskDetailId = signal<string | null>(null)
export const activeTaskEditId = signal<string | null>(null)
export const isCreateTaskModalOpen = signal<boolean>(false)
export const createTaskInitialColumnId = signal<string | null>(null)
export const activeStrategyId = signal<string | null>(null)
export const activeGlossaryId = signal<string | null>(null)

// Computed
export const filteredTasks = computed(() => {
	const currentTasks = tasks.value
	const { searchQuery, selectedTags, selectedMilestone, selectedPriorities } =
		filters.value

	const normalizedQuery = searchQuery.trim().toLowerCase()

	return currentTasks.filter((task) => {
		// 1. Text search across title, id, summary, body, and tags
		if (normalizedQuery !== '') {
			const inTitle = task.title.toLowerCase().includes(normalizedQuery)
			const inId = task.id.toLowerCase().includes(normalizedQuery)
			const inSummary = (task.summary || '')
				.toLowerCase()
				.includes(normalizedQuery)
			const inBody = (task.body || '').toLowerCase().includes(normalizedQuery)
			const inTags =
				task.tags?.some((t) => t.toLowerCase().includes(normalizedQuery))
				?? false
			if (!inTitle && !inId && !inSummary && !inBody && !inTags) {
				return false
			}
		}

		// 2. Tag filter
		if (selectedTags.length > 0) {
			const taskTags = task.tags || []
			const hasAllTags = selectedTags.every((t) => taskTags.includes(t))
			if (!hasAllTags) {
				return false
			}
		}

		// 3. Milestone filter
		if (selectedMilestone !== null) {
			if (task.milestone !== selectedMilestone) {
				return false
			}
		}

		// 4. Priority filter
		if (selectedPriorities.length > 0) {
			if (!selectedPriorities.includes(task.priority)) {
				return false
			}
		}

		return true
	})
})

export const columnTasks = computed(() => {
	const cols = config.value.board.columns
	const taskList = filteredTasks.value
	const map: Record<string, Task[]> = {}

	for (const col of cols) {
		map[col.id] = []
	}

	for (const task of taskList) {
		if (map[task.status]) {
			map[task.status].push(task)
		} else {
			// If status doesn't match a defined column, default to first column or backlog
			const fallbackId = cols[0]?.id || 'backlog'
			if (!map[fallbackId]) {
				map[fallbackId] = []
			}
			map[fallbackId].push(task)
		}
	}

	return map
})

export const allTags = computed(() => {
	const tagSet = new Set<string>()

	for (const t of tasks.value) {
		for (const tag of t.tags || []) {
			tagSet.add(tag)
		}
	}
	for (const m of milestones.value) {
		for (const tag of m.tags || []) {
			tagSet.add(tag)
		}
	}
	for (const s of strategies.value) {
		for (const tag of s.tags || []) {
			tagSet.add(tag)
		}
	}
	for (const g of glossary.value) {
		for (const tag of g.tags || []) {
			tagSet.add(tag)
		}
	}

	return Array.from(tagSet).sort()
})

// Actions
export function getCurrentSnapshot(): Snapshot {
	return {
		config: config.value,
		tasks: tasks.value,
		milestones: milestones.value,
		strategies: strategies.value,
		glossary: glossary.value,
	}
}

export function persistCurrentState(): void {
	try {
		saveStateToStorage(getCurrentSnapshot())
	} catch (err) {
		console.warn('Failed to persist current state:', err)
	}
}

export function initFromSnapshot(
	snapshot: Snapshot,
	warnings: string[] = [],
	targetMode: AppMode = 'static',
): void {
	mode.value = targetMode
	config.value = snapshot.config
	tasks.value = snapshot.tasks
	milestones.value = snapshot.milestones
	strategies.value = snapshot.strategies
	glossary.value = snapshot.glossary
	validationWarnings.value = warnings
	connectionStatus.value = 'connected'
}

export function upsertTask(task: Task): void {
	const idx = tasks.value.findIndex((t) => t.id === task.id)
	if (idx >= 0) {
		const updated = [...tasks.value]
		updated[idx] = task
		tasks.value = updated
	} else {
		tasks.value = [...tasks.value, task]
	}
	persistCurrentState()
}

export function removeTask(taskId: string): void {
	tasks.value = tasks.value.filter((t) => t.id !== taskId)
	if (activeTaskDetailId.value === taskId) {
		activeTaskDetailId.value = null
	}
	if (activeTaskEditId.value === taskId) {
		activeTaskEditId.value = null
	}
	persistCurrentState()
}

export function upsertMilestone(
	milestone: Snapshot['milestones'][number],
): void {
	const idx = milestones.value.findIndex((m) => m.id === milestone.id)
	if (idx >= 0) {
		const updated = [...milestones.value]
		updated[idx] = milestone
		milestones.value = updated
	} else {
		milestones.value = [...milestones.value, milestone]
	}
	persistCurrentState()
}

export function removeMilestone(id: string): void {
	milestones.value = milestones.value.filter((m) => m.id !== id)
	persistCurrentState()
}

export function setSearchQuery(q: string): void {
	filters.value = { ...filters.value, searchQuery: q }
}

export function toggleTagFilter(tag: string): void {
	const current = filters.value.selectedTags
	const next = current.includes(tag)
		? current.filter((t) => t !== tag)
		: [...current, tag]
	filters.value = { ...filters.value, selectedTags: next }
}

export function togglePriorityFilter(p: Priority): void {
	const current = filters.value.selectedPriorities
	const next = current.includes(p)
		? current.filter((item) => item !== p)
		: [...current, p]
	filters.value = { ...filters.value, selectedPriorities: next }
}

export function setMilestoneFilter(m: string | null): void {
	filters.value = { ...filters.value, selectedMilestone: m }
}

export function resetFilters(): void {
	filters.value = {
		searchQuery: '',
		selectedTags: [],
		selectedMilestone: null,
		selectedPriorities: [],
	}
}

// REST Fetch for Live Mode
export async function fetchLiveBoard(): Promise<boolean> {
	try {
		const res = await fetch('/api/board')
		if (!res.ok) {
			throw new Error(`HTTP ${res.status}: ${res.statusText}`)
		}
		const data = await res.json()
		const parsed = v.safeParse(BoardStateSchema, data)
		if (parsed.success) {
			config.value = {
				...config.value,
				project: {
					...config.value.project,
					name: parsed.output.project_name,
				},
				board: {
					columns: parsed.output.columns.map((c) => ({
						id: c.id,
						name: c.name,
						color: c.color,
					})),
				},
			}

			// Gather all tasks from columns
			const allBoardTasks: Task[] = []
			for (const col of parsed.output.columns) {
				allBoardTasks.push(...col.tasks)
			}
			tasks.value = allBoardTasks
			return true
		}
		return false
	} catch (err) {
		console.warn('Failed to fetch live board state:', err)
		return false
	}
}

export async function fetchLiveEntities(): Promise<void> {
	try {
		const [msRes, stratRes, glossRes] = await Promise.all([
			fetch('/api/milestones'),
			fetch('/api/strategies'),
			fetch('/api/glossary'),
		])

		if (msRes.ok) {
			const msData = await msRes.json()
			const parsed = v.safeParse(v.array(MilestoneSchema), msData)
			if (parsed.success) {
				milestones.value = parsed.output
			}
		}

		if (stratRes.ok) {
			const stratData = await stratRes.json()
			const parsed = v.safeParse(v.array(StrategySchema), stratData)
			if (parsed.success) {
				strategies.value = parsed.output
			}
		}

		if (glossRes.ok) {
			const glossData = await glossRes.json()
			const parsed = v.safeParse(v.array(GlossaryTermSchema), glossData)
			if (parsed.success) {
				glossary.value = parsed.output
			}
		}
	} catch (err) {
		console.warn('Failed to fetch live entities:', err)
	}
}
