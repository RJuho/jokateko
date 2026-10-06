import { expect, test } from '@playwright/test'
import { readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { tmpdir } from 'node:os'
import { WEB_DIR } from './helpers/paths'

test.describe('Valibot Schema Resilience & Error Boundary E2E', () => {
	const templateHtml = readFileSync(
		join(WEB_DIR, 'dist', 'index.html'),
		'utf8',
	)

	test('displays validation error banner on schema issue without crashing', async ({
		page,
	}) => {
		// Inject snapshot with schema issues (invalid project name type, missing columns)
		const invalidPayload = {
			config: {
				project: { name: 12345 },
				board: {
					columns: [
						{ id: 'backlog', name: 'Backlog', color: '#64748b' },
					],
				},
			},
			tasks: [
				{
					id: 'task-1',
					title: 'Resilient Task',
					status: 'backlog',
					priority: 'high',
					summary: 'Testing resilience',
				},
			],
			milestones: [],
			strategies: [],
			glossary: [],
		}

		const testHtml = templateHtml.replace(
			'/* JOKATEKO_PAYLOAD_PLACEHOLDER */',
			JSON.stringify(invalidPayload),
		)
		const tempFile = join(tmpdir(), `valibot-schema-resilience-${Date.now()}.html`)
		writeFileSync(tempFile, testHtml, 'utf8')

		await page.goto(`file://${tempFile}`)

		// 1. Verify validation error banner is visible
		const banner = page.locator('[data-testid="validation-error-banner"]')
		await expect(banner).toBeVisible()
		await expect(banner).toContainText('Schema Warning')

		// 2. Verify board renders the resilient task and column despite warnings
		const col = page.locator('[data-testid="column-backlog"]')
		await expect(col).toBeVisible()

		const card = page.locator('[data-testid="task-card-task-1"]')
		await expect(card).toBeVisible()
		await expect(card).toContainText('Resilient Task')
	})

	test('gracefully catches corrupt JSON syntax without crashing UI', async ({
		page,
	}) => {
		const corruptPayload = '{\n  "corrupt": json syntax without quotes\n}'
		const testHtml = templateHtml.replace(
			'/* JOKATEKO_PAYLOAD_PLACEHOLDER */',
			corruptPayload,
		)
		const tempFile = join(tmpdir(), `valibot-syntax-resilience-${Date.now()}.html`)
		writeFileSync(tempFile, testHtml, 'utf8')

		// Catch any unhandled page errors
		const errors = []
		page.on('pageerror', (err) => errors.push(err.message))

		await page.goto(`file://${tempFile}`)

		// 1. Verify validation error banner reports schema warning
		const banner = page.locator('[data-testid="validation-error-banner"]')
		await expect(banner).toBeVisible()
		await expect(banner).toContainText('Schema Warning')

		// 2. Expand details and verify JSON parse failure message
		const viewDetailsBtn = page.getByRole('button', {
			name: 'Show issue details',
		})
		await viewDetailsBtn.click()
		await expect(banner).toContainText('Failed to parse embedded JSON payload')

		// 3. Verify no uncaught fatal JavaScript errors crashed the page
		expect(errors).toHaveLength(0)
	})
})
