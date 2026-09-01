import * as v from 'valibot'
import { type Snapshot, SnapshotSchema } from '../schemas/models'

export const PAYLOAD_PLACEHOLDER = '/* JOKATEKO_PAYLOAD_PLACEHOLDER */'
export const SCRIPT_DATA_ID = 'jokateko-data'

export type AppMode = 'live' | 'static'

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
 * Detects whether the application boots in Live Mode or Static Mode
 * by inspecting the `<script id="jokateko-data">` DOM element.
 */
export function bootstrap(doc: Document = document): BootstrapResult {
	const scriptEl = doc.getElementById(SCRIPT_DATA_ID)
	if (!scriptEl) {
		return {
			mode: 'live',
			snapshot: null,
			warnings: [],
		}
	}

	const parsed = parseSnapshotContent(scriptEl.textContent)
	if (!parsed) {
		return {
			mode: 'live',
			snapshot: null,
			warnings: [],
		}
	}

	return {
		mode: 'static',
		snapshot: parsed.snapshot,
		warnings: parsed.warnings,
	}
}
