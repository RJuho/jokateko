/**
 * Markdown checklist helpers shared by task editors.
 * Mirrors the backend Goldmark criteria extraction: `- [ ]` / `* [x]` list items.
 */
const CHECKBOX_LINE = /^\s*[-*+]\s+\[[ xX]\]/
const CHECKED_LINE = /^\s*[-*+]\s+\[[xX]\]/

export interface CriteriaCount {
	total_criteria: number
	completed_criteria: number
}

export function countCriteria(body: string): CriteriaCount {
	let total = 0
	let completed = 0
	for (const line of body.split('\n')) {
		if (CHECKBOX_LINE.test(line)) {
			total++
			if (CHECKED_LINE.test(line)) {
				completed++
			}
		}
	}
	return { total_criteria: total, completed_criteria: completed }
}

/**
 * Sets the checked state of the n-th (1-based) checklist item.
 * Returns null when no such item exists.
 */
export function setCriterionChecked(
	body: string,
	index: number,
	checked: boolean,
): string | null {
	const lines = body.split('\n')
	let count = 0
	for (let i = 0; i < lines.length; i++) {
		if (CHECKBOX_LINE.test(lines[i])) {
			count++
			if (count === index) {
				lines[i] = checked
					? lines[i].replace(/\[ \]/, '[x]')
					: lines[i].replace(/\[[xX]\]/, '[ ]')
				return lines.join('\n')
			}
		}
	}
	return null
}
