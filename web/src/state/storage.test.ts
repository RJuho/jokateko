import { beforeEach, describe, expect, it } from 'bun:test'
import { sampleSnapshot } from '../fixtures/sampleData'
import {
	clearStateFromStorage,
	loadStateFromStorage,
	STORAGE_KEY,
	saveStateToStorage,
} from './storage'

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

describe('LocalStorage State Persistence with Valibot', () => {
	beforeEach(() => {
		localStorage.clear()
	})

	it('returns null when localStorage is empty', () => {
		expect(loadStateFromStorage()).toBeNull()
	})

	it('saves and reloads valid snapshot state', () => {
		const saved = saveStateToStorage(sampleSnapshot)
		expect(saved).toBe(true)

		const loaded = loadStateFromStorage()
		expect(loaded).not.toBeNull()
		expect(loaded?.snapshot?.config.project.name).toBe(
			sampleSnapshot.config.project.name,
		)
		expect(loaded?.snapshot?.tasks.length).toBe(sampleSnapshot.tasks.length)
		expect(loaded?.warnings).toEqual([])
	})

	it('handles corrupt json gracefully with warnings', () => {
		localStorage.setItem(STORAGE_KEY, 'not-valid-json{{')
		const loaded = loadStateFromStorage()
		expect(loaded).not.toBeNull()
		expect(loaded?.snapshot).toBeNull()
		expect(loaded?.warnings.length).toBeGreaterThan(0)
	})

	it('handles schema validation errors with Valibot diagnostics', () => {
		localStorage.setItem(
			STORAGE_KEY,
			JSON.stringify({
				config: { project: { name: 123 } }, // invalid: name should be string
				tasks: 'not-an-array', // invalid
			}),
		)
		const loaded = loadStateFromStorage()
		expect(loaded).not.toBeNull()
		expect(loaded?.warnings.length).toBeGreaterThan(0)
	})

	it('clears state correctly', () => {
		saveStateToStorage(sampleSnapshot)
		expect(localStorage.getItem(STORAGE_KEY)).not.toBeNull()

		clearStateFromStorage()
		expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
		expect(loadStateFromStorage()).toBeNull()
	})
})
