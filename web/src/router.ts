import {
	activeGlossaryId,
	activeStrategyId,
	activeTab,
	activeTaskDetailId,
	config,
	filters,
	glossary,
	milestones,
	setMilestoneFilter,
	strategies,
	type Tab,
	tasks,
} from './state/store'

/**
 * Route state representation.
 */
export interface RouteState {
	tab: Tab
	milestoneId: string | null
	taskId: string | null
	strategyId: string | null
	glossaryId: string | null
}

/**
 * Parse the current window hash into route parameters.
 * Supports:
 * - #board -> { tab: 'board', milestoneId: null, taskId: null, strategyId: null, glossaryId: null }
 * - #strategies -> { tab: 'strategies', milestoneId: null, taskId: null, strategyId: null, glossaryId: null }
 * - #strategy/<id> -> { tab: 'strategies', milestoneId: null, taskId: null, strategyId: '<id>', glossaryId: null }
 * - #glossary -> { tab: 'glossary', milestoneId: null, taskId: null, strategyId: null, glossaryId: null }
 * - #glossary/<id> -> { tab: 'glossary', milestoneId: null, taskId: null, strategyId: null, glossaryId: '<id>' }
 * - #milestone/<id> -> { tab: 'board', milestoneId: '<id>', taskId: null, strategyId: null, glossaryId: null }
 * - #task/<id> -> { tab: 'board', milestoneId: null, taskId: '<id>', strategyId: null, glossaryId: null }
 */
export function parseHash(hash: string): RouteState {
	const raw = hash.replace(/^#\/?/, '').trim()

	if (raw.startsWith('task/')) {
		const taskId = raw.slice(5).trim()
		return {
			tab: 'board',
			milestoneId: null,
			taskId: taskId || null,
			strategyId: null,
			glossaryId: null,
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
		}
	}

	if (raw === 'strategies') {
		return {
			tab: 'strategies',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
		}
	}

	if (raw === 'glossary') {
		return {
			tab: 'glossary',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
		}
	}

	// Default to board
	return {
		tab: 'board',
		milestoneId: null,
		taskId: null,
		strategyId: null,
		glossaryId: null,
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

	// Apply milestone filter (only affects board)
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
