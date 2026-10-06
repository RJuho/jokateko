import { expect, test } from '@playwright/test'
import { spawn } from 'node:child_process'
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { BINARY_PATH } from './helpers/paths'

const binaryPath = BINARY_PATH
const TASK_ID = 'task-xss'

function runBinary(args) {
	return new Promise((resolve, reject) => {
		const proc = spawn(binaryPath, args)
		proc.on('exit', (code) => {
			if (code === 0) resolve()
			else reject(new Error(`jokateko ${args[0]} failed: ${code}`))
		})
	})
}

// .jokateko/ content comes from anyone who can open a PR, and exports are shared,
// so raw HTML in a body must never run script when someone opens the export.
test.describe('Static export with malicious Markdown', () => {
	let dir
	let exportPath

	test.beforeAll(async () => {
		dir = mkdtempSync(join(tmpdir(), 'jokateko-static-xss-'))
		await runBinary(['init', '-dir', dir])

		writeFileSync(
			join(dir, '.jokateko', 'tasks', `${TASK_ID}.md`),
			`+++
id = "${TASK_ID}"
title = "Malicious Task"
status = "backlog"
priority = "medium"
tags = []
summary = "Body with raw HTML"
+++
## Payload

<img src="x" onerror="document.title='PWNED'">

Inline <script>document.title='PWNED'</script> and <svg onload="document.title='PWNED'"></svg> text.

[Click me](javascript:document.title='PWNED')
`,
			'utf8',
		)

		exportPath = join(dir, 'export.html')
		await runBinary(['build', '-dir', dir, '-out', exportPath, '--mermaidjs=none'])
	})

	test.afterAll(() => {
		if (dir) rmSync(dir, { recursive: true, force: true })
	})

	test('raw HTML is shown as text and no script runs', async ({ page }) => {
		await page.goto(`file://${exportPath}#task/${TASK_ID}`)
		const modal = page.locator('[data-testid="task-detail-modal"]')
		await expect(modal).toBeVisible()

		await expect(modal.getByText(`<img src="x" onerror="document.title='PWNED'">`)).toBeVisible()
		await expect(modal.locator('img[onerror], svg[onload], script')).toHaveCount(0)
		await expect(modal.locator('a', { hasText: 'Click me' })).not.toHaveAttribute(
			'href',
			/javascript:/,
		)

		// Even a link that slipped through must not run when clicked
		await modal.locator('a', { hasText: 'Click me' }).click()
		expect(await page.title()).not.toBe('PWNED')
	})

	test('export carries a CSP that forbids inline script', async ({ page }) => {
		await page.goto(`file://${exportPath}`)
		const policy = await page
			.locator('meta[http-equiv="Content-Security-Policy"]')
			.getAttribute('content')
		const scriptSrc = policy
			?.split(';')
			.map((d) => d.trim())
			.find((d) => d.startsWith('script-src '))
		expect(scriptSrc).toMatch(/^script-src 'self' 'sha256-/)
		expect(scriptSrc).not.toContain('unsafe-inline')

		// Live markup injected after load (e.g. by a future renderer regression) is blocked by CSP
		const ran = await page.evaluate(async () => {
			document.body.insertAdjacentHTML(
				'beforeend',
				`<img src="x" onerror="window.__xssRan = true">`,
			)
			await new Promise((r) => setTimeout(r, 200))
			return globalThis.__xssRan === true
		})
		expect(ran).toBe(false)
	})
})
