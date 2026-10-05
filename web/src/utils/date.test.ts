import { describe, expect, it } from 'bun:test'
import {
	extractDateKey,
	formatDateKey,
	formatMonthYearTitle,
	formatWeekTitle,
	formatYearMonth,
	formatYearWeek,
	getDateFromISOWeek,
	getISOWeek,
	getMonthCalendarWeeks,
	getWeekCalendar,
	monthName,
	parseYearMonth,
	parseYearWeek,
	weekdayName,
} from './date'

describe('Calendar Date Utilities', () => {
	it('formats date key and extracts date key correctly', () => {
		const d = new Date(2026, 8, 6) // Sep 6, 2026
		expect(formatDateKey(d)).toBe('2026-09-06')
		expect(extractDateKey('2026-09-06T12:00:00Z')).toBe('2026-09-06')
		expect(extractDateKey('2026-09-06')).toBe('2026-09-06')
		expect(extractDateKey('')).toBeNull()
		expect(extractDateKey(null)).toBeNull()
		expect(extractDateKey('invalid')).toBeNull()
	})

	it('formats and parses year-month string', () => {
		const d = new Date(2026, 8, 1)
		expect(formatYearMonth(d)).toBe('2026-09')
		expect(parseYearMonth('2026-09')).toEqual({ year: 2026, month: 8 })
		expect(parseYearMonth('2026-13')).toBeNull()
		expect(parseYearMonth('invalid')).toBeNull()
	})

	it('computes ISO week numbers accurately', () => {
		// Sep 6, 2026 is Sunday of week 36
		const d = new Date(2026, 8, 6)
		const iso = getISOWeek(d)
		expect(iso.year).toBe(2026)
		expect(iso.week).toBe(36)
		expect(formatYearWeek(d)).toBe('2026-W36')
		expect(parseYearWeek('2026-W36')).toEqual({ year: 2026, week: 36 })
	})

	it('derives Monday date from ISO week', () => {
		const mon = getDateFromISOWeek(2026, 36)
		expect(formatDateKey(mon)).toBe('2026-08-31')
	})

	it('generates full monthly calendar grid spanning Monday to Sunday', () => {
		// Sep 2026 starts on Tuesday Sep 1, ends on Wednesday Sep 30
		const weeks = getMonthCalendarWeeks(2026, 8)
		expect(weeks.length).toBeGreaterThanOrEqual(5)

		// First day of first week must be Monday Aug 31
		expect(weeks[0].days[0].dateKey).toBe('2026-08-31')
		expect(weeks[0].days[0].isCurrentMonth).toBe(false)
		expect(weeks[0].days[0].dayOfWeekIndex).toBe(0) // Monday

		// Second day of first week is Sep 1
		expect(weeks[0].days[1].dateKey).toBe('2026-09-01')
		expect(weeks[0].days[1].isCurrentMonth).toBe(true)

		// Last day of last week must be Sunday
		const lastWeek = weeks[weeks.length - 1]
		expect(lastWeek.days[6].dayOfWeekIndex).toBe(6) // Sunday
	})

	it('generates 7-day week calendar', () => {
		const week = getWeekCalendar(2026, 36)
		expect(week.weekNumber).toBe(36)
		expect(week.days.length).toBe(7)
		expect(week.days[0].dateKey).toBe('2026-08-31')
		expect(week.days[6].dateKey).toBe('2026-09-06')
	})

	it('formats week title with week number, month and year without W prefix', () => {
		expect(formatWeekTitle(2026, 36, 'en-US')).toBe('36 · Sep 2026')
		expect(formatWeekTitle(2026, 36, 'fi-FI')).toBe('36 · syys 2026')
	})

	it('formats month title in the given locale', () => {
		expect(formatMonthYearTitle(2026, 8, 'en-US')).toBe('September 2026')
		expect(formatMonthYearTitle(2026, 8, 'fi-FI')).toBe('syyskuu 2026')
	})
})

describe('Intl weekday and month names', () => {
	it('returns weekdays Monday first, keeping the locale capitalisation', () => {
		const en = Array.from({ length: 7 }, (_, i) =>
			weekdayName(i, 'short', 'en-US'),
		)
		expect(en).toEqual(['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'])
		expect(weekdayName(0, 'short', 'fi-FI')).toBe('ma')
		expect(weekdayName(6, 'long', 'fi-FI')).toBe('sunnuntai')
		expect(weekdayName(0, 'long', 'en-US')).toBe('Monday')
	})

	it('returns stand-alone month names', () => {
		expect(monthName(0, 'long', 'en-US')).toBe('January')
		expect(monthName(11, 'short', 'en-US')).toBe('Dec')
		expect(monthName(8, 'long', 'fi-FI')).toBe('syyskuu')
		expect(monthName(8, 'short', 'fi-FI')).toBe('syys')
	})

	it('uses the browser locale when locale is undefined', () => {
		const browser = new Intl.DateTimeFormat(undefined, {
			month: 'long',
		}).format(new Date(2024, 2, 1))
		expect(monthName(2)).toBe(browser)
	})
})
