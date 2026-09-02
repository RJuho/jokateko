import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

test.describe('Markdown Rendering & @tailwindcss/typography E2E', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				{
					id: '260902-dynamic-mcp-proxy-hot-failover-auto-reco',
					title: 'Dynamic MCP Proxy Hot-Failover & Auto-Reconnect Engine',
					status: 'done',
					priority: 'critical',
					tags: ['backend', 'proxy', 'mcp'],
					summary: 'Hot-failover between remote proxy and in-process standalone MCP engine',
					body: `# Dynamic MCP Proxy Hot-Failover

## Acceptance Criteria
- [x] Implement dynamic MCP transport router
- [ ] Fallback test criterion

## Completion Summary
- **Completed At:** 2026-09-02T17:04:31Z

### What Was Done
Engineered dynamic MCP transport router with immediate hot-failover.`,
				},
			],
		})
	})

	test.afterAll(async () => {
		if (server) {
			await server.stop()
		}
	})

	test('renders task modal markdown with bold text and @tailwindcss/typography prose class', async ({
		page,
	}) => {
		await page.goto(server.url)

		// 1. Locate and click target task card
		const card = page.locator(
			'[data-testid="task-card-260902-dynamic-mcp-proxy-hot-failover-auto-reco"]',
		)
		await expect(card).toBeVisible()
		await card.click()

		// 2. Locate task detail modal
		const modal = page.locator('[data-testid="task-detail-modal"]')
		await expect(modal).toBeVisible()

		// 3. Verify .prose class exists for typography styling
		const prose = modal.locator('.prose')
		await expect(prose).toBeVisible()

		// 4. Verify bold text "Completed At:" is rendered as <strong> tag and NOT as raw "**Completed At:**"
		const strongEl = prose.locator('strong', { hasText: 'Completed At:' })
		await expect(strongEl).toBeVisible()
		await expect(strongEl).toHaveText('Completed At:')

		// Verify raw markdown asterisks are not shown as plain text
		const proseText = await prose.innerText()
		expect(proseText).not.toContain('**Completed At:**')

		// 5. Verify computed font-weight of <strong> tag is bold (>= 600 or '700')
		const fontWeight = await strongEl.evaluate((el) => {
			return window.getComputedStyle(el).fontWeight
		})
		expect(['bold', '600', '700', '800', '900']).toContain(fontWeight)

		// 6. Verify interactive checkboxes are rendered properly inside prose
		const checkboxes = prose.locator('input[type="checkbox"]')
		await expect(checkboxes).toHaveCount(2)
		await expect(checkboxes.first()).toBeChecked()
		await expect(checkboxes.nth(1)).not.toBeChecked()
	})

	test('renders strategies markdown with .prose typography', async ({
		page,
	}) => {
		await page.goto(server.url)

		// Navigate to strategies view
		await page.locator('[data-testid="nav-tab-strategies"]').click()
		const strategyCard = page.locator(
			'[data-testid="strategy-card-architecture"]',
		)
		await expect(strategyCard).toBeVisible()

		// Expand strategy
		await strategyCard.click()

		// Verify prose container is present and styled
		const prose = strategyCard.locator('.prose')
		await expect(prose).toBeVisible()
		await expect(prose.locator('p, h1, h2, h3')).not.toHaveCount(0)
	})

	test('renders glossary markdown with .prose typography', async ({ page }) => {
		await page.goto(server.url)

		// Navigate to glossary view
		await page.locator('[data-testid="nav-tab-glossary"]').click()
		const glossaryCard = page.locator(
			'[data-testid="glossary-card-tasks-as-code"]',
		)
		await expect(glossaryCard).toBeVisible()

		// Expand glossary term
		await glossaryCard.click()

		// Verify prose container is present and styled
		const prose = glossaryCard.locator('.prose')
		await expect(prose).toBeVisible()
		await expect(prose.locator('p, h1, h2, h3')).not.toHaveCount(0)
	})
})
