/**
 * Calculates optimal text color (#ffffff or #0f172a) based on hex background luminance.
 * Uses standard YIQ formula.
 */
export function getContrastTextColor(hexColor?: string): string {
	if (!hexColor?.startsWith('#')) return '#ffffff'
	const hex = hexColor.replace('#', '')
	let r = 255
	let g = 255
	let b = 255

	if (hex.length === 3) {
		r = parseInt(hex[0] + hex[0], 16)
		g = parseInt(hex[1] + hex[1], 16)
		b = parseInt(hex[2] + hex[2], 16)
	} else if (hex.length === 6) {
		r = parseInt(hex.slice(0, 2), 16)
		g = parseInt(hex.slice(2, 4), 16)
		b = parseInt(hex.slice(4, 6), 16)
	} else {
		return '#ffffff'
	}

	const yiq = (r * 299 + g * 587 + b * 114) / 1000
	return yiq >= 160 ? '#0f172a' : '#ffffff'
}
