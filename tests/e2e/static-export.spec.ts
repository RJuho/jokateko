import { expect, test } from '@playwright/test'
import { spawn } from 'node:child_process'
import { existsSync, mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

test.describe('Static Export & Offline Execution E2E', () => {
	let dir
	let exportPath

	test.beforeAll(async () => {
		dir = mkdtempSync(join(tmpdir(), 'jokateko-static-'))
		const binaryPath = '/workspaces/jokateko/bin/jokateko'

		// 1. Initialize workspace
		await new Promise((resolve, reject) => {
			const initProc = spawn(binaryPath, ['init', '-dir', dir])
			initProc.on('exit', (code) => {
				if (code === 0) resolve()
				else reject(new Error(`jokateko init failed: ${code}`))
			})
		})

		// 2. Export standalone HTML
		exportPath = join(dir, 'standalone.html')
		await new Promise((resolve, reject) => {
			const buildProc = spawn(binaryPath, [
				'build',
				'-dir',
				dir,
				'-out',
				exportPath,
			])
			buildProc.on('exit', (code) => {
				if (code === 0) resolve()
				else reject(new Error(`jokateko build failed: ${code}`))
			})
		})

		expect(existsSync(exportPath)).toBe(true)
	})

	test.afterAll(async () => {
		if (dir) {
			try {
				rmSync(dir, { recursive: true, force: true })
			} catch {
				// ignore
			}
		}
	})

	test('opens standalone HTML via file:// protocol in static mode', async ({
		page,
	}) => {
		// Track network requests to ensure zero external/server requests
		const networkRequests = []
		page.on('request', (req) => {
			if (!req.url().startsWith('file://') && !req.url().startsWith('data:')) {
				networkRequests.push(req.url())
			}
		})

		await page.goto(`file://${exportPath}`)

		// 1. Verify zero external network requests
		expect(networkRequests).toHaveLength(0)

		// 2. Verify static mode indicator badge is visible
		const staticBadge = page.locator(
			'[data-testid="mode-indicator-static"]',
		)
		await expect(staticBadge).toBeVisible()

		// 3. Verify board renders configured columns
		const readyCol = page.locator('[data-testid="column-ready"]')
		await expect(readyCol).toBeVisible()

		// 4. Verify initial task card is displayed
		const taskCard = page.locator(
			'[data-testid^="task-card-"][data-testid$="-initial-setup"]',
		)
		await expect(taskCard).toBeVisible()
		await expect(taskCard).toContainText('Initial Project Setup')

		// 5. Verify editing controls (add buttons) are hidden in static mode
		const addBtn = page.locator('[data-testid="add-task-backlog"]')
		await expect(addBtn).not.toBeVisible()
	})

	test('supports offline search and task detail modal exploration', async ({
		page,
	}) => {
		await page.goto(`file://${exportPath}`)

		// 1. Test search filter offline
		const searchInput = page.locator('[data-testid="search-input"]')
		await expect(searchInput).toBeVisible()
		await searchInput.fill('non-matching-query-xyz')

		const taskCard = page.locator(
			'[data-testid^="task-card-"][data-testid$="-initial-setup"]',
		)
		await expect(taskCard).not.toBeVisible()

		await searchInput.fill('Setup')
		await expect(taskCard).toBeVisible()

		// 2. Click task card to open detail modal offline
		await taskCard.click()
		const modal = page.locator('[data-testid="task-detail-modal"]')
		await expect(modal).toBeVisible()
		await expect(modal).toContainText('Initial Project Setup')
		await expect(modal).toContainText('Acceptance Criteria')

		// 3. Verify Edit & Delete mutation actions are hidden in static mode
		const editBtn = page.locator('[data-testid="edit-task-button"]')
		await expect(editBtn).not.toBeVisible()

		// 4. Verify typography prose styling and markdown elements in task modal
		const proseContainer = modal.locator('.prose')
		await expect(proseContainer).toBeVisible()
		await expect(proseContainer.locator('h2')).toContainText('Acceptance Criteria')
		await expect(proseContainer.locator('input[type="checkbox"]')).toHaveCount(2)
	})

	test('renders markdown bodies with typography in strategies and glossary views', async ({
		page,
	}) => {
		await page.goto(`file://${exportPath}`)

		// 1. Strategies View
		await page.locator('[data-testid="nav-tab-strategies"]').click()
		const strategyCard = page.locator(
			'[data-testid="strategy-card-architecture"]',
		)
		await expect(strategyCard).toBeVisible()
		await strategyCard.click()
		const stratProse = strategyCard.locator('.prose')
		await expect(stratProse).toBeVisible()
		await expect(stratProse.locator('p')).not.toHaveCount(0)

		// 2. Glossary View
		await page.locator('[data-testid="nav-tab-glossary"]').click()
		const glossaryCard = page.locator(
			'[data-testid="glossary-card-tasks-as-code"]',
		)
		await expect(glossaryCard).toBeVisible()
		await glossaryCard.click()
		const glossProse = glossaryCard.locator('.prose')
		await expect(glossProse).toBeVisible()
		await expect(glossProse.locator('p')).not.toHaveCount(0)
	})
})
