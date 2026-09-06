import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

test.describe('Multi-Criteria Task Sorting Web UI E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				// Backlog column tasks
				{
					id: 'task-b-low',
					title: 'Zebra Backlog Low',
					status: 'backlog',
					priority: 'low',
					tags: ['backend'],
					summary: 'Low priority backlog task',
					changed_at: '2026-09-01T10:00:00Z',
					created_at: '2026-09-01T10:00:00Z',
				},
				{
					id: 'task-b-crit',
					title: 'Alpha Backlog Critical',
					status: 'backlog',
					priority: 'critical',
					tags: ['backend'],
					summary: 'Critical priority backlog task',
					changed_at: '2026-09-02T10:00:00Z',
					created_at: '2026-09-02T10:00:00Z',
				},
				{
					id: 'task-b-high-old',
					title: 'Middle Backlog High Old',
					status: 'backlog',
					priority: 'high',
					tags: ['backend'],
					summary: 'High priority backlog older task',
					changed_at: '2026-09-03T10:00:00Z',
					created_at: '2026-09-03T10:00:00Z',
				},
				{
					id: 'task-b-high-new',
					title: 'Beta Backlog High New',
					status: 'backlog',
					priority: 'high',
					tags: ['backend'],
					summary: 'High priority backlog newer task',
					changed_at: '2026-09-05T10:00:00Z',
					created_at: '2026-09-05T10:00:00Z',
				},
				// In Progress column tasks
				{
					id: 'task-p-sooner',
					title: 'Delta Sooner Deadline',
					status: 'in_progress',
					priority: 'high',
					tags: ['frontend'],
					summary: 'Approaching target deadline',
					target_at: '2026-09-10T00:00:00Z',
					changed_at: '2026-09-01T00:00:00Z',
				},
				{
					id: 'task-p-later',
					title: 'Echo Later Deadline',
					status: 'in_progress',
					priority: 'high',
					tags: ['frontend'],
					summary: 'Far target deadline',
					target_at: '2026-09-25T00:00:00Z',
					changed_at: '2026-09-01T00:00:00Z',
				},
				{
					id: 'task-p-notarget',
					title: 'Foxtrot No Deadline',
					status: 'in_progress',
					priority: 'high',
					tags: ['frontend'],
					summary: 'Task without target deadline',
					changed_at: '2026-09-06T00:00:00Z',
				},
				// Done column tasks (verifies default sorting is Recently Changed regardless of priority)
				{
					id: 'task-d-crit-old',
					title: 'Golf Critical Done Old',
					status: 'done',
					priority: 'critical',
					tags: ['backend'],
					summary: 'Critical priority done older task',
					changed_at: '2026-09-01T12:00:00Z',
					created_at: '2026-09-01T12:00:00Z',
				},
				{
					id: 'task-d-low-new',
					title: 'Hotel Low Done New',
					status: 'done',
					priority: 'low',
					tags: ['backend'],
					summary: 'Low priority done newer task',
					changed_at: '2026-09-06T12:00:00Z',
					created_at: '2026-09-06T12:00:00Z',
				},
			],
		})
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('renders default workflow state ordering per column', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Verify Column sort buttons are visible in each column header pill
		const backlogSortBtn = page.locator('[data-testid="column-sort-button-backlog"]')
		const inProgressSortBtn = page.locator(
			'[data-testid="column-sort-button-in_progress"]',
		)
		await expect(backlogSortBtn).toBeVisible()
		await expect(inProgressSortBtn).toBeVisible()

		// 2. Verify Backlog column default order: Critical -> High (newest first) -> Low
		const backlogCards = page
			.locator('[data-testid="column-backlog"]')
			.locator('[data-testid^="task-card-"]')

		await expect(backlogCards).toHaveCount(4)
		await expect(backlogCards.nth(0)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-crit',
		)
		await expect(backlogCards.nth(1)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-new',
		)
		await expect(backlogCards.nth(2)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-old',
		)
		await expect(backlogCards.nth(3)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-low',
		)

		// 3. Verify In Progress column default order: soonest target_at -> later target_at -> no target_at
		const inProgressCards = page
			.locator('[data-testid="column-in_progress"]')
			.locator('[data-testid^="task-card-"]')

		await expect(inProgressCards).toHaveCount(3)
		await expect(inProgressCards.nth(0)).toHaveAttribute(
			'data-testid',
			'task-card-task-p-sooner',
		)
		await expect(inProgressCards.nth(1)).toHaveAttribute(
			'data-testid',
			'task-card-task-p-later',
		)
		await expect(inProgressCards.nth(2)).toHaveAttribute(
			'data-testid',
			'task-card-task-p-notarget',
		)

		// 4. Verify Done column default order: Recently Changed (changed_at desc) regardless of priority
		const doneCards = page
			.locator('[data-testid="column-done"]')
			.locator('[data-testid^="task-card-"]')

		await expect(doneCards).toHaveCount(2)
		// task-d-low-new (Sep 06, low priority) comes before task-d-crit-old (Sep 01, critical priority)
		await expect(doneCards.nth(0)).toHaveAttribute(
			'data-testid',
			'task-card-task-d-low-new',
		)
		await expect(doneCards.nth(1)).toHaveAttribute(
			'data-testid',
			'task-card-task-d-crit-old',
		)
	})

	test('updates card ordering when selecting Alphabetical (Title A-Z) per column', async ({
		page,
	}) => {
		await page.goto(server.url)

		// Open Backlog sort dropdown
		await page.click('[data-testid="column-sort-button-backlog"]')
		const sortMenu = page.locator(
			'[data-testid="column-sort-menu-backlog"]',
		)
		await expect(sortMenu).toBeVisible()

		// Select Alphabetical (Title A-Z) for Backlog
		await page.click('[data-testid="column-sort-option-backlog-title"]')

		// Verify Backlog ordering: Alpha -> Beta -> Middle -> Zebra
		const backlogCards = page
			.locator('[data-testid="column-backlog"]')
			.locator('[data-testid^="task-card-"]')

		await expect(backlogCards.nth(0)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-crit',
		) // Alpha
		await expect(backlogCards.nth(1)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-new',
		) // Beta
		await expect(backlogCards.nth(2)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-old',
		) // Middle
		await expect(backlogCards.nth(3)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-low',
		) // Zebra

		// Verify In Progress ordering was NOT altered by Backlog sorting
		const inProgressCards = page
			.locator('[data-testid="column-in_progress"]')
			.locator('[data-testid^="task-card-"]')
		await expect(inProgressCards.nth(0)).toHaveAttribute(
			'data-testid',
			'task-card-task-p-sooner',
		)
	})

	test('updates card ordering when selecting Recently Changed per column', async ({
		page,
	}) => {
		await page.goto(server.url)

		// Open Backlog sort dropdown
		await page.click('[data-testid="column-sort-button-backlog"]')
		// Select Recently Changed
		await page.click('[data-testid="column-sort-option-backlog-changed_at"]')

		// Verify Backlog ordering by changed_at desc:
		// task-b-high-new (Sep 05) -> task-b-high-old (Sep 03) -> task-b-crit (Sep 02) -> task-b-low (Sep 01)
		const backlogCards = page
			.locator('[data-testid="column-backlog"]')
			.locator('[data-testid^="task-card-"]')

		await expect(backlogCards.nth(0)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-new',
		)
		await expect(backlogCards.nth(1)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-old',
		)
		await expect(backlogCards.nth(2)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-crit',
		)
		await expect(backlogCards.nth(3)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-low',
		)
	})

	test('reset filters restores default workflow sorting', async ({
		page,
	}) => {
		await page.goto(server.url)

		// Switch Backlog to Recently Changed
		await page.click('[data-testid="column-sort-button-backlog"]')
		await page.click('[data-testid="column-sort-option-backlog-changed_at"]')

		// Reset button in FilterBar should now be visible
		const resetBtn = page.locator('button[aria-label="Reset all filters"]')
		await expect(resetBtn).toBeVisible()
		await resetBtn.click()

		// Verify Backlog restored to default: Critical -> High (newest first) -> Low
		const backlogCards = page
			.locator('[data-testid="column-backlog"]')
			.locator('[data-testid^="task-card-"]')

		await expect(backlogCards.nth(0)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-crit',
		)
		await expect(backlogCards.nth(1)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-new',
		)
		await expect(backlogCards.nth(2)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-high-old',
		)
		await expect(backlogCards.nth(3)).toHaveAttribute(
			'data-testid',
			'task-card-task-b-low',
		)
	})
})
