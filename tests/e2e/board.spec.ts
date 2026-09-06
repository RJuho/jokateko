import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

test.describe('Kanban Board & Real-Time Filtering E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				{
					id: '260901-task-alpha',
					title: 'Alpha Task Documentation',
					status: 'backlog',
					priority: 'high',
					tags: ['docs', 'architecture'],
					summary: 'Write architectural guidelines for alpha release',
				},
				{
					id: '260901-task-beta',
					title: 'Beta Implementation Engine',
					status: 'in_progress',
					priority: 'critical',
					tags: ['backend', 'engine'],
					summary: 'Implement core processing engine in pure Go',
				},
				{
					id: '260901-task-gamma',
					title: 'Gamma Frontend Component',
					status: 'done',
					priority: 'low',
					tags: ['frontend', 'ui'],
					summary: 'Build clean UI view component with Preact',
				},
			],
		})
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('renders board with configured columns and matching cards', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Verify live mode indicator badge
		const liveBadge = page.locator('[data-testid="mode-indicator-live"]')
		await expect(liveBadge).toBeVisible()

		// 2. Verify configured columns exist
		const expectedColumns = [
			'backlog',
			'ready',
			'in_progress',
			'in_review',
			'done',
		]
		for (const col of expectedColumns) {
			const colEl = page.locator(`[data-testid="column-${col}"]`)
			await expect(colEl).toBeVisible()
		}

		// 3. Verify cards render in their respective columns
		const alphaCard = page.locator(
			'[data-testid="column-backlog"] [data-testid="task-card-260901-task-alpha"]',
		)
		await expect(alphaCard).toBeVisible()
		await expect(alphaCard).toContainText('Alpha Task Documentation')

		const betaCard = page.locator(
			'[data-testid="column-in_progress"] [data-testid="task-card-260901-task-beta"]',
		)
		await expect(betaCard).toBeVisible()
		await expect(betaCard).toContainText('Beta Implementation Engine')

		const gammaCard = page.locator(
			'[data-testid="column-done"] [data-testid="task-card-260901-task-gamma"]',
		)
		await expect(gammaCard).toBeVisible()
		await expect(gammaCard).toContainText('Gamma Frontend Component')
	})

	test('filters tasks in real time by search query', async ({ page }) => {
		await page.goto(server.url)

		const searchInput = page.locator('[data-testid="search-input"]')
		await expect(searchInput).toBeVisible()

		// Type "engine" -> only betaCard should match
		await searchInput.fill('engine')

		await expect(
			page.locator('[data-testid="task-card-260901-task-beta"]'),
		).toBeVisible()
		await expect(
			page.locator('[data-testid="task-card-260901-task-alpha"]'),
		).not.toBeVisible()
		await expect(
			page.locator('[data-testid="task-card-260901-task-gamma"]'),
		).not.toBeVisible()

		// Clear search -> all cards return
		await searchInput.fill('')
		await expect(
			page.locator('[data-testid="task-card-260901-task-alpha"]'),
		).toBeVisible()
		await expect(
			page.locator('[data-testid="task-card-260901-task-beta"]'),
		).toBeVisible()
		await expect(
			page.locator('[data-testid="task-card-260901-task-gamma"]'),
		).toBeVisible()
	})

	test('navigates between views seamlessly via tabs', async ({ page }) => {
		await page.goto(server.url)

		// Click Strategies tab
		const strategiesTab = page.locator('[data-testid="nav-tab-strategies"]')
		await strategiesTab.click()
		await expect(page).toHaveURL(/#strategies/)
		await expect(
			page.getByRole('heading', { name: 'Architectural Strategies' }),
		).toBeVisible()

		// Click Glossary tab
		const glossaryTab = page.locator('[data-testid="nav-tab-glossary"]')
		await glossaryTab.click()
		await expect(page).toHaveURL(/#glossary/)
		await expect(
			page.getByRole('heading', { name: 'Project Glossary' }),
		).toBeVisible()

		// Click Board tab
		const boardTab = page.locator('[data-testid="nav-tab-board"]')
		await boardTab.click()
		await expect(page).toHaveURL(/#board/)
		await expect(
			page.locator('[data-testid="column-in_progress"]'),
		).toBeVisible()
	})

	test('opens column detail modal when clicking column header pill', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Click column header pill for "in_progress"
		const pill = page.locator('[data-testid="column-header-pill-in_progress"]')
		await expect(pill).toBeVisible()
		await pill.click()

		// 2. Verify modal opened
		const modal = page.locator('[data-testid="column-detail-modal"]')
		await expect(modal).toBeVisible()
		await expect(
			page.locator('[data-testid="column-modal-badge"]'),
		).toHaveText('In Progress')
		await expect(
			page.locator('[data-testid="column-modal-id"]'),
		).toHaveText('status: in_progress')

		// 3. Verify close button closes modal
		const closeBtn = page.locator('[data-testid="column-modal-close-btn"]')
		await closeBtn.click()
		await expect(modal).not.toBeVisible()

		// 4. Test keyboard Esc dismiss
		await pill.click()
		await expect(modal).toBeVisible()
		await page.keyboard.press('Escape')
		await expect(modal).not.toBeVisible()
	})
})

test.describe('Column Detail Modal with Configured Prompts and Handled By', () => {
	let customServer

	test.beforeAll(async () => {
		customServer = await startTestServer({
			customConfig: `
version = "0"
[project]
name = "Prompt Project"

[board]
editable_states = ["backlog"]
creatable_states = ["backlog"]
default_create_state = "backlog"

[[board.columns]]
id = "backlog"
name = "Backlog"
color = "#94a3b8"
handled_by = "human"
instructions = "Triage, scope verification, and spec-first definition."

[[board.columns]]
id = "in_progress"
name = "In Progress"
color = "#f59e0b"
handled_by = "agent:coder"
instructions = "Implement pure Go code with unit tests and zero CGO."

[[board.columns]]
id = "in_review"
name = "In Review"
color = "#a855f7"
handled_by = "agent:reviewer"
instructions = "Deterministic validation and Playwright E2E testing."

[[board.columns]]
id = "done"
name = "Done"
color = "#10b981"
`,
		})
	})

	test.afterAll(async () => {
		if (customServer) {
			await customServer.stop()
		}
	})

	test('displays configured handled_by and prompt instructions in modal', async ({
		page,
	}) => {
		await page.goto(customServer.url)

		// Click "in_progress" column pill
		const inProgressPill = page.locator(
			'[data-testid="column-header-pill-in_progress"]',
		)
		await inProgressPill.click()

		const modal = page.locator('[data-testid="column-detail-modal"]')
		await expect(modal).toBeVisible()
		await expect(
			page.locator('[data-testid="column-modal-handled-by"]'),
		).toContainText('agent:coder')
		await expect(
			page.locator('[data-testid="column-modal-instructions"]'),
		).toContainText(
			'Implement pure Go code with unit tests and zero CGO.',
		)

		// Close modal
		await page.locator('[data-testid="column-modal-close"]').click()
		await expect(modal).not.toBeVisible()

		// Click "in_review" column pill
		const inReviewPill = page.locator(
			'[data-testid="column-header-pill-in_review"]',
		)
		await inReviewPill.click()
		await expect(modal).toBeVisible()
		await expect(
			page.locator('[data-testid="column-modal-handled-by"]'),
		).toContainText('agent:reviewer')
		await expect(
			page.locator('[data-testid="column-modal-instructions"]'),
		).toContainText(
			'Deterministic validation and Playwright E2E testing.',
		)
	})
})

