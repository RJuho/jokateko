import { describe, expect, it } from 'bun:test'
import { countCriteria, setCriterionChecked } from './criteria'

describe('countCriteria', () => {
	it('counts dash, star and plus checklist items', () => {
		const body = [
			'## Criteria',
			'- [ ] one',
			'* [x] two',
			'  + [X] nested three',
			'- not a checkbox',
			'-[ ] missing space',
		].join('\n')
		expect(countCriteria(body)).toEqual({
			total_criteria: 3,
			completed_criteria: 2,
		})
	})

	it('returns zero for empty body', () => {
		expect(countCriteria('')).toEqual({
			total_criteria: 0,
			completed_criteria: 0,
		})
	})
})

describe('setCriterionChecked', () => {
	const body = '- [ ] a\ntext\n* [x] b\n- [ ] c'

	it('checks the n-th item', () => {
		expect(setCriterionChecked(body, 3, true)).toBe(
			'- [ ] a\ntext\n* [x] b\n- [x] c',
		)
	})

	it('unchecks the n-th item', () => {
		expect(setCriterionChecked(body, 2, false)).toBe(
			'- [ ] a\ntext\n* [ ] b\n- [ ] c',
		)
	})

	it('returns null for out-of-range index', () => {
		expect(setCriterionChecked(body, 4, true)).toBeNull()
	})
})
