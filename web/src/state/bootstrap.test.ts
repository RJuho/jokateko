import { describe, expect, it } from 'bun:test'
import {
	bootstrap,
	PAYLOAD_PLACEHOLDER,
	parseSnapshotContent,
	SCRIPT_DATA_ID,
} from './bootstrap'

function createMockDocument(scriptContent: string | null): Document {
	const elements = new Map<string, { textContent: string | null }>()
	if (scriptContent !== null) {
		elements.set(SCRIPT_DATA_ID, { textContent: scriptContent })
	}

	return {
		getElementById(id: string) {
			const el = elements.get(id)
			return el ? (el as unknown as HTMLElement) : null
		},
	} as unknown as Document
}

describe('Bootstrap Loader & Error Boundary', () => {
	it('boots into Client mode by default when script tag does not exist', () => {
		const doc = createMockDocument(null)
		const result = bootstrap(doc, { checkStorage: false })
		expect(result.mode).toBe('client')
		expect(result.snapshot).toBeNull()
		expect(result.warnings).toEqual([])
	})

	it('boots into Client mode when script tag contains placeholder comment', () => {
		const doc = createMockDocument(`\n  ${PAYLOAD_PLACEHOLDER}\n  `)
		const result = bootstrap(doc, { checkStorage: false })
		expect(result.mode).toBe('client')
		expect(result.snapshot).toBeNull()
		expect(result.warnings).toEqual([])
	})

	it('boots into Client mode when script tag is empty', () => {
		const doc = createMockDocument('   ')
		const result = bootstrap(doc, { checkStorage: false })
		expect(result.mode).toBe('client')
		expect(result.snapshot).toBeNull()
	})

	it('boots into Static mode when script tag contains valid snapshot', () => {
		const validSnapshot = {
			config: {
				project: { name: 'Test Project', description: 'Desc' },
				board: {
					columns: [{ id: 'c1', name: 'Col 1', color: '#123456' }],
				},
			},
			tasks: [
				{
					id: 't1',
					title: 'Task 1',
					status: 'c1',
					summary: 'Summary 1',
				},
			],
			milestones: [],
			strategies: [],
			glossary: [],
		}

		const doc = createMockDocument(JSON.stringify(validSnapshot))
		const result = bootstrap(doc)
		expect(result.mode).toBe('static')
		expect(result.warnings).toEqual([])
		expect(result.snapshot).not.toBeNull()
		expect(result.snapshot?.config.project.name).toBe('Test Project')
		expect(result.snapshot?.tasks[0].priority).toBe('medium') // default value applied
	})

	it('handles malformed JSON syntax gracefully with diagnostic error', () => {
		const doc = createMockDocument('{ invalid json')
		const result = bootstrap(doc)
		expect(result.mode).toBe('static')
		expect(result.snapshot).toBeNull()
		expect(result.warnings.length).toBeGreaterThan(0)
		expect(result.warnings[0]).toContain(
			'Failed to parse embedded JSON payload',
		)
	})

	it('handles schema validation issues and produces detailed diagnostic warnings', () => {
		const invalidSnapshot = {
			config: {
				project: { name: 123 }, // invalid type: should be string
				board: { columns: [] },
			},
			tasks: [
				{
					id: 't1',
					title: 'Task 1',
					status: 'c1',
					priority: 99999, // invalid type: should be string
				},
			],
			milestones: [],
			strategies: [],
			glossary: [],
		}

		const doc = createMockDocument(JSON.stringify(invalidSnapshot))
		const result = bootstrap(doc)
		expect(result.mode).toBe('static')
		expect(result.warnings.length).toBeGreaterThanOrEqual(2)
		expect(result.warnings.some((w) => w.includes('config.project.name'))).toBe(
			true,
		)
		expect(result.warnings.some((w) => w.includes('tasks.0.priority'))).toBe(
			true,
		)
	})

	it('parseSnapshotContent directly returns null on empty or placeholder', () => {
		expect(parseSnapshotContent(undefined)).toBeNull()
		expect(parseSnapshotContent('')).toBeNull()
		expect(
			parseSnapshotContent('/* JOKATEKO_PAYLOAD_PLACEHOLDER */'),
		).toBeNull()
	})
})
