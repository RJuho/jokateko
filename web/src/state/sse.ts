import * as v from 'valibot'
import {
	GlossaryTermSchema,
	MilestoneSchema,
	StrategySchema,
	TaskSchema,
} from '../schemas/models'
import {
	connectionStatus,
	fetchLiveBoard,
	fetchLiveEntities,
	removeGlossaryTerm,
	removeMilestone,
	removeStrategy,
	removeTask,
	upsertGlossaryTerm,
	upsertMilestone,
	upsertStrategy,
	upsertTask,
} from './store'

const EntityIdSchema = v.object({
	id: v.string(),
})

let eventSource: EventSource | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let reconnectDelay = 1000
let isExplicitlyClosed = false

export function startSSE(endpoint = '/api/events'): void {
	if (eventSource) {
		return
	}

	isExplicitlyClosed = false
	connectionStatus.value = 'connecting'

	try {
		eventSource = new EventSource(endpoint)
	} catch (err) {
		console.warn('Failed to construct EventSource:', err)
		scheduleReconnect(endpoint)
		return
	}

	eventSource.onopen = () => {
		connectionStatus.value = 'connected'
		reconnectDelay = 1000
	}

	eventSource.onerror = () => {
		if (isExplicitlyClosed) {
			return
		}
		connectionStatus.value = 'disconnected'
		stopSSE(false)
		scheduleReconnect(endpoint)
	}

	// Task Events
	eventSource.addEventListener('task.created', (e) => {
		handleTaskPayload(e.data)
	})

	eventSource.addEventListener('task.updated', (e) => {
		handleTaskPayload(e.data)
	})

	eventSource.addEventListener('task.deleted', (e) => {
		try {
			const parsed = JSON.parse(e.data)
			const res = v.safeParse(EntityIdSchema, parsed)
			if (res.success) {
				removeTask(res.output.id)
			} else {
				fetchLiveBoard()
			}
		} catch {
			fetchLiveBoard()
		}
	})

	// Milestone Events
	eventSource.addEventListener('milestone.created', (e) => {
		handleMilestonePayload(e.data)
	})

	eventSource.addEventListener('milestone.updated', (e) => {
		handleMilestonePayload(e.data)
	})

	eventSource.addEventListener('milestone.deleted', (e) => {
		try {
			const parsed = JSON.parse(e.data)
			const res = v.safeParse(EntityIdSchema, parsed)
			if (res.success) {
				removeMilestone(res.output.id)
			} else {
				fetchLiveEntities()
			}
		} catch {
			fetchLiveEntities()
		}
	})

	// Strategy Events
	eventSource.addEventListener('strategy.created', (e) => {
		handleStrategyPayload(e.data)
	})

	eventSource.addEventListener('strategy.updated', (e) => {
		handleStrategyPayload(e.data)
	})

	eventSource.addEventListener('strategy.deleted', (e) => {
		try {
			const parsed = JSON.parse(e.data)
			const res = v.safeParse(EntityIdSchema, parsed)
			if (res.success) {
				removeStrategy(res.output.id)
			} else {
				fetchLiveEntities()
			}
		} catch {
			fetchLiveEntities()
		}
	})

	// Glossary Events
	eventSource.addEventListener('glossary.created', (e) => {
		handleGlossaryPayload(e.data)
	})

	eventSource.addEventListener('glossary.updated', (e) => {
		handleGlossaryPayload(e.data)
	})

	eventSource.addEventListener('glossary.deleted', (e) => {
		try {
			const parsed = JSON.parse(e.data)
			const res = v.safeParse(EntityIdSchema, parsed)
			if (res.success) {
				removeGlossaryTerm(res.output.id)
			} else {
				fetchLiveEntities()
			}
		} catch {
			fetchLiveEntities()
		}
	})

	// Board refresh event
	eventSource.addEventListener('board.refreshed', () => {
		fetchLiveBoard()
		fetchLiveEntities()
	})

	// Heartbeat
	eventSource.addEventListener('ping', () => {
		connectionStatus.value = 'connected'
	})
}

export function stopSSE(explicit = true): void {
	if (explicit) {
		isExplicitlyClosed = true
		if (reconnectTimer) {
			clearTimeout(reconnectTimer)
			reconnectTimer = null
		}
		connectionStatus.value = 'disconnected'
	}

	if (eventSource) {
		eventSource.close()
		eventSource = null
	}
}

function scheduleReconnect(endpoint: string): void {
	if (isExplicitlyClosed || reconnectTimer) {
		return
	}

	reconnectTimer = setTimeout(() => {
		reconnectTimer = null
		reconnectDelay = Math.min(reconnectDelay * 1.5, 10000)
		startSSE(endpoint)
	}, reconnectDelay)
}

function handleTaskPayload(rawData: string): void {
	try {
		const parsed = JSON.parse(rawData)
		const res = v.safeParse(TaskSchema, parsed)
		if (res.success) {
			upsertTask(res.output)
		} else {
			fetchLiveBoard()
		}
	} catch (err) {
		console.warn('Failed to parse task SSE data:', err)
		fetchLiveBoard()
	}
}

function handleMilestonePayload(rawData: string): void {
	try {
		const parsed = JSON.parse(rawData)
		const res = v.safeParse(MilestoneSchema, parsed)
		if (res.success) {
			upsertMilestone(res.output)
		} else {
			fetchLiveEntities()
		}
	} catch (err) {
		console.warn('Failed to parse milestone SSE data:', err)
		fetchLiveEntities()
	}
}

function handleStrategyPayload(rawData: string): void {
	try {
		const parsed = JSON.parse(rawData)
		const res = v.safeParse(StrategySchema, parsed)
		if (res.success) {
			upsertStrategy(res.output)
		} else {
			fetchLiveEntities()
		}
	} catch (err) {
		console.warn('Failed to parse strategy SSE data:', err)
		fetchLiveEntities()
	}
}

function handleGlossaryPayload(rawData: string): void {
	try {
		const parsed = JSON.parse(rawData)
		const res = v.safeParse(GlossaryTermSchema, parsed)
		if (res.success) {
			upsertGlossaryTerm(res.output)
		} else {
			fetchLiveEntities()
		}
	} catch (err) {
		console.warn('Failed to parse glossary SSE data:', err)
		fetchLiveEntities()
	}
}
