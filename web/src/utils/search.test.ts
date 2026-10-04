import { describe, expect, it } from 'bun:test'
import { matchesQuery, normalizeQuery } from './search'

describe('matchesQuery', () => {
	const item = {
		id: 'task-42',
		title: 'Fix Login',
		summary: 'Session bug',
		body: 'Details about OAuth',
		tags: ['Backend'],
	}

	it('matches every field case-insensitively', () => {
		for (const q of ['task-42', 'login', 'SESSION', 'oauth', 'backend']) {
			expect(matchesQuery(item, normalizeQuery(q))).toBe(true)
		}
	})

	it('empty query matches', () => {
		expect(matchesQuery(item, normalizeQuery('   '))).toBe(true)
	})

	it('rejects non-matching query', () => {
		expect(matchesQuery(item, normalizeQuery('frontend'))).toBe(false)
	})

	it('tolerates missing optional fields', () => {
		expect(matchesQuery({ id: 'x', title: 'Y' }, 'z')).toBe(false)
	})
})
