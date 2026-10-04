/**
 * Copies text to the clipboard. Returns false instead of throwing when the
 * Clipboard API is unavailable (e.g. non-secure http origin) or permission is denied.
 */
export async function copyToClipboard(text: string): Promise<boolean> {
	try {
		if (typeof navigator === 'undefined' || !navigator.clipboard) {
			return false
		}
		await navigator.clipboard.writeText(text)
		return true
	} catch (err) {
		console.warn('Clipboard write failed:', err)
		return false
	}
}
