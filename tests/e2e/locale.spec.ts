import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

test.describe('Project locale', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customConfig: `
version = "0"
[project]
name = "Locale Project"
locale = "fi-FI"

[translations]
task_edit = "Muokkaa tehtävää"
`,
			customTasks: [
				{
					id: 'task-fi-target',
					title: 'Finnish Target Task',
					status: 'backlog',
					priority: 'medium',
					summary: 'Has a target date',
					target_at: '2026-09-15T12:00:00Z',
				},
			],
		})
	})

	test.afterAll(async () => {
		await server?.stop()
	})

	test('overrides the en-US browser locale for calendar names and <html lang>', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#calendar/2026-09`)

		await expect(page.locator('html')).toHaveAttribute('lang', 'fi-FI')
		await expect(
			page.locator('[data-testid="calendar-period-title"]'),
		).toContainText('syyskuu 2026')

		const view = page.locator('[data-testid="calendar-view"]')
		await expect(view.getByText('ma', { exact: true }).first()).toBeVisible()
		await expect(view.getByText('su', { exact: true }).first()).toBeVisible()
	})

	test('formats task detail timestamps in the project locale', async ({
		page,
	}) => {
		await page.goto(`${server.url}/#task/task-fi-target`)

		await expect(page.locator('[data-testid="task-target-at"]')).toContainText(
			'15.9.2026',
		)
	})

	test('applies config.toml translations in the live UI', async ({ page }) => {
		await page.goto(`${server.url}/#task/task-fi-target`)

		await expect(page.locator('[data-testid="edit-task-button"]')).toHaveText(
			'Muokkaa tehtävää',
		)
	})
})
