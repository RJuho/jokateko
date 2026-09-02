/**
 * Formats an ISO datetime string or timestamp into the user's current browser locale.
 * Example in en-US: 'Sep 2, 2026, 05:30 AM'
 * Example in fi-FI: '2. syysk. 2026 klo 5.30'
 */
export function formatBrowserDateTime(isoOrDateString?: string | null): string {
	if (!isoOrDateString) return ''
	const d = new Date(isoOrDateString)
	if (Number.isNaN(d.getTime())) {
		return isoOrDateString
	}
	try {
		return new Intl.DateTimeFormat(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
		}).format(d)
	} catch {
		return d.toLocaleString()
	}
}
