import { expect, test } from '@playwright/test'
import { startTestServer } from './helpers/test-server'

// Browsers (Chrome Issues panel) flag form fields without an id or name attribute.
test.describe('Form field accessibility', () => {
	let server

	test.beforeAll(async () => {
		server = await startTestServer({
			customTasks: [
				{
					id: 'form-field-task',
					title: 'Form Field Task',
					status: 'in_progress',
					priority: 'low',
					tags: [],
					summary: 'Has checklist and note input',
				},
			],
		})
	})

	test.afterAll(async () => {
		await server?.stop()
	})

	test('every input, textarea and select has an id or name', async ({
		page,
	}) => {
		const unnamedFields = () =>
			page.evaluate(() =>
				[...document.querySelectorAll('input, textarea, select')]
					.filter((el) => !el.id && !el.getAttribute('name'))
					.map((el) => el.outerHTML.slice(0, 120)),
			)

		await page.goto(server.url)
		expect(await unnamedFields()).toEqual([])

		// Task modal: server-rendered checklist checkboxes and note textarea
		await page.goto(`${server.url}/#task/form-field-task`)
		await expect(page.locator('[data-testid="add-note-input"]')).toBeVisible()
		expect(await unnamedFields()).toEqual([])

		await page.keyboard.press('Escape')
		await page.locator('[data-testid="about-modal-trigger"]').click()
		expect(await unnamedFields()).toEqual([])
	})
})
