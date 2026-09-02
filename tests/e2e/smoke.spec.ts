import { expect, test } from '@playwright/test'

test('browser launches and verifies basic context in Chromium headless', async ({
	page,
}) => {
	await page.setContent(
		'<html><body><div data-testid="app">Jokateko E2E Ready</div></body></html>',
	)
	const el = page.locator('[data-testid="app"]')
	await expect(el).toHaveText('Jokateko E2E Ready')
})
