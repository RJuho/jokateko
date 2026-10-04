import { expect, test } from '@playwright/test'
import { spawn } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const binaryPath = '/workspaces/jokateko/bin/jokateko'
const mermaidRuntime = JSON.parse(
	readFileSync('/workspaces/jokateko/web/dist/mermaid.json', 'utf8'),
)
const mermaidMinJs = readFileSync(
	'/workspaces/jokateko/web/node_modules/mermaid/dist/mermaid.min.js',
)
const CDN_PATTERN = /^https:\/\/cdn\.jsdelivr\.net\/npm\/mermaid@.*\/dist\/mermaid\.min\.js$/
const TASK_ID = 'task-diagram'

function runBinary(args) {
	return new Promise((resolve, reject) => {
		const proc = spawn(binaryPath, args)
		proc.on('exit', (code) => {
			if (code === 0) resolve()
			else reject(new Error(`jokateko ${args[0]} failed: ${code}`))
		})
	})
}

// Opens the exported task modal and records every non-file:// request.
async function openTask(page, exportPath) {
	const requests = []
	page.on('request', (req) => {
		if (!req.url().startsWith('file://') && !req.url().startsWith('data:')) {
			requests.push(req.url())
		}
	})
	await page.goto(`file://${exportPath}#task/${TASK_ID}`)
	await expect(page.locator('[data-testid="task-detail-modal"]')).toBeVisible()
	return requests
}

test.describe('Static Export Mermaid modes (--mermaidjs)', () => {
	let dir
	const exports = {}

	test.beforeAll(async () => {
		dir = mkdtempSync(join(tmpdir(), 'jokateko-static-mermaid-'))
		await runBinary(['init', '-dir', dir])

		writeFileSync(
			join(dir, '.jokateko', 'tasks', `${TASK_ID}.md`),
			`+++
id = "${TASK_ID}"
title = "Diagram Task"
status = "backlog"
priority = "medium"
tags = []
summary = "Task with a Mermaid diagram"
+++
## Flow

\`\`\`mermaid
graph TD
  A[Start] --> B[Done]
\`\`\`
`,
			'utf8',
		)

		for (const mode of ['bundled', 'cdn', 'none']) {
			exports[mode] = join(dir, `${mode}.html`)
			await runBinary(['build', '-dir', dir, '-out', exports[mode], `--mermaidjs=${mode}`])
		}
	})

	test.afterAll(() => {
		if (dir) rmSync(dir, { recursive: true, force: true })
	})

	test('bundled: renders diagrams fully offline from the inline block', async ({ page }) => {
		const requests = await openTask(page, exports.bundled)
		await expect(page.locator('[data-testid="mermaid-diagram"] .mermaid-inner svg')).toBeVisible()
		expect(requests).toEqual([])
	})

	test('cdn: loads the pinned version from jsDelivr with SRI', async ({ page }) => {
		const cdnRequests = []
		// Serve the lockfile-pinned npm file instead of hitting the network; SRI must still pass
		await page.route(CDN_PATTERN, (route) => {
			cdnRequests.push(route.request().url())
			route.fulfill({
				body: mermaidMinJs,
				contentType: 'text/javascript',
				headers: { 'Access-Control-Allow-Origin': '*' },
			})
		})

		await openTask(page, exports.cdn)
		await expect(page.locator('[data-testid="mermaid-diagram"] .mermaid-inner svg')).toBeVisible()
		expect(cdnRequests).toEqual([
			`https://cdn.jsdelivr.net/npm/mermaid@${mermaidRuntime.version}/dist/mermaid.min.js`,
		])
		await expect(page.locator('script[src*="cdn.jsdelivr.net"]')).toHaveAttribute(
			'integrity',
			mermaidRuntime.integrity,
		)
	})

	test('cdn: tampered runtime is blocked by SRI and the raw block is kept', async ({ page }) => {
		await page.route(CDN_PATTERN, (route) =>
			route.fulfill({
				body: `${mermaidMinJs};/* tampered */`,
				contentType: 'text/javascript',
				headers: { 'Access-Control-Allow-Origin': '*' },
			}),
		)

		await openTask(page, exports.cdn)
		await expect(page.locator('pre > code.language-mermaid')).toBeVisible()
		await expect(page.locator('[data-testid="mermaid-diagram"]')).toHaveCount(0)
		expect(await page.evaluate(() => typeof globalThis.mermaid)).toBe('undefined')
	})

	test('none: keeps Mermaid blocks as code without loading anything', async ({ page }) => {
		const requests = await openTask(page, exports.none)
		await expect(page.locator('pre > code.language-mermaid')).toBeVisible()
		await expect(page.locator('[data-testid="mermaid-diagram"]')).toHaveCount(0)
		expect(requests).toEqual([])
	})
})
