import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { expect, test } from '@playwright/test'
import { startTestServer } from '../e2e/helpers/test-server'
import { DEMO_MONTH, seedDemoWorkspace } from './demo-workspace'

// Gitignored output: per-view captures, the cover page built from them, and its renders
const OUT_DIR = join(process.cwd(), 'screenshots')
const VIEWS_DIR = join(OUT_DIR, 'views')

const COVER_WIDTH = 1280
const COVER_HEIGHT = 640


// A mix of views, a few in dark mode, ordered as they appear on the cover wall
const VIEWS = [
	{ name: 'board', hash: '#board', scheme: 'light', ready: '[data-testid^="column-header-"]' },
	{ name: 'task-detail', hash: '#task/260901-user-auth', scheme: 'dark', ready: '[data-testid="task-detail-modal"]' },
	{ name: 'calendar-month', hash: `#calendar/${DEMO_MONTH}`, scheme: 'light', ready: '[data-testid="calendar-view"]' },
	{ name: 'board', hash: '#board', scheme: 'dark', ready: '[data-testid^="column-header-"]' },
	{ name: 'strategies', hash: '#strategy/strat-tasks-as-code', scheme: 'light', ready: '[data-testid="strategy-card-strat-tasks-as-code"] [aria-expanded="true"]' },
	{ name: 'milestone', hash: '#milestone/m1-mvp-release', scheme: 'dark', ready: '[data-testid^="column-header-"]' },
	{ name: 'glossary', hash: '#glossary/term-mcp', scheme: 'dark', ready: '[data-testid="glossary-card-term-mcp"] [aria-expanded="true"]' },
	{ name: 'task-detail', hash: '#task/260901-kanban-board', scheme: 'light', ready: '[data-testid="task-detail-modal"]' },
	{ name: 'calendar-month', hash: `#calendar/${DEMO_MONTH}`, scheme: 'dark', ready: '[data-testid="calendar-view"]' },
]

const fileName = (v) => `${v.name}-${v.scheme}.png`

async function capture(page, baseURL, view) {
	await page.emulateMedia({ colorScheme: view.scheme, reducedMotion: 'reduce' })
	await page.goto(`${baseURL}/${view.hash}`)
	await expect(page.locator(view.ready).first()).toBeVisible()
	await page.evaluate(() => document.fonts.ready)
	await page.screenshot({ path: join(VIEWS_DIR, fileName(view)), animations: 'disabled', caret: 'hide' })
}

test.describe.configure({ mode: 'serial' })

test.describe('README screenshots', () => {
	let server

	test.beforeAll(async () => {
		mkdirSync(VIEWS_DIR, { recursive: true })
		server = await startTestServer({ seed: seedDemoWorkspace })
	})

	test.afterAll(async () => {
		await server?.stop()
	})

	test('capture views', async ({ browser }) => {
		const context = await browser.newContext({
			viewport: { width: 1440, height: 900 },
			deviceScaleFactor: 1,
			locale: 'en-US',
		})
		// Fresh page per view: hash-only navigations would reuse the previous view's state
		for (const view of VIEWS) {
			const page = await context.newPage()
			await capture(page, server.url, view)
			await page.close()
		}
		await context.close()
	})

	test('render cover', async ({ browser }) => {
		// The cover page lives next to the captures so it can be opened and tweaked by hand
		const template = readFileSync(join(__dirname, 'cover.html'), 'utf8')
		const shots = VIEWS.map((v) => `views/${fileName(v)}`)
		const coverPath = join(OUT_DIR, 'cover.html')
		const logo = readFileSync(join(__dirname, '../../web/src/logo.svg'), 'utf8')
		const html = template.replace('/*SHOTS*/[]', JSON.stringify(shots)).replace('<!--LOGO-->', logo)
		writeFileSync(coverPath, html, 'utf8')

		const context = await browser.newContext({
			viewport: { width: COVER_WIDTH, height: COVER_HEIGHT },
			deviceScaleFactor: 2,
		})
		const page = await context.newPage()
		await page.goto(`file://${coverPath}`)
		await page.waitForFunction(() => [...document.images].every((img) => img.complete && img.naturalWidth > 0))
		await page.evaluate(() => document.fonts.ready)

		// 2x PNG master, 2x JPEG for the README (.github/assets/cover.jpg),
		// 1x JPEG for the GitHub social preview (1 MB limit)
		await page.screenshot({ path: join(OUT_DIR, 'cover.png') })
		await page.screenshot({ path: join(OUT_DIR, 'cover.jpg'), type: 'jpeg', quality: 85 })
		await context.close()

		const social = await browser.newContext({ viewport: { width: COVER_WIDTH, height: COVER_HEIGHT } })
		const socialPage = await social.newPage()
		await socialPage.goto(`file://${coverPath}`)
		await socialPage.waitForFunction(() => [...document.images].every((img) => img.complete && img.naturalWidth > 0))
		await socialPage.screenshot({ path: join(OUT_DIR, 'cover-social.jpg'), type: 'jpeg', quality: 88 })
		await social.close()
	})
})
