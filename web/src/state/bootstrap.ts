import * as v from 'valibot'
import { type Snapshot, SnapshotSchema } from '../schemas/models'
import { loadStateFromStorage } from './storage'

export const PAYLOAD_PLACEHOLDER = [
	'/*',
	'JOKATEKO_PAYLOAD_PLACEHOLDER',
	'*/',
].join(' ')
export const SCRIPT_DATA_ID = 'jokateko-data'

export type AppMode = 'client' | 'live' | 'static'

export interface BootstrapResult {
	mode: AppMode
	snapshot: Snapshot | null
	warnings: string[]
}

/**
 * Extracts and parses snapshot data from a raw JSON string.
 * Returns null if the content is missing, empty, or equals the placeholder comment.
 */
export function parseSnapshotContent(content: string | null | undefined): {
	snapshot: Snapshot | null
	warnings: string[]
} | null {
	if (!content) {
		return null
	}

	const trimmed = content.trim()
	if (
		trimmed === ''
		|| trimmed === PAYLOAD_PLACEHOLDER
		|| trimmed.includes('JOKATEKO_PAYLOAD_PLACEHOLDER')
	) {
		return null
	}

	let rawJson: unknown
	try {
		rawJson = JSON.parse(trimmed)
	} catch (err) {
		return {
			snapshot: null,
			warnings: [
				`Failed to parse embedded JSON payload: ${
					err instanceof Error ? err.message : String(err)
				}`,
			],
		}
	}

	const result = v.safeParse(SnapshotSchema, rawJson)
	if (!result.success) {
		const warnings = result.issues.map((issue) => {
			const path = issue.path?.map((p) => String(p.key)).join('.') || 'root'
			return `${path}: ${issue.message}`
		})

		// Graceful recovery: return partial output if available
		const partialSnapshot = (result.output as Snapshot) || null
		return {
			snapshot: partialSnapshot,
			warnings,
		}
	}

	return {
		snapshot: result.output,
		warnings: [],
	}
}

/**
 * Detects whether the application boots in Client Mode, Static Mode, or Live Mode.
 * Checks localStorage first, then embedded script tag, defaulting to Client mode.
 */
export function bootstrap(
	doc: Document = document,
	options: { checkStorage?: boolean; defaultMode?: AppMode } = {},
): BootstrapResult {
	const checkStorage = options.checkStorage ?? true
	const isHttp =
		typeof window !== 'undefined'
		&& Boolean(
			window.location
				&& (window.location.protocol === 'http:'
					|| window.location.protocol === 'https:'),
		)

	// 1. Check embedded static snapshot script tag first
	const scriptEl = doc.getElementById(SCRIPT_DATA_ID)
	if (scriptEl) {
		const parsed = parseSnapshotContent(scriptEl.textContent)
		if (parsed?.snapshot) {
			return {
				mode: 'static',
				snapshot: parsed.snapshot,
				warnings: parsed.warnings,
			}
		}
		if (parsed && parsed.warnings.length > 0) {
			return {
				mode: 'static',
				snapshot: null,
				warnings: parsed.warnings,
			}
		}
	}

	// 2. If served over HTTP/HTTPS, boot into Live mode unless explicitly configured otherwise
	if (isHttp && options.defaultMode !== 'client') {
		return {
			mode: options.defaultMode ?? 'live',
			snapshot: null,
			warnings: [],
		}
	}

	// 3. In non-HTTP or client mode, check saved state in localStorage
	if (checkStorage) {
		const storageRes = loadStateFromStorage()
		if (storageRes?.snapshot) {
			return {
				mode: 'client',
				snapshot: storageRes.snapshot,
				warnings: storageRes.warnings,
			}
		}
	}

	return {
		mode: options.defaultMode ?? (isHttp ? 'live' : 'client'),
		snapshot: null,
		warnings: [],
	}
}
