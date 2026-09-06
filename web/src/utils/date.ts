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

/**
 * Formats a Date object into 'YYYY-MM-DD' using local date components.
 */
export function formatDateKey(d: Date): string {
	const y = d.getFullYear()
	const m = String(d.getMonth() + 1).padStart(2, '0')
	const day = String(d.getDate()).padStart(2, '0')
	return `${y}-${m}-${day}`
}

/**
 * Extracts 'YYYY-MM-DD' from an ISO string, date string, or timestamp.
 * Returns null if string is empty or invalid date.
 */
export function extractDateKey(isoOrDateString?: string | null): string | null {
	if (!isoOrDateString) return null
	const trimmed = isoOrDateString.trim()
	if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
		return trimmed
	}
	const d = new Date(trimmed)
	if (Number.isNaN(d.getTime())) {
		return null
	}
	return formatDateKey(d)
}

/**
 * Formats a Date into 'YYYY-MM'.
 */
export function formatYearMonth(d: Date): string {
	const y = d.getFullYear()
	const m = String(d.getMonth() + 1).padStart(2, '0')
	return `${y}-${m}`
}

/**
 * Parses 'YYYY-MM' string into { year, month } with 0-indexed month (0..11).
 * Returns null if invalid.
 */
export function parseYearMonth(
	str?: string | null,
): { year: number; month: number } | null {
	if (!str) return null
	const m = str.trim().match(/^(\d{4})-(\d{2})$/)
	if (!m) return null
	const year = Number.parseInt(m[1], 10)
	const month1 = Number.parseInt(m[2], 10)
	if (month1 < 1 || month1 > 12) return null
	return { year, month: month1 - 1 }
}

/**
 * Calculates ISO-8601 week number and week-year (Monday-based).
 */
export function getISOWeek(d: Date): { year: number; week: number } {
	const target = new Date(d.getFullYear(), d.getMonth(), d.getDate())
	const dayNr = (target.getDay() + 6) % 7 // Mon = 0, Sun = 6
	target.setDate(target.getDate() - dayNr + 3) // Nearest Thursday
	const firstThursday = new Date(target.getFullYear(), 0, 4)
	const dayDiff = (target.getTime() - firstThursday.getTime()) / 86400000
	const week =
		1 + Math.round((dayDiff - 3 + ((firstThursday.getDay() + 6) % 7)) / 7)
	return { year: target.getFullYear(), week }
}

/**
 * Formats Date into 'YYYY-Www' (e.g. '2026-W36').
 */
export function formatYearWeek(d: Date): string {
	const { year, week } = getISOWeek(d)
	return `${year}-W${String(week).padStart(2, '0')}`
}

/**
 * Parses 'YYYY-Www' into { year, week }.
 */
export function parseYearWeek(
	str?: string | null,
): { year: number; week: number } | null {
	if (!str) return null
	const m = str.trim().match(/^(\d{4})-W(\d{1,2})$/i)
	if (!m) return null
	const year = Number.parseInt(m[1], 10)
	const week = Number.parseInt(m[2], 10)
	if (week < 1 || week > 53) return null
	return { year, week }
}

/**
 * Returns Monday Date for given ISO week.
 */
export function getDateFromISOWeek(year: number, week: number): Date {
	const simple = new Date(year, 0, 1 + (week - 1) * 7)
	const dayOfWeek = (simple.getDay() + 6) % 7
	const isoWeekStart = new Date(simple)
	if (dayOfWeek <= 3) {
		isoWeekStart.setDate(simple.getDate() - dayOfWeek)
	} else {
		isoWeekStart.setDate(simple.getDate() + 7 - dayOfWeek)
	}
	return isoWeekStart
}

export interface CalendarDayInfo {
	date: Date
	dateKey: string // 'YYYY-MM-DD'
	dayOfMonth: number
	isCurrentMonth: boolean
	isToday: boolean
	isWeekend: boolean
	dayOfWeekIndex: number // 0 = Mon, 6 = Sun
}

export interface CalendarWeekInfo {
	weekNumber: number
	weekYear: number
	days: CalendarDayInfo[]
}

/**
 * Builds the array of weeks (Monday to Sunday) for a given month.
 * Days from preceding or subsequent months are included with isCurrentMonth: false.
 */
export function getMonthCalendarWeeks(
	year: number,
	month: number,
): CalendarWeekInfo[] {
	const todayKey = formatDateKey(new Date())
	const firstOfMonth = new Date(year, month, 1)
	const lastOfMonth = new Date(year, month + 1, 0)

	// First day of calendar grid: Monday of the week containing firstOfMonth
	const startDayOfWeek = (firstOfMonth.getDay() + 6) % 7 // 0 = Mon, 6 = Sun
	const startDate = new Date(year, month, 1 - startDayOfWeek)

	// End day: Sunday of the week containing lastOfMonth
	const endDayOfWeek = (lastOfMonth.getDay() + 6) % 7
	const endDate = new Date(year, month + 1, 0 + (6 - endDayOfWeek))

	const weeks: CalendarWeekInfo[] = []
	const curr = new Date(startDate)

	while (curr <= endDate) {
		const days: CalendarDayInfo[] = []
		const weekMeta = getISOWeek(curr)

		for (let i = 0; i < 7; i++) {
			const d = new Date(curr)
			const dateKey = formatDateKey(d)
			const dayOfWeek = (d.getDay() + 6) % 7
			days.push({
				date: d,
				dateKey,
				dayOfMonth: d.getDate(),
				isCurrentMonth: d.getMonth() === month && d.getFullYear() === year,
				isToday: dateKey === todayKey,
				isWeekend: dayOfWeek === 5 || dayOfWeek === 6,
				dayOfWeekIndex: dayOfWeek,
			})
			curr.setDate(curr.getDate() + 1)
		}

		weeks.push({
			weekNumber: weekMeta.week,
			weekYear: weekMeta.year,
			days,
		})
	}

	return weeks
}

/**
 * Builds a single 7-day week (Monday to Sunday) for Week View.
 */
export function getWeekCalendar(year: number, week: number): CalendarWeekInfo {
	const monday = getDateFromISOWeek(year, week)
	const todayKey = formatDateKey(new Date())
	const days: CalendarDayInfo[] = []
	const curr = new Date(monday)

	for (let i = 0; i < 7; i++) {
		const d = new Date(curr)
		const dateKey = formatDateKey(d)
		const dayOfWeek = (d.getDay() + 6) % 7
		days.push({
			date: d,
			dateKey,
			dayOfMonth: d.getDate(),
			isCurrentMonth: true,
			isToday: dateKey === todayKey,
			isWeekend: dayOfWeek === 5 || dayOfWeek === 6,
			dayOfWeekIndex: dayOfWeek,
		})
		curr.setDate(curr.getDate() + 1)
	}

	return {
		weekNumber: week,
		weekYear: year,
		days,
	}
}

/**
 * Formats a localized Month Year title, e.g. "September 2026"
 */
export function formatMonthYearTitle(
	year: number,
	month: number,
	monthName?: string,
): string {
	if (monthName) {
		return `${monthName} ${year}`
	}
	try {
		return new Intl.DateTimeFormat(undefined, {
			month: 'long',
			year: 'numeric',
		}).format(new Date(year, month, 1))
	} catch {
		const months = [
			'January',
			'February',
			'March',
			'April',
			'May',
			'June',
			'July',
			'August',
			'September',
			'October',
			'November',
			'December',
		]
		return `${months[month] ?? ''} ${year}`
	}
}

/**
 * Formats a localized Week title, e.g. "36 · Sep 2026"
 * Note: ISO week belongs to the year/month containing its Thursday.
 * Omits 'W' prefix per user preference.
 */
export function formatWeekTitle(
	year: number,
	week: number,
	monthName?: string,
): string {
	const monday = getDateFromISOWeek(year, week)
	const thursday = new Date(monday)
	thursday.setDate(monday.getDate() + 3)
	const fallbackMonth = new Intl.DateTimeFormat(undefined, {
		month: 'short',
	}).format(thursday)
	const mName = monthName || fallbackMonth
	return `${week} · ${mName} ${thursday.getFullYear()}`
}
