import { beforeEach, describe, expect, it } from 'bun:test'
import { sampleSnapshot } from '../fixtures/sampleData'
import { STORAGE_KEY } from './storage'
import { initFromSnapshot, persistCurrentState } from './store'

if (typeof globalThis.localStorage === 'undefined') {
	const store = new Map<string, string>()
	globalThis.localStorage = {
		getItem: (key: string) => store.get(key) ?? null,
		setItem: (key: string, value: string) => store.set(key, String(value)),
		removeItem: (key: string) => store.delete(key),
		clear: () => store.clear(),
		key: (index: number) => Array.from(store.keys())[index] ?? null,
		get length() {
			return store.size
		},
	} as Storage
}

describe('persistCurrentState mode gating', () => {
	beforeEach(() => {
		localStorage.clear()
	})

	it('writes localStorage in client mode', () => {
		initFromSnapshot(sampleSnapshot, [], 'client')
		persistCurrentState()
		expect(localStorage.getItem(STORAGE_KEY)).not.toBeNull()
	})

	it('skips localStorage in live and static modes', () => {
		for (const m of ['live', 'static'] as const) {
			initFromSnapshot(sampleSnapshot, [], m)
			persistCurrentState()
			expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
		}
	})
})
