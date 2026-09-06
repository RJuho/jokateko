import { describe, expect, it } from 'bun:test'
import { sampleSnapshot } from './fixtures/sampleData'
import {
	navigateTo,
	parseHash,
	syncFromHash,
	updateDocumentTitle,
} from './router'
import {
	activeGlossaryId,
	activeStrategyId,
	activeTab,
	activeTaskDetailId,
	filters,
	initFromSnapshot,
	setMilestoneFilter,
} from './state/store'

// Setup lightweight DOM mocks for headless Bun test runner
const mockLocation = { hash: '' }
const mockWindow = {
	location: mockLocation,
	history: {
		pushState: (_data: unknown, _unused: string, url: string) => {
			mockLocation.hash = url
		},
		replaceState: (_data: unknown, _unused: string, url: string) => {
			mockLocation.hash = url
		},
	},
	addEventListener: () => {},
	removeEventListener: () => {},
}
const mockDocument = { title: '' }

;(globalThis as unknown as { window: unknown }).window = mockWindow
;(globalThis as unknown as { document: unknown }).document = mockDocument
;(globalThis as unknown as { history: unknown }).history = mockWindow.history

describe('Router & URL Anchor Parser', () => {
	it('parses empty or standard anchor hashes correctly', () => {
		expect(parseHash('')).toEqual({
			tab: 'board',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#')).toEqual({
			tab: 'board',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#board')).toEqual({
			tab: 'board',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#strategies')).toEqual({
			tab: 'strategies',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#glossary')).toEqual({
			tab: 'glossary',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
	})

	it('parses task, milestone, strategy, and glossary routes accurately', () => {
		expect(parseHash('#task/260901-user-auth')).toEqual({
			tab: 'board',
			milestoneId: null,
			taskId: '260901-user-auth',
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#/task/260901-user-auth')).toEqual({
			tab: 'board',
			milestoneId: null,
			taskId: '260901-user-auth',
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#milestone/m1-mvp-release')).toEqual({
			tab: 'board',
			milestoneId: 'm1-mvp-release',
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#/milestone/m1-mvp-release')).toEqual({
			tab: 'board',
			milestoneId: 'm1-mvp-release',
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#strategy/strat-arch-01')).toEqual({
			tab: 'strategies',
			milestoneId: null,
			taskId: null,
			strategyId: 'strat-arch-01',
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#/strategy/strat-arch-01')).toEqual({
			tab: 'strategies',
			milestoneId: null,
			taskId: null,
			strategyId: 'strat-arch-01',
			glossaryId: null,
			calendarPeriod: null,
		})
		expect(parseHash('#glossary/term-tasks-as-code')).toEqual({
			tab: 'glossary',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: 'term-tasks-as-code',
			calendarPeriod: null,
		})
		expect(parseHash('#/glossary/term-tasks-as-code')).toEqual({
			tab: 'glossary',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: 'term-tasks-as-code',
			calendarPeriod: null,
		})
	})

	it('updates document.title dynamically based on route and store entities', () => {
		initFromSnapshot(sampleSnapshot)

		// 1. Board tab title
		activeTab.value = 'board'
		activeTaskDetailId.value = null
		setMilestoneFilter(null)
		updateDocumentTitle()
		expect(mockDocument.title).toBe('Board · Jokateko Kanban')

		// 2. Strategies tab title
		activeTab.value = 'strategies'
		activeStrategyId.value = null
		updateDocumentTitle()
		expect(mockDocument.title).toBe('Strategies · Jokateko Kanban')

		// 3. Strategy specific title
		activeStrategyId.value = 'strat-progressive-disclosure'
		updateDocumentTitle()
		expect(mockDocument.title).toBe(
			'Tiered Progressive Disclosure · Jokateko Kanban',
		)

		// 4. Glossary tab title
		activeTab.value = 'glossary'
		activeGlossaryId.value = null
		updateDocumentTitle()
		expect(mockDocument.title).toBe('Glossary · Jokateko Kanban')

		// 5. Glossary specific title
		activeGlossaryId.value = 'term-tasks-as-code'
		updateDocumentTitle()
		expect(mockDocument.title).toBe('Tasks-as-Code · Jokateko Kanban')

		// 6. Milestone title
		activeTab.value = 'board'
		activeStrategyId.value = null
		activeGlossaryId.value = null
		setMilestoneFilter('m1-mvp-release')
		updateDocumentTitle()
		expect(mockDocument.title).toBe('v1.0 MVP Launch · Jokateko Kanban')

		// 7. Task title
		activeTaskDetailId.value = '260901-user-auth'
		updateDocumentTitle()
		expect(mockDocument.title).toBe(
			'Implement OAuth2 and Session Authentication · Jokateko Kanban',
		)
	})

	it('syncs state accurately from hash', () => {
		initFromSnapshot(sampleSnapshot)

		mockLocation.hash = '#task/260901-user-auth'
		syncFromHash()
		expect(activeTab.value).toBe('board')
		expect(activeTaskDetailId.value).toBe('260901-user-auth')

		mockLocation.hash = '#milestone/m1-mvp-release'
		syncFromHash()
		expect(activeTab.value).toBe('board')
		expect(activeTaskDetailId.value).toBe(null)
		expect(filters.value.selectedMilestone).toBe('m1-mvp-release')

		mockLocation.hash = '#strategies'
		syncFromHash()
		expect(activeTab.value).toBe('strategies')
		expect(activeTaskDetailId.value).toBe(null)

		mockLocation.hash = '#strategy/strat-arch-01'
		syncFromHash()
		expect(activeTab.value).toBe('strategies')
		expect(activeStrategyId.value).toBe('strat-arch-01')

		mockLocation.hash = '#glossary/term-tasks-as-code'
		syncFromHash()
		expect(activeTab.value).toBe('glossary')
		expect(activeGlossaryId.value).toBe('term-tasks-as-code')

		// navigateTo helper
		navigateTo('glossary')
		expect(mockLocation.hash).toBe('#glossary')
		expect(activeTab.value).toBe('glossary')

		navigateTo('task/260901-user-auth')
		expect(mockLocation.hash).toBe('#task/260901-user-auth')
		expect(activeTaskDetailId.value).toBe('260901-user-auth')

		// calendar navigation
		mockLocation.hash = '#calendar'
		syncFromHash()
		expect(activeTab.value).toBe('calendar')

		mockLocation.hash = '#calendar/2026-09'
		syncFromHash()
		expect(activeTab.value).toBe('calendar')

		mockLocation.hash = '#calendar/2026-09/task/260901-user-auth'
		syncFromHash()
		expect(activeTab.value).toBe('calendar')
		expect(activeTaskDetailId.value).toBe('260901-user-auth')
	})

	it('parses calendar routes with periods, tasks, and milestones', () => {
		expect(parseHash('#calendar')).toEqual({
			tab: 'calendar',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})

		expect(parseHash('#calendar/2026-09')).toEqual({
			tab: 'calendar',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: '2026-09',
		})

		expect(parseHash('#calendar/2026-W36')).toEqual({
			tab: 'calendar',
			milestoneId: null,
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: '2026-W36',
		})

		expect(parseHash('#calendar/2026-09/task/task-abc')).toEqual({
			tab: 'calendar',
			milestoneId: null,
			taskId: 'task-abc',
			strategyId: null,
			glossaryId: null,
			calendarPeriod: '2026-09',
		})

		expect(parseHash('#calendar/task/task-xyz')).toEqual({
			tab: 'calendar',
			milestoneId: null,
			taskId: 'task-xyz',
			strategyId: null,
			glossaryId: null,
			calendarPeriod: null,
		})

		expect(parseHash('#calendar/2026-09/milestone/m1-mvp')).toEqual({
			tab: 'calendar',
			milestoneId: 'm1-mvp',
			taskId: null,
			strategyId: null,
			glossaryId: null,
			calendarPeriod: '2026-09',
		})
	})
})
