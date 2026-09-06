import {
	activeGlossaryId,
	activeStrategyId,
	activeTab,
	activeTaskDetailId,
	calendarCurrentDate,
	calendarViewMode,
	config,
	filters,
	glossary,
	milestones,
	setMilestoneFilter,
	strategies,
	type Tab,
	tasks,
} from './state/store'

import {
	formatYearMonth,
	formatYearWeek,
	getDateFromISOWeek,
	parseYearMonth,
	parseYearWeek,
} from './utils/date'

/**
 * Route state representation.
 */
export interface RouteState {
	tab: Tab
	milestoneId: string | null
	taskId: string | null
	strategyId: string | null
	glossaryId: string | null
	calendarPeriod: string | null
}

/**
 * Parse the current window hash into route parameters.
 * Supports:
 * - #board -> { tab: 'board', ... }
 * - #calendar -> { tab: 'calendar', calendarPeriod: null, ... }
 * - #calendar/2026-09 -> { tab: 'calendar', calendarPeriod: '2026-09', ... }
 * - #calendar/2026-W36 -> { tab: 'calendar', calendarPeriod: '2026-W36', ... }
 * - #calendar/2026-09/task/<id> -> { tab: 'calendar', calendarPeriod: '2026-09', taskId: '<id>', ... }
 * - #calendar/task/<id> -> { tab: 'calendar', calendarPeriod: null, taskId: '<id>', ... }
 * - #calendar/2026-09/milestone/<id> -> { tab: 'calendar', calendarPeriod: '2026-09', milestoneId: '<id>', ... }
 * - #strategies -> { tab: 'strategies', ... }
 * - #strategy/<id> -> { tab: 'strategies', strategyId: '<id>', ... }
 * - #glossary -> { tab: 'glossary', ... }
 * - #glossary/<id> -> { tab: 'glossary', glossaryId: '<id>', ... }
 * - #milestone/<id> -> { tab: 'board', milestoneId: '<id>', ... }
 * - #task/<id> -> { tab: 'board', taskId: '<id>', ... }
 */
export function parseHash(hash: string): RouteState {
	const raw = hash.replace(/^#\/?/, '').trim()

	if (raw.startsWith('calendar')) {
		const remainder = raw.slice(8).replace(/^\/+/, '').trim()
		if (!remainder) {
			return {
				tab: 'calendar',
				milestoneId: null,
				taskId: null,
				strategyId: null,
				glossaryId: null,
				calendarPeriod: null,
			}
		}

		const segments = remainder
			.split('/')
			.map((s) => s.trim())
			.filter(Boolean)
		let period: string | null = null
		let taskId: string | null = null
		let milestoneId: string | null = null

		for (let i = 0; i < segments.length; i++) {
			const seg = segments[i]
			if (seg === 'task' && i + 1 < segments.length) {
				taskId = segments[i + 1]
				i++
			} else if (seg === 'milestone' && i + 1 < segments.length) {
				milestoneId = segments[i + 1]
				i++
			} else if (/^\d{4}-\d{2}$/.test(seg) || /^\d{4}-W\d{1,2}$/i.test(seg)) {
				period = seg
			}
		}

		return {
			tab: 'calendar',
			milestoneId,
			taskId,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: period,
		}
	}

	if (raw.startsWith('task/')) {
		const taskId = raw.slice(5).trim()
		return {
			tab: 'board',
			milestoneId: null,
			taskId: taskId || null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		}
	}

	if (raw.startsWith('milestone/')) {
		const milestoneId = raw.slice(10).trim()
		return {
			tab: 'board',
			milestoneId: milestoneId || null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		}
	}

	if (raw.startsWith('strategy/')) {
		const strategyId = raw.slice(9).trim()
		return {
			tab: 'strategies',
			milestoneId: null,
			taskId: null,
			strategyId: strategyId || null,
			glossaryId: null,
			calendarPeriod: null,
		}
	}

	if (raw.startsWith('glossary/')) {
		const glossaryId = raw.slice(9).trim()
		return {
			tab: 'glossary',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: glossaryId || null,
			calendarPeriod: null,
		}
	}

	if (raw === 'strategies') {
		return {
			tab: 'strategies',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		}
	}

	if (raw === 'glossary') {
		return {
			tab: 'glossary',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		}
	}

	// Default to board
	return {
		tab: 'board',
		milestoneId: null,
		taskId: null,
		strategyId: null,
		glossaryId: null,
		calendarPeriod: null,
	}
}

/**
 * Updates document.title to reflect current view, task, or milestone.
 */
export function updateDocumentTitle() {
	if (typeof document === 'undefined') {
		return
	}

	const projectName = config.value.project?.name || 'Jokateko'
	const taskId = activeTaskDetailId.value

	if (taskId) {
		const task = tasks.value.find((t) => t.id === taskId)
		if (task?.title) {
			document.title = `${task.title} · ${projectName}`
			return
		}
		document.title = `${taskId} · ${projectName}`
		return
	}

	const milestoneId = filters.value.selectedMilestone
	if (milestoneId) {
		const m = milestones.value.find((m) => m.id === milestoneId)
		if (m?.title) {
			document.title = `${m.title} · ${projectName}`
			return
		}
		document.title = `${milestoneId} · ${projectName}`
		return
	}

	const tab = activeTab.value
	if (tab === 'strategies') {
		const stratId = activeStrategyId.value
		if (stratId) {
			const s = strategies.value.find((item) => item.id === stratId)
			if (s?.title) {
				document.title = `${s.title} · ${projectName}`
				return
			}
			document.title = `${stratId} · ${projectName}`
			return
		}
		document.title = `Strategies · ${projectName}`
		return
	}

	if (tab === 'glossary') {
		const glossId = activeGlossaryId.value
		if (glossId) {
			const term = glossary.value.find((item) => item.id === glossId)
			if (term?.title) {
				document.title = `${term.title} · ${projectName}`
				return
			}
			document.title = `${glossId} · ${projectName}`
			return
		}
		document.title = `Glossary · ${projectName}`
		return
	}

	if (tab === 'calendar') {
		const period =
			calendarViewMode.value === 'week'
				? formatYearWeek(calendarCurrentDate.value)
				: formatYearMonth(calendarCurrentDate.value)
		document.title = `Calendar (${period}) · ${projectName}`
		return
	}

	document.title = `Board · ${projectName}`
}

/**
 * Synchronize application state from the current window.location.hash.
 */
export function syncFromHash() {
	if (typeof window === 'undefined') {
		return
	}

	const currentHash = window.location.hash
	const route = parseHash(currentHash)

	// Apply tab
	if (activeTab.value !== route.tab) {
		activeTab.value = route.tab
	}

	// Apply calendar period & view mode
	if (route.tab === 'calendar' && route.calendarPeriod) {
		const parsedMonth = parseYearMonth(route.calendarPeriod)
		if (parsedMonth) {
			calendarViewMode.value = 'month'
			calendarCurrentDate.value = new Date(
				parsedMonth.year,
				parsedMonth.month,
				1,
			)
		} else {
			const parsedWeek = parseYearWeek(route.calendarPeriod)
			if (parsedWeek) {
				calendarViewMode.value = 'week'
				calendarCurrentDate.value = getDateFromISOWeek(
					parsedWeek.year,
					parsedWeek.week,
				)
			}
		}
	}

	// Apply milestone filter (affects board & calendar)
	if (route.milestoneId !== null) {
		if (filters.value.selectedMilestone !== route.milestoneId) {
			setMilestoneFilter(route.milestoneId)
		}
	} else if (!route.taskId && filters.value.selectedMilestone !== null) {
		setMilestoneFilter(null)
	}

	// Apply task modal
	if (activeTaskDetailId.value !== route.taskId) {
		activeTaskDetailId.value = route.taskId
	}

	// Apply active strategy
	if (activeStrategyId.value !== (route.strategyId || null)) {
		activeStrategyId.value = route.strategyId || null
	}

	// Apply active glossary
	if (activeGlossaryId.value !== (route.glossaryId || null)) {
		activeGlossaryId.value = route.glossaryId || null
	}

	updateDocumentTitle()
}

/**
 * Navigate to calendar with optional period, taskId, and milestoneId.
 */
export function navigateToCalendar(options?: {
	period?: string | null
	taskId?: string | null
	milestoneId?: string | null
	replace?: boolean
}) {
	const period =
		options?.period !== undefined
			? options.period
			: calendarViewMode.value === 'week'
				? formatYearWeek(calendarCurrentDate.value)
				: formatYearMonth(calendarCurrentDate.value)

	const parts: string[] = ['calendar']
	if (period) {
		parts.push(period)
	}
	if (options?.milestoneId) {
		parts.push('milestone', options.milestoneId)
	}
	if (options?.taskId) {
		parts.push('task', options.taskId)
	}
	navigateTo(parts.join('/'), options?.replace ?? false)
}

/**
 * Navigate to a specific route and push to browser history.
 * @param path e.g. 'board', 'strategies', 'glossary', 'task/123', 'milestone/m1', 'strategy/strat-01', 'glossary/term-01'
 */
export function navigateTo(path: string, replace = false) {
	if (typeof window === 'undefined') {
		return
	}

	const clean = path.replace(/^#\/?/, '').trim()
	const targetHash = `#${clean}`

	if (window.location.hash !== targetHash) {
		if (replace) {
			window.history.replaceState(null, '', targetHash)
		} else {
			window.history.pushState(null, '', targetHash)
		}
	}

	syncFromHash()
}

/**
 * Initialize router event listeners.
 */
export function initRouter() {
	if (typeof window === 'undefined') {
		return () => {}
	}

	// Initial sync from current URL
	syncFromHash()

	const handleLocationChange = () => {
		syncFromHash()
	}

	window.addEventListener('popstate', handleLocationChange)
	window.addEventListener('hashchange', handleLocationChange)

	return () => {
		window.removeEventListener('popstate', handleLocationChange)
		window.removeEventListener('hashchange', handleLocationChange)
	}
}
