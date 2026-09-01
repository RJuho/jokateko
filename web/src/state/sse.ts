import * as v from 'valibot'
import { MilestoneSchema, TaskSchema } from '../schemas/models'
import {
	connectionStatus,
	fetchLiveBoard,
	removeMilestone,
	removeTask,
	upsertMilestone,
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
			}
		} catch (err) {
			console.warn('Failed to handle task.deleted SSE:', err)
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
			}
		} catch (err) {
			console.warn('Failed to handle milestone.deleted SSE:', err)
		}
	})

	// Board refresh event
	eventSource.addEventListener('board.refreshed', () => {
		fetchLiveBoard()
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
			console.warn('Invalid task SSE payload:', res.issues)
		}
	} catch (err) {
		console.warn('Failed to parse task SSE data:', err)
	}
}

function handleMilestonePayload(rawData: string): void {
	try {
		const parsed = JSON.parse(rawData)
		const res = v.safeParse(MilestoneSchema, parsed)
		if (res.success) {
			upsertMilestone(res.output)
		} else {
			console.warn('Invalid milestone SSE payload:', res.issues)
		}
	} catch (err) {
		console.warn('Failed to parse milestone SSE data:', err)
	}
}
