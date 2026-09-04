import { expect, test } from '@playwright/test'
import { rmSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { startTestServer } from './helpers/test-server'

test.describe('Live Reload & SSE Broadcasting E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				{
					id: '260901-sse-task-alpha',
					title: 'Initial SSE Task Title',
					status: 'backlog',
					priority: 'medium',
					tags: ['sse', 'live'],
					summary: 'Testing real-time update over SSE',
				},
			],
		})
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('updates task card in real-time when file on disk is modified', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Verify card appears in Backlog initially
		const backlogCard = page.locator(
			'[data-testid="column-backlog"] [data-testid="task-card-260901-sse-task-alpha"]',
		)
		await expect(backlogCard).toBeVisible()
		await expect(backlogCard).toContainText('Initial SSE Task Title')

		// 2. Modify task on disk (change title and move to 'done' column)
		const taskPath = join(
			server.dir,
			'.jokateko',
			'tasks',
			'260901-sse-task-alpha.md',
		)
		const updatedContent = `+++
id = "260901-sse-task-alpha"
title = "Reloaded Title Without Refresh"
status = "done"
priority = "critical"
tags = ["sse", "live", "updated"]
summary = "Updated over disk file modification"
+++
## Acceptance Criteria
- [x] Live reload works seamlessly
`
		writeFileSync(taskPath, updatedContent, 'utf8')

		// 3. Without page reload, assert that card moves to Done column and has updated title
		const doneCard = page.locator(
			'[data-testid="column-done"] [data-testid="task-card-260901-sse-task-alpha"]',
		)
		await expect(doneCard).toBeVisible({ timeout: 5000 })
		await expect(doneCard).toContainText('Reloaded Title Without Refresh')
		await expect(
			page.locator(
				'[data-testid="column-backlog"] [data-testid="task-card-260901-sse-task-alpha"]',
			),
		).toHaveCount(0)
	})

	test('renders new card in real-time when new markdown file is created on disk', async ({
		page,
	}) => {
		await page.goto(server.url)

		const newTaskPath = join(
			server.dir,
			'.jokateko',
			'tasks',
			'260901-sse-task-beta.md',
		)
		const newContent = `+++
id = "260901-sse-task-beta"
title = "Dynamically Added Task Over Disk"
status = "ready"
priority = "high"
tags = ["disk-event"]
summary = "New task created directly on disk"
+++
## Acceptance Criteria
- [ ] Render automatically
`
		writeFileSync(newTaskPath, newContent, 'utf8')

		// Without page reload, assert that new card appears in Ready column
		const readyCard = page.locator(
			'[data-testid="column-ready"] [data-testid="task-card-260901-sse-task-beta"]',
		)
		await expect(readyCard).toBeVisible({ timeout: 5000 })
		await expect(readyCard).toContainText('Dynamically Added Task Over Disk')

		// Delete the task on disk and verify it disappears from Web UI in real-time
		rmSync(newTaskPath)
		await expect(readyCard).toHaveCount(0, { timeout: 5000 })
	})
})
