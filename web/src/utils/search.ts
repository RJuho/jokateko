export interface Searchable {
	id: string
	title: string
	summary?: string
	body?: string
	tags?: string[]
}

/** Normalizes raw user input into the form expected by matchesQuery. */
export function normalizeQuery(query: string): string {
	return query.trim().toLowerCase()
}

/**
 * Case-insensitive match over id, title, summary, body and tags.
 * `normalizedQuery` must come from normalizeQuery; an empty query matches everything.
 */
export function matchesQuery(
	item: Searchable,
	normalizedQuery: string,
): boolean {
	if (normalizedQuery === '') {
		return true
	}
	return (
		item.title.toLowerCase().includes(normalizedQuery)
		|| item.id.toLowerCase().includes(normalizedQuery)
		|| (item.summary || '').toLowerCase().includes(normalizedQuery)
		|| (item.body || '').toLowerCase().includes(normalizedQuery)
		|| (item.tags?.some((t) => t.toLowerCase().includes(normalizedQuery))
			?? false)
	)
}
