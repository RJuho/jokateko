import { expect, test } from '@playwright/test'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { startTestServer } from './helpers/test-server'

test.describe('Task Creation & Mutation E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer()
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('creates a new task via modal and persists file to disk', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Click "+" button on Backlog column
		const addBtn = page.locator('[data-testid="add-task-backlog"]')
		await expect(addBtn).toBeVisible()
		await addBtn.click()

		// 2. Fill in Create Task modal
		const modal = page.locator('[data-testid="create-task-modal"]')
		await expect(modal).toBeVisible()

		const titleInput = page.locator('[data-testid="task-create-title-input"]')
		await titleInput.fill('Playwright Created Task')

		const summaryInput = page.locator(
			'[data-testid="task-create-summary-input"]',
		)
		await summaryInput.fill('Created during Playwright E2E test')

		const submitBtn = page.locator(
			'[data-testid="create-task-submit-button"]',
		)
		await submitBtn.click()

		// 3. Verify modal closes
		await expect(modal).not.toBeVisible()

		// 4. Verify new task card renders on the board
		const newCard = page.locator('text=Playwright Created Task').first()
		await expect(newCard).toBeVisible()

		// 5. Verify the markdown file was written to disk
		const tasksDir = join(server.dir, '.jokateko', 'tasks')
		const files = readdirSync(tasksDir)
		const createdFile = files.find((f) => f.includes('playwright-created-task'))
		expect(createdFile).toBeDefined()

		const fileContent = readFileSync(join(tasksDir, createdFile), 'utf8')
		expect(fileContent).toContain('Playwright Created Task')
		expect(fileContent).toContain("status = 'backlog'")
	})

	test('edits an existing task via modal and updates file on disk', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Click on the created task card
		const card = page.locator('text=Playwright Created Task').first()
		await expect(card).toBeVisible()
		await card.click()

		// 2. Verify detail modal opens
		const detailModal = page.locator('[data-testid="task-detail-modal"]')
		await expect(detailModal).toBeVisible()

		// 3. Click Edit Task button
		const editBtn = page.locator('[data-testid="edit-task-button"]')
		await expect(editBtn).toBeVisible()
		await editBtn.click()

		// 4. Update title in Edit modal
		const editModal = page.locator('[data-testid="task-edit-modal"]')
		await expect(editModal).toBeVisible()

		const titleInput = page.locator('[data-testid="task-edit-title-input"]')
		await titleInput.fill('Playwright Mutated Task Title')

		const saveBtn = page.locator('[data-testid="save-task-button"]')
		await saveBtn.click()

		// 5. Verify modal closes and updated card renders
		await expect(editModal).not.toBeVisible()
		const updatedCard = page.locator('text=Playwright Mutated Task Title').first()
		await expect(updatedCard).toBeVisible()

		// 6. Verify file on disk is updated
		const tasksDir = join(server.dir, '.jokateko', 'tasks')
		const files = readdirSync(tasksDir)
		const mutatedFile = files.find((f) => f.includes('playwright-created-task'))
		expect(mutatedFile).toBeDefined()

		const fileContent = readFileSync(join(tasksDir, mutatedFile), 'utf8')
		expect(fileContent).toContain('Playwright Mutated Task Title')
	})
})
