import * as v from 'valibot'
import { type Snapshot, SnapshotSchema } from '../schemas/models'

export const STORAGE_KEY = 'jokateko_state'

export interface StorageLoadResult {
	snapshot: Snapshot | null
	warnings: string[]
}

function getStorage(): Storage | null {
	if (
		typeof globalThis !== 'undefined'
		&& typeof globalThis.localStorage !== 'undefined'
	) {
		return globalThis.localStorage
	}
	return null
}

/**
 * Loads and validates saved state from localStorage using Valibot parser.
 */
export function loadStateFromStorage(): StorageLoadResult | null {
	const storage = getStorage()
	if (!storage) {
		return null
	}

	let raw: string | null = null
	try {
		raw = storage.getItem(STORAGE_KEY)
	} catch (err) {
		console.warn('Failed to read from localStorage:', err)
		return null
	}

	if (!raw) {
		return null
	}

	let json: unknown
	try {
		json = JSON.parse(raw)
	} catch (err) {
		return {
			snapshot: null,
			warnings: [
				`Failed to parse localStorage data: ${
					err instanceof Error ? err.message : String(err)
				}`,
			],
		}
	}

	const result = v.safeParse(SnapshotSchema, json)
	if (!result.success) {
		const warnings = result.issues.map((issue) => {
			const path = issue.path?.map((p) => String(p.key)).join('.') || 'root'
			return `${path}: ${issue.message}`
		})
		return {
			snapshot: (result.output as Snapshot) || null,
			warnings,
		}
	}

	return {
		snapshot: result.output,
		warnings: [],
	}
}

/**
 * Validates with Valibot and persists snapshot state to localStorage.
 */
export function saveStateToStorage(snapshot: Snapshot): boolean {
	const storage = getStorage()
	if (!storage) {
		return false
	}

	const result = v.safeParse(SnapshotSchema, snapshot)
	if (!result.success) {
		console.warn('Cannot persist invalid state to localStorage:', result.issues)
		return false
	}

	try {
		storage.setItem(STORAGE_KEY, JSON.stringify(result.output))
		return true
	} catch (err) {
		console.error('Failed to write state to localStorage:', err)
		return false
	}
}

/**
 * Clears saved state from localStorage.
 */
export function clearStateFromStorage(): void {
	const storage = getStorage()
	if (!storage) {
		return
	}
	try {
		storage.removeItem(STORAGE_KEY)
	} catch (err) {
		console.warn('Failed to clear state from localStorage:', err)
	}
}
