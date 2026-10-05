import { expect, test } from '@playwright/test'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { startTestServer } from './helpers/test-server'

// Attaching a task to a closed milestone needs explicit confirmation: the
// server answers 409 "milestone_archived" and the modal asks before retrying
// with reopen_milestone.
test.describe('Archived milestone confirmation', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customMilestones: [
				{ id: '260101-closed', title: 'Closed Milestone', status: 'closed' },
			],
		})
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	async function fillCreateForm(page, title) {
		await page.goto(server.url)
		await page.locator('[data-testid="add-task-backlog"]').click()
		await expect(
			page.locator('[data-testid="create-task-modal"]'),
		).toBeVisible()
		await page.locator('[data-testid="task-create-title-input"]').fill(title)
		await page
			.locator('[data-testid="task-create-summary-input"]')
			.fill('Attached to a closed milestone')
		await page
			.locator('[data-testid="task-create-milestone-select"]')
			.selectOption('260101-closed')
	}

	function taskFile(slug) {
		const tasksDir = join(server.dir, '.jokateko', 'tasks')
		const file = readdirSync(tasksDir).find((f) => f.includes(slug))
		return file && readFileSync(join(tasksDir, file), 'utf8')
	}

	test('declining the confirmation keeps the modal open with the error', async ({
		page,
	}) => {
		await fillCreateForm(page, 'Declined Reopen Task')

		const dialogs = []
		page.once('dialog', async (dialog) => {
			dialogs.push(dialog.message())
			await dialog.dismiss()
		})
		await page.locator('[data-testid="create-task-submit-button"]').click()

		const modal = page.locator('[data-testid="create-task-modal"]')
		await expect(modal.getByRole('alert')).toContainText('reopen_milestone')
		expect(dialogs).toHaveLength(1)
		expect(dialogs[0]).toContain('Closed Milestone')
		expect(taskFile('declined-reopen-task')).toBeUndefined()
	})

	test('accepting the confirmation creates the task on the milestone', async ({
		page,
	}) => {
		await fillCreateForm(page, 'Accepted Reopen Task')

		page.once('dialog', (dialog) => dialog.accept())
		await page.locator('[data-testid="create-task-submit-button"]').click()

		await expect(
			page.locator('[data-testid="create-task-modal"]'),
		).not.toBeVisible()
		expect(taskFile('accepted-reopen-task')).toContain(
			"milestone = '260101-closed'",
		)
	})
})
