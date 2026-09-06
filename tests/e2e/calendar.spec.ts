import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

test.describe('Full-Screen Interactive Calendar View E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customMilestones: [
				{
					id: 'm-launch-v1',
					title: 'Launch Version 1.0',
					status: 'open',
					target_date: '2026-09-18',
					summary: 'Final stabilization and public release',
				},
			],
			customTasks: [
				{
					id: 'task-active-start',
					title: 'Milestone Start Task',
					status: 'in_progress',
					priority: 'medium',
					milestone: 'm-launch-v1',
					tags: ['backend'],
					summary: 'Initial kickoff',
					target_at: '2026-09-08T08:00:00Z',
				},
				{
					id: 'task-active-targeted',
					title: 'Targeted Active Task',
					status: 'in_progress',
					priority: 'high',
					milestone: 'm-launch-v1',
					tags: ['frontend'],
					summary: 'Scheduled for middle of September',
					target_at: '2026-09-15T12:00:00Z',
				},
				{
					id: 'task-active-untargeted',
					title: 'Untargeted Active Task',
					status: 'ready',
					priority: 'medium',
					tags: ['backend'],
					summary: 'Active task without deadline',
					changed_at: '2026-09-10T09:00:00Z',
				},
				{
					id: 'task-completed',
					title: 'Completed Task',
					status: 'done',
					priority: 'low',
					tags: ['docs'],
					summary: 'Finished documentation task',
					changed_at: '2026-09-22T14:00:00Z',
				},
				{
					id: 'task-overflow-1',
					title: 'Overflow Task 1',
					status: 'in_progress',
					priority: 'low',
					tags: ['backend'],
					summary: 'First extra task on 2026-09-15',
					target_at: '2026-09-15T13:00:00Z',
				},
				{
					id: 'task-overflow-2',
					title: 'Overflow Task 2',
					status: 'in_progress',
					priority: 'medium',
					tags: ['frontend'],
					summary: 'Second extra task on 2026-09-15',
					target_at: '2026-09-15T14:00:00Z',
				},
				{
					id: 'task-overflow-3',
					title: 'Overflow Task 3',
					status: 'in_progress',
					priority: 'high',
					tags: ['release'],
					summary: 'Third extra task on 2026-09-15',
					target_at: '2026-09-15T15:00:00Z',
				},
			],
		})
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('navigates to calendar via header tab and URL anchor', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Click Calendar tab in Header
		const calTab = page.locator('[data-testid="nav-tab-calendar"]')
		await expect(calTab).toBeVisible()
		await calTab.click()

		// 2. Verify URL hash updates to #calendar and view is visible
		await expect(page).toHaveURL(/#calendar/)
		const calendarView = page.locator('[data-testid="calendar-view"]')
		await expect(calendarView).toBeVisible()

		// 3. Verify Month and Week view buttons exist
		const monthBtn = page.locator('[data-testid="calendar-view-month"]')
		const weekBtn = page.locator('[data-testid="calendar-view-week"]')
		await expect(monthBtn).toBeVisible()
		await expect(weekBtn).toBeVisible()

		// 4. Verify Week row indicators exist
		const weekRow = page.locator('[data-testid^="calendar-week-row-"]').first()
		await expect(weekRow).toBeVisible()
		const weekNum = page.locator('[data-testid="calendar-week-number"]').first()
		await expect(weekNum).toBeVisible()
	})

	test('supports deep linking to specific month and navigating periods', async ({
		page,
	}) => {
		// Deep link to September 2026
		await page.goto(`${server.url}/#calendar/2026-09`)

		const periodTitle = page.locator('[data-testid="calendar-period-title"]')
		await expect(periodTitle).toContainText('September 2026')

		// Click Next month -> October 2026
		const nextBtn = page.locator('[data-testid="calendar-next-button"]')
		await nextBtn.click()
		await expect(page).toHaveURL(/#calendar\/2026-10/)
		await expect(periodTitle).toContainText('October 2026')

		// Click Previous month -> September 2026
		const prevBtn = page.locator('[data-testid="calendar-prev-button"]')
		await prevBtn.click()
		await expect(page).toHaveURL(/#calendar\/2026-09/)
		await expect(periodTitle).toContainText('September 2026')
	})

	test('renders multi-day milestone timeline bars and navigates on click', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#calendar/2026-09`)

		// Verify Milestone timeline bar is rendered
		const milestoneBar = page
			.locator('[data-testid="calendar-milestone-m-launch-v1"]')
			.first()
		await expect(milestoneBar).toBeVisible()
		await expect(milestoneBar).toContainText('Launch Version 1.0')

		// Click milestone bar -> should navigate to Kanban board filtered to milestone
		await milestoneBar.click()
		await expect(page).toHaveURL(/#milestone\/m-launch-v1/)
	})

	test('renders task chips with opacity differentiation, status dots, and modal anchor URLs', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#calendar/2026-09`)

		// 1. Targeted Active Task on 2026-09-15
		const targetedChip = page.locator(
			'[data-testid="calendar-task-task-active-targeted"]',
		)
		await expect(targetedChip).toBeVisible()
		await expect(targetedChip).toHaveClass(/opacity-100/)

		// 2. Untargeted Active Task on changed_at 2026-09-10
		const untargetedChip = page.locator(
			'[data-testid="calendar-task-task-active-untargeted"]',
		)
		await expect(untargetedChip).toBeVisible()
		await expect(untargetedChip).toHaveClass(/opacity-75/)

		// 3. Completed Task on changed_at 2026-09-22 with line-through
		const doneChip = page.locator('[data-testid="calendar-task-task-completed"]')
		await expect(doneChip).toBeVisible()
		await expect(doneChip).toHaveClass(/line-through/)

		// 4. Click task chip -> opens TaskDetailModal with calendar anchor URL
		await targetedChip.click()
		await expect(page).toHaveURL(/#calendar\/2026-09\/task\/task-active-targeted/)

		const detailModal = page.locator('[data-testid="task-detail-modal"]')
		await expect(detailModal).toBeVisible()

		// 5. Close modal -> returns to calendar view
		const closeBtn = page.locator('[data-testid="modal-close-button"]')
		await closeBtn.click()
		await expect(detailModal).not.toBeVisible()
		await expect(page).toHaveURL(/#calendar\/2026-09/)
	})

	test('opens task creation modal with pre-filled target date when clicking add button on day cell', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#calendar/2026-09`)

		// Hover and click the "+" button for 2026-09-15
		const addDayBtn = page.locator(
			'[data-testid="calendar-add-task-2026-09-15"]',
		)
		await addDayBtn.click({ force: true })

		// Verify CreateTaskModal opens with target_at set to 2026-09-15
		const createModal = page.locator('[data-testid="create-task-modal"]')
		await expect(createModal).toBeVisible()

		const targetInput = page.locator(
			'[data-testid="task-create-target-at-input"]',
		)
		await expect(targetInput).toHaveValue('2026-09-15T09:00')

		// Close modal
		const cancelBtn = page.locator('[data-testid="create-task-cancel-button"]')
		await cancelBtn.click()
		await expect(createModal).not.toBeVisible()
	})

	test('switches between Month view and Week view with accurate week title', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#calendar/2026-09`)

		// Switch to Week view
		const weekBtn = page.locator('[data-testid="calendar-view-week"]')
		await weekBtn.click()

		await expect(page).toHaveURL(/#calendar\/2026-W\d+/)
		const periodTitle = page.locator('[data-testid="calendar-period-title"]')
		// Title format should be e.g. "36 · Sep 2026" without 'W' prefix
		await expect(periodTitle).toContainText(/\d{1,2} · \w+ \d{4}/)
		await expect(periodTitle).not.toContainText(/W\d+/)

		// Switch back to Month view
		const monthBtn = page.locator('[data-testid="calendar-view-month"]')
		await monthBtn.click()

		await expect(page).toHaveURL(/#calendar\/\d{4}-\d{2}/)
	})

	test('clicking +N more navigates to weekly view showing all tasks with bigger font', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#calendar/2026-09`)

		// Verify "+1 more" button is visible on 2026-09-15
		const moreBtn = page.locator(
			'[data-testid="calendar-more-tasks-2026-09-15"]',
		)
		await expect(moreBtn).toBeVisible()
		await expect(moreBtn).toContainText('+1')

		// Click "+1 more" -> navigates to week view for week 38
		await moreBtn.click()
		await expect(page).toHaveURL(/#calendar\/2026-W38/)

		const periodTitle = page.locator('[data-testid="calendar-period-title"]')
		await expect(periodTitle).toContainText('38 · Sep 2026')

		// In week view, all 4 tasks for Sep 15 are visible
		await expect(
			page.locator('[data-testid="calendar-task-task-active-targeted"]'),
		).toBeVisible()
		await expect(
			page.locator('[data-testid="calendar-task-task-overflow-1"]'),
		).toBeVisible()
		await expect(
			page.locator('[data-testid="calendar-task-task-overflow-2"]'),
		).toBeVisible()
		await expect(
			page.locator('[data-testid="calendar-task-task-overflow-3"]'),
		).toBeVisible()

		// Verify bigger font styling in weekly view
		const taskChip = page.locator(
			'[data-testid="calendar-task-task-active-targeted"]',
		)
		await expect(taskChip).toHaveClass(/text-xs|text-sm/)
	})

	test('filters calendar items using the state filter in FilterBar', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#calendar/2026-09`)

		// Filter by 'ready' state
		const readyFilterBtn = page.locator('[data-testid="filter-state-ready"]')
		await expect(readyFilterBtn).toBeVisible()
		await readyFilterBtn.click()

		// Ready task should be visible, in_progress and done tasks should be filtered out
		await expect(
			page.locator('[data-testid="calendar-task-task-active-untargeted"]'),
		).toBeVisible()
		await expect(
			page.locator('[data-testid="calendar-task-task-active-targeted"]'),
		).not.toBeVisible()
		await expect(
			page.locator('[data-testid="calendar-task-task-completed"]'),
		).not.toBeVisible()

		// Toggle 'ready' filter off -> all tasks return
		await readyFilterBtn.click()
		await expect(
			page.locator('[data-testid="calendar-task-task-active-targeted"]'),
		).toBeVisible()
	})

	test('renders mobile weekly view with full-width snap-x scrollable day columns and quick nav', async ({
		page,
	}) => {
		// Set mobile viewport (iPhone dimensions)
		await page.setViewportSize({ width: 375, height: 667 })
		await page.goto(`${server.url}/#calendar/2026-W38`)

		// Verify week-day quick navigation bar is visible on mobile
		const quickNav = page.locator('[data-testid="week-day-quick-nav"]')
		await expect(quickNav).toBeVisible()

		// Verify scroll container has snap-x classes
		const weekContainer = page.locator(
			'[data-testid="calendar-week-scroll-container"]',
		)
		await expect(weekContainer).toHaveClass(/snap-x/)
		await expect(weekContainer).toHaveClass(/snap-mandatory/)

		// Day column for 2026-09-15 is in DOM with snap-center
		const dayCol = page.locator('[data-testid="calendar-day-2026-09-15"]')
		await expect(dayCol).toBeAttached()
		await expect(dayCol).toHaveClass(/snap-center/)

		// Clicking quick-nav button for 2026-09-15 scrolls to day
		const dayPill = page.locator('[data-testid="quick-nav-day-2026-09-15"]')
		await expect(dayPill).toBeVisible()
		await dayPill.click()

		// Tasks are visible in mobile weekly view
		await expect(
			page.locator('[data-testid="calendar-task-task-active-targeted"]'),
		).toBeVisible()
	})
})

