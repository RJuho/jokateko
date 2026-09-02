import { describe, expect, it } from 'bun:test'
import { getContrastTextColor } from './colors'

describe('getContrastTextColor', () => {
	it('returns white text for dark or vibrant priority colors', () => {
		expect(getContrastTextColor('#ef4444')).toBe('#ffffff') // Red
		expect(getContrastTextColor('#f97316')).toBe('#ffffff') // Orange
		expect(getContrastTextColor('#3b82f6')).toBe('#ffffff') // Blue
		expect(getContrastTextColor('#000000')).toBe('#ffffff') // Black
	})

	it('returns dark text for light or pastel priority colors', () => {
		expect(getContrastTextColor('#eab308')).toBe('#0f172a') // Yellow
		expect(getContrastTextColor('#ffffff')).toBe('#0f172a') // White
		expect(getContrastTextColor('#fef08a')).toBe('#0f172a') // Light yellow
	})

	it('handles shorthand 3-digit hex colors', () => {
		expect(getContrastTextColor('#fff')).toBe('#0f172a')
		expect(getContrastTextColor('#000')).toBe('#ffffff')
	})

	it('falls back to white for undefined or invalid hex input', () => {
		expect(getContrastTextColor(undefined)).toBe('#ffffff')
		expect(getContrastTextColor('')).toBe('#ffffff')
		expect(getContrastTextColor('red')).toBe('#ffffff')
	})
})
