import { expect, test } from '@playwright/test'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { startTestServer } from './helpers/test-server'

test.describe('Kanban Drag and Drop E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				{
					id: '260901-dnd-task-alpha',
					title: 'Drag and Drop Test Alpha',
					status: 'backlog',
					priority: 'high',
					tags: ['dnd', 'ui'],
					summary: 'Test moving card across kanban columns',
				},
			],
		})
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('drags a task card between columns, updating Web UI and disk frontmatter without 405 error', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Verify card starts in Backlog column
		const initialCard = page.locator(
			'[data-testid="column-backlog"] [data-testid="task-card-260901-dnd-task-alpha"]',
		)
		await expect(initialCard).toBeVisible()

		// 2. Drag to "in_progress" column and wait for PUT /status response
		const inProgressCol = page.locator('[data-testid="column-in_progress"]')
		const putStatusPromise = page.waitForResponse(
			(res) =>
				res.url().includes('/api/tasks/260901-dnd-task-alpha/status')
				&& res.request().method() === 'PUT',
		)

		await initialCard.dragTo(inProgressCol)

		// 3. Verify server returned HTTP 200 OK (not 405 Method Not Allowed)
		const statusRes = await putStatusPromise
		expect(statusRes.status()).toBe(200)

		// 4. Verify card now exists in "in_progress" column and removed from "backlog"
		const inProgressCard = page.locator(
			'[data-testid="column-in_progress"] [data-testid="task-card-260901-dnd-task-alpha"]',
		)
		await expect(inProgressCard).toBeVisible()
		await expect(
			page.locator(
				'[data-testid="column-backlog"] [data-testid="task-card-260901-dnd-task-alpha"]',
			),
		).toHaveCount(0)

		// 5. Verify the markdown file on disk reflects status = "in_progress"
		const taskPath = join(
			server.dir,
			'.jokateko',
			'tasks',
			'260901-dnd-task-alpha.md',
		)
		const fileContent = readFileSync(taskPath, 'utf8')
		expect(fileContent).toContain("status = 'in_progress'")

		// 6. Drag from "in_progress" to "done" and wait for PUT /status response
		const doneCol = page.locator('[data-testid="column-done"]')
		await doneCol.scrollIntoViewIfNeeded()

		const putDonePromise = page.waitForResponse(
			(res) =>
				res.url().includes('/api/tasks/260901-dnd-task-alpha/status')
				&& res.request().method() === 'PUT',
		)

		await inProgressCard.dragTo(doneCol)

		const doneRes = await putDonePromise
		expect(doneRes.status()).toBe(200)

		const doneCard = page.locator(
			'[data-testid="column-done"] [data-testid="task-card-260901-dnd-task-alpha"]',
		)
		await expect(doneCard).toBeVisible()
		await expect(
			page.locator(
				'[data-testid="column-in_progress"] [data-testid="task-card-260901-dnd-task-alpha"]',
			),
		).toHaveCount(0)

		// 7. Verify disk file updated to status = "done"
		const finalContent = readFileSync(taskPath, 'utf8')
		expect(finalContent).toContain("status = 'done'")
	})
})
