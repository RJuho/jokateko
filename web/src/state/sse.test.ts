import { afterEach, beforeEach, describe, expect, it } from 'bun:test'
import { startSSE, stopSSE } from './sse'

type Listener = (e: { data: string }) => void

class FakeEventSource {
	static last: FakeEventSource | null = null
	onopen: (() => void) | null = null
	onerror: (() => void) | null = null
	listeners = new Map<string, Listener>()

	constructor(public url: string) {
		FakeEventSource.last = this
	}

	addEventListener(type: string, fn: Listener) {
		this.listeners.set(type, fn)
	}

	emit(type: string, data: unknown) {
		this.listeners.get(type)?.({ data: JSON.stringify(data) })
	}

	close() {}
}

const realEventSource = globalThis.EventSource
const realFetch = globalThis.fetch
let fetched: string[] = []

beforeEach(() => {
	fetched = []
	globalThis.EventSource = FakeEventSource as unknown as typeof EventSource
	globalThis.fetch = (async (input: string) => {
		fetched.push(input)
		return new Response('{}')
	}) as typeof fetch
})

afterEach(() => {
	stopSSE()
	globalThis.EventSource = realEventSource
	globalThis.fetch = realFetch
})

describe('SSE live sync', () => {
	it('re-syncs board and entities on the first open', () => {
		startSSE()
		expect(fetched).toEqual([])

		FakeEventSource.last?.onopen?.()

		expect(fetched).toContain('/api/board')
		expect(fetched).toContain('/api/milestones')
	})

	it('coalesces task events into a single milestone refetch', async () => {
		startSSE()
		const es = FakeEventSource.last as FakeEventSource

		es.emit('task.created', { id: 'a' })
		es.emit('task.updated', { id: 'b' })
		es.emit('task.deleted', { id: 'c' })
		await new Promise((resolve) => setTimeout(resolve, 150))

		expect(fetched.filter((u) => u === '/api/milestones')).toHaveLength(1)
	})
})
