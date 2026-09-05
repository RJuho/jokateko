import { expect, test } from '@playwright/test'
import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { startTestServer } from './helpers/test-server'

test.describe('Timestamps and Milestone Timeframe E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer()
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('creates task with target_at, verifies frontmatter and card badge', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Open Create Task modal
		const addBtn = page.locator('[data-testid="add-task-backlog"]')
		await expect(addBtn).toBeVisible()
		await addBtn.click()

		const modal = page.locator('[data-testid="create-task-modal"]')
		await expect(modal).toBeVisible()

		// 2. Fill in title, summary, and target date
		const titleInput = page.locator('[data-testid="task-create-title-input"]')
		await titleInput.fill('Task With Target Timestamp')

		const summaryInput = page.locator(
			'[data-testid="task-create-summary-input"]',
		)
		await summaryInput.fill('Verifying target date handling and badge')

		const targetInput = page.locator(
			'[data-testid="task-create-target-at-input"]',
		)
		await expect(targetInput).toBeVisible()
		await targetInput.fill('2026-12-31T15:30')

		const submitBtn = page.locator(
			'[data-testid="create-task-submit-button"]',
		)
		await submitBtn.click()

		// 3. Verify modal closes
		await expect(modal).not.toBeVisible()

		// 4. Verify task card renders with target date badge
		const newCard = page.locator('text=Task With Target Timestamp').first()
		await expect(newCard).toBeVisible()

		const targetBadge = page.locator('[data-testid="target-date-badge"]').first()
		await expect(targetBadge).toBeVisible()
		await expect(targetBadge).toContainText('Dec 31')

		// 5. Verify file on disk contains created_at, changed_at, target_at
		const tasksDir = join(server.dir, '.jokateko', 'tasks')
		const files = readdirSync(tasksDir)
		const createdFile = files.find((f) => f.includes('task-with-target-timestamp'))
		expect(createdFile).toBeDefined()

		const fileContent = readFileSync(join(tasksDir, createdFile), 'utf8')
		expect(fileContent).toContain('created_at =')
		expect(fileContent).toContain('changed_at =')
		expect(fileContent).toContain("target_at = '2026-12-31T15:30:00Z'")
	})

	test('opens task detail modal and verifies timestamp display', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Click on the task card
		const card = page.locator('text=Task With Target Timestamp').first()
		await expect(card).toBeVisible()
		await card.click()

		// 2. Verify detail modal opens and shows timestamps
		const detailModal = page.locator('[data-testid="task-detail-modal"]')
		await expect(detailModal).toBeVisible()

		const timestampsRow = page.locator('[data-testid="task-timestamps"]')
		await expect(timestampsRow).toBeVisible()

		const createdAtEl = page.locator('[data-testid="task-created-at"]')
		await expect(createdAtEl).toBeVisible()
		await expect(createdAtEl).toContainText('Created:')

		const changedAtEl = page.locator('[data-testid="task-changed-at"]')
		await expect(changedAtEl).toBeVisible()
		await expect(changedAtEl).toContainText('Updated:')

		const targetAtEl = page.locator('[data-testid="task-target-at"]')
		await expect(targetAtEl).toBeVisible()
		await expect(targetAtEl).toContainText('Target:')

		// 3. Edit task target date via Edit Modal
		const editBtn = page.locator('[data-testid="edit-task-button"]')
		await expect(editBtn).toBeVisible()
		await editBtn.click()

		const editModal = page.locator('[data-testid="task-edit-modal"]')
		await expect(editModal).toBeVisible()

		const editTargetInput = page.locator(
			'[data-testid="task-edit-target-at-input"]',
		)
		await expect(editTargetInput).toBeVisible()
		await editTargetInput.fill('2026-11-15T10:00')

		const saveBtn = page.locator('[data-testid="save-task-button"]')
		await saveBtn.click()

		await expect(editModal).not.toBeVisible()

		// 4. Verify detail modal shows updated target
		await expect(targetAtEl).toBeVisible()

		// 5. Verify file on disk updated
		const tasksDir = join(server.dir, '.jokateko', 'tasks')
		const files = readdirSync(tasksDir)
		const createdFile = files.find((f) => f.includes('task-with-target-timestamp'))
		const updatedContent = readFileSync(join(tasksDir, createdFile), 'utf8')
		expect(updatedContent).toContain("target_at = '2026-11-15T10:00:00Z'")
	})

	test('derives milestone target timeframe from tasks with target dates', async ({
		page,
	}) => {
		// 1. Create a milestone on disk
		const milestonesDir = join(server.dir, '.jokateko', 'milestones')
		const msContent = `+++
id = "261001-q4-release"
title = "Q4 Release"
status = "open"
target_date = "2026-12-31"
tags = ["release"]
summary = "Deliver Q4 roadmap."
+++
`
		writeFileSync(join(milestonesDir, '261001-q4-release.md'), msContent, 'utf8')

		// 2. Create two tasks assigned to this milestone with different target dates
		const tasksDir = join(server.dir, '.jokateko', 'tasks')
		const task1 = `+++
id = "260910-task-alpha"
title = "Task Alpha"
status = "ready"
priority = "high"
milestone = "261001-q4-release"
tags = ["frontend"]
summary = "Alpha feature"
target_at = "2026-10-05T00:00:00Z"
created_at = "2026-09-05T08:00:00Z"
changed_at = "2026-09-05T08:00:00Z"
+++
`
		const task2 = `+++
id = "260910-task-beta"
title = "Task Beta"
status = "in_progress"
priority = "medium"
milestone = "261001-q4-release"
tags = ["backend"]
summary = "Beta feature"
target_at = "2026-10-25T00:00:00Z"
created_at = "2026-09-05T08:00:00Z"
changed_at = "2026-09-05T08:00:00Z"
+++
`
		writeFileSync(join(tasksDir, '260910-task-alpha.md'), task1, 'utf8')
		writeFileSync(join(tasksDir, '260910-task-beta.md'), task2, 'utf8')

		// 3. Reload page
		await page.goto(server.url)

		// 4. Verify MilestoneCards displays timeframe on board
		const timeframeEl = page.locator(
			'[data-testid="milestone-timeframe-261001-q4-release"]',
		)
		await expect(timeframeEl).toBeVisible()
		await expect(timeframeEl).toContainText('2026-10-05')
		await expect(timeframeEl).toContainText('2026-10-25')

		// 5. Navigate to Milestones View and verify timeframe
		const milestonesNav = page.locator('button[data-testid="nav-milestones"]')
		if (await milestonesNav.isVisible()) {
			await milestonesNav.click()
			const viewTimeframe = page.locator(
				'[data-testid="milestone-timeframe-261001-q4-release"]',
			)
			await expect(viewTimeframe).toBeVisible()
			await expect(viewTimeframe).toContainText('2026-10-05')
			await expect(viewTimeframe).toContainText('2026-10-25')
		}
	})
})
