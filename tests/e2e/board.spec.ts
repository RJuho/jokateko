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
})
