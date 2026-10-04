import { mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { chromium, expect } from '@playwright/test'
import lighthouse, { desktopConfig } from 'lighthouse'

// Types are derived rather than imported: the Bun-backed Playwright runner
// cannot transpile type-only imports.
type Flags = NonNullable<Parameters<typeof lighthouse>[1]>
export type Result = NonNullable<Awaited<ReturnType<typeof lighthouse>>>['lhr']

export const REPORT_DIR = join(process.cwd(), 'lighthouse-report')

const CATEGORIES = {
	performance: 'LH_MIN_PERFORMANCE',
	accessibility: 'LH_MIN_ACCESSIBILITY',
	'best-practices': 'LH_MIN_BEST_PRACTICES',
	seo: 'LH_MIN_SEO',
} as const

const DEFAULT_MINIMUMS: Record<keyof typeof CATEGORIES, number> = {
	performance: 90,
	accessibility: 90,
	'best-practices': 90,
	seo: 80,
}

/**
 * Launch Playwright Chromium with a CDP remote debugging port that
 * Lighthouse attaches to directly (one port per worker).
 */
export async function launchAuditBrowser(workerIndex = 0) {
	const port = 9222 + workerIndex
	const browser = await chromium.launch({
		args: [`--remote-debugging-port=${port}`],
	})
	const page = await browser.newPage()
	return { browser, page, port }
}

/**
 * Run Lighthouse against url over the given CDP port and write
 * lighthouse-report/<name>.html and <name>.json.
 */
export async function runAudit(url: string, name: string, port: number): Promise<Result> {
	const flags: Flags = {
		port,
		output: ['html', 'json'],
		logLevel: 'error',
		onlyCategories: Object.keys(CATEGORIES),
		// The UI keeps an SSE connection open; cap load waiting so audits never hang on it.
		maxWaitForLoad: 15_000,
	}
	const result = await lighthouse(url, flags, desktopConfig)
	if (!result) {
		throw new Error(`Lighthouse returned no result for ${url}`)
	}

	mkdirSync(REPORT_DIR, { recursive: true })
	const [html, json] = result.report as string[]
	writeFileSync(join(REPORT_DIR, `${name}.html`), html)
	writeFileSync(join(REPORT_DIR, `${name}.json`), json)
	return result.lhr
}

function minimumFor(category: keyof typeof CATEGORIES): number {
	const raw = process.env[CATEGORIES[category]]
	const parsed = raw === undefined ? Number.NaN : Number(raw)
	return Number.isFinite(parsed) ? parsed : DEFAULT_MINIMUMS[category]
}

/**
 * Soft-assert every category against its configured minimum and log
 * the failing audits of any category that falls short.
 */
export function assertThresholds(lhr: Result, name: string): void {
	const scores: string[] = []
	for (const category of Object.keys(CATEGORIES) as Array<keyof typeof CATEGORIES>) {
		const cat = lhr.categories[category]
		const score = Math.round((cat?.score ?? 0) * 100)
		const min = minimumFor(category)
		scores.push(`${category}=${score}`)

		if (score < min && cat) {
			const failing = cat.auditRefs
				.filter((ref) => ref.weight > 0)
				.map((ref) => lhr.audits[ref.id])
				.filter((a) => a.score !== null && a.score < 1)
				.map((a) => `${a.id} (${Math.round((a.score ?? 0) * 100)})`)
			console.log(`[${name}] ${category} failing audits: ${failing.join(', ')}`)
		}

		expect.soft(score, `${name}: ${category} score ${score} below minimum ${min}`).toBeGreaterThanOrEqual(min)
	}
	console.log(`[${name}] ${scores.join(' ')}`)
}
