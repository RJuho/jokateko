import { expect, test } from '@playwright/test'
import { startTestServer } from '../e2e/helpers/test-server'
import { assertThresholds, launchAuditBrowser, runAudit } from './helpers/audit'

const MILESTONE_ID = 'm-lh-release'
const MILESTONE_TITLE = 'Lighthouse Release'
const TASK_ID = 'task-lh-targeted'
const TASK_TITLE = 'Lighthouse Targeted Task'
const STRATEGY_ID = 'strat-lh-performance'
const GLOSSARY_ID = 'gloss-lh-audit'
const MONTH = '2026-09'
const WEEK = '2026-W38'

const visible = (selector) => async (page) => {
	await expect(page.locator(selector).first()).toBeVisible()
}

// Every top-level view plus each deep link that points at a resource.
const VIEWS = [
	{ name: 'board', hash: '#board', title: /^Board · /, ready: visible('[data-testid^="column-header-"]') },
	{ name: 'board-task', hash: `#task/${TASK_ID}`, title: new RegExp(`^${TASK_TITLE} · `), ready: visible('[data-testid="task-detail-modal"]') },
	{ name: 'board-milestone', hash: `#milestone/${MILESTONE_ID}`, title: new RegExp(`^${MILESTONE_TITLE} · `), ready: visible('[data-testid^="column-header-"]') },
	{ name: 'calendar-month', hash: `#calendar/${MONTH}`, title: new RegExp(`^Calendar \\(${MONTH}\\) · `), ready: visible('[data-testid="calendar-view"]') },
	{ name: 'calendar-week', hash: `#calendar/${WEEK}`, title: new RegExp(`^Calendar \\(${WEEK}\\) · `), ready: visible('[data-testid="calendar-view"]') },
	{ name: 'calendar-task', hash: `#calendar/${MONTH}/task/${TASK_ID}`, title: new RegExp(`^${TASK_TITLE} · `), ready: visible('[data-testid="task-detail-modal"]') },
	{ name: 'calendar-milestone', hash: `#calendar/${MONTH}/milestone/${MILESTONE_ID}`, title: new RegExp(`^${MILESTONE_TITLE} · `), ready: visible('[data-testid="calendar-view"]') },
	{ name: 'strategies', hash: '#strategies', title: /^Strategies · /, ready: visible(`[data-testid="strategy-card-${STRATEGY_ID}"]`) },
	{ name: 'strategy-detail', hash: `#strategy/${STRATEGY_ID}`, title: /^Lighthouse Performance Budget · /, ready: visible(`[data-testid="strategy-card-${STRATEGY_ID}"] [aria-expanded="true"]`) },
	{ name: 'glossary', hash: '#glossary', title: /^Glossary · /, ready: visible(`[data-testid="glossary-card-${GLOSSARY_ID}"]`) },
	{ name: 'glossary-term', hash: `#glossary/${GLOSSARY_ID}`, title: /^Lighthouse Audit · /, ready: visible(`[data-testid="glossary-card-${GLOSSARY_ID}"] [aria-expanded="true"]`) },
]

test.describe('Lighthouse audits', () => {
	let server
	let audit

	test.beforeAll(async ({}, testInfo) => {
		server = await startTestServer({
			customMilestones: [
				{
					id: MILESTONE_ID,
					title: MILESTONE_TITLE,
					status: 'open',
					target_date: '2026-09-18',
					summary: 'Milestone used by the Lighthouse audit suite',
				},
			],
			customTasks: [
				{
					id: TASK_ID,
					title: TASK_TITLE,
					status: 'in_progress',
					priority: 'high',
					milestone: MILESTONE_ID,
					tags: ['frontend'],
					summary: 'Task with a target date inside the audited calendar period',
					target_at: '2026-09-15T12:00:00Z',
				},
				{
					id: 'task-lh-backlog',
					title: 'Lighthouse Backlog Task',
					status: 'backlog',
					priority: 'medium',
					tags: ['backend'],
					summary: 'Untargeted backlog task',
				},
			],
			customStrategies: [
				{
					id: STRATEGY_ID,
					title: 'Lighthouse Performance Budget',
					tier: 1,
					summary: 'Keep every Web UI view within the Lighthouse score budget',
					tags: ['testing'],
					body: '## Rule\nAll views must meet the configured Lighthouse category minimums.\n',
				},
			],
			customGlossary: [
				{
					id: GLOSSARY_ID,
					title: 'Lighthouse Audit',
					summary: 'Automated page quality report produced by Lighthouse',
					tags: ['testing'],
					body: 'A Lighthouse audit scores performance, accessibility, best practices and SEO.\n',
				},
			],
		})
		audit = await launchAuditBrowser(testInfo.workerIndex)
	})

	test.afterAll(async () => {
		await audit?.browser.close()
		await server?.stop()
	})

	for (const view of VIEWS) {
		test(`${view.name} (${view.hash})`, async () => {
			const url = `${server.url}/${view.hash}`

			// Warm up with a fresh load (as Lighthouse does, not a same-document hash
			// change) and prove the route resolved to the intended view.
			await audit.page.goto('about:blank')
			await audit.page.goto(url, { waitUntil: 'load' })
			await view.ready(audit.page)
			await expect(audit.page).toHaveTitle(view.title)

			const lhr = await runAudit(url, view.name, audit.port)
			assertThresholds(lhr, view.name)
		})
	}
})
