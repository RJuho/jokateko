import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

test.describe('Modal state isolation & keyboard handling', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				{
					id: 'modal-task-alpha',
					title: 'Alpha Modal Task',
					status: 'backlog',
					priority: 'high',
					tags: [],
					summary: 'Alpha summary',
				},
				{
					id: 'modal-task-beta',
					title: 'Beta Modal Task',
					status: 'backlog',
					priority: 'low',
					tags: [],
					summary: 'Beta summary',
				},
			],
			customStrategies: [
				{
					id: 'modal-strategy',
					title: 'Modal Strategy',
					tier: 1,
					summary: 'Strategy summary',
				},
			],
		})
	})

	test.afterAll(async () => {
		await server?.stop()
	})

	test('edit modal shows values of the task being edited, not a previous one', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#task/modal-task-alpha`)
		await page.locator('[data-testid="edit-task-button"]').click()
		const editModal = page.locator('[data-testid="task-edit-modal"]')
		await expect(editModal).toBeVisible()
		await expect(
			page.locator('[data-testid="task-edit-title-input"]'),
		).toHaveValue('Alpha Modal Task')
		await page.keyboard.press('Escape')
		await expect(editModal).not.toBeVisible()

		await page.goto(`${server.url}/#task/modal-task-beta`)
		await page.locator('[data-testid="edit-task-button"]').click()
		await expect(editModal).toBeVisible()
		await expect(
			page.locator('[data-testid="task-edit-title-input"]'),
		).toHaveValue('Beta Modal Task')
		await expect(
			page.locator('[data-testid="task-edit-summary-input"]'),
		).toHaveValue('Beta summary')
	})

	test('create modal starts empty on every open', async ({ page }) => {
		await page.goto(server.url)
		await page.locator('[data-testid="add-task-backlog"]').click()
		const titleInput = page.locator('[data-testid="task-create-title-input"]')
		await titleInput.fill('Abandoned draft')
		await page.locator('[data-testid="create-task-cancel-button"]').click()
		await expect(
			page.locator('[data-testid="create-task-modal"]'),
		).not.toBeVisible()

		await page.locator('[data-testid="add-task-backlog"]').click()
		await expect(titleInput).toHaveValue('')
	})

	test('Escape on strategies page does not navigate to the board', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#strategy/modal-strategy`)
		await expect(
			page.locator('[data-testid="strategy-card-modal-strategy"]'),
		).toBeVisible()
		await page.locator('body').click({ position: { x: 5, y: 300 } })
		await page.keyboard.press('Escape')
		await page.waitForTimeout(200)
		expect(new URL(page.url()).hash).toBe('#strategy/modal-strategy')
		await expect(page.locator('[data-testid="nav-tab-strategies"]').first()).toHaveAttribute(
			'aria-current',
			'page',
		)
	})
})
