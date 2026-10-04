import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

test.describe('Strategy card interactions', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customStrategies: [
				{
					id: 'strat-card-a',
					title: 'Card A',
					tier: 1,
					summary: 'Summary of card A',
					tags: ['alpha'],
					body: 'Body of card A\n',
				},
				{
					id: 'strat-card-b',
					title: 'Card B',
					tier: 1,
					summary: 'Summary of card B',
					tags: ['beta'],
					body: 'Body of card B\n',
				},
			],
		})
	})

	test.afterAll(async () => {
		if (server) await server.stop()
	})

	test('tags filter without toggling; summary and title toggle; no nested buttons', async ({ page }) => {
		await page.goto(`${server.url}/#strategies`)
		const card = page.locator('[data-testid="strategy-card-strat-card-a"]')
		const toggle = card.getByRole('button', { name: 'Strategy: Card A' })
		await expect(card).toBeVisible()
		await expect(toggle).toHaveAttribute('aria-expanded', 'false')

		// Interactive elements are never nested (invalid HTML, a11y failure)
		expect(await page.locator('[data-testid^="strategy-card-"] button button').count()).toBe(0)

		// Tag button sits above the toggle overlay: filters, does not expand the card
		await card.getByRole('button', { name: 'Filter by tag alpha' }).click()
		await expect(card.getByRole('button', { name: 'Filter by tag alpha' })).toHaveAttribute('aria-pressed', 'true')
		await expect(page.locator('[data-testid="strategy-card-strat-card-b"]')).toHaveCount(0)
		await expect(toggle).toHaveAttribute('aria-expanded', 'false')

		// Clicking on the summary toggles the card: the stretched toggle button's overlay
		// covers it, so click at its position like a user (element click would be intercepted)
		const summaryBox = await card.getByText('Summary of card A').boundingBox()
		await page.mouse.click(summaryBox.x + summaryBox.width / 2, summaryBox.y + summaryBox.height / 2)
		await expect(toggle).toHaveAttribute('aria-expanded', 'true')
		await expect(card.getByText('Body of card A')).toBeVisible()

		// Keyboard: Enter on the title button collapses it again
		await toggle.focus()
		await page.keyboard.press('Enter')
		await expect(toggle).toHaveAttribute('aria-expanded', 'false')
	})
})
