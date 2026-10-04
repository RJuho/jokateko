import { defineConfig, devices } from '@playwright/test'

// Lighthouse audits attach to Playwright Chromium over a CDP debugging port,
// so they run serially in a single worker with a generous per-test timeout.
export default defineConfig({
	testDir: './tests/lighthouse',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: 0,
	workers: 1,
	timeout: 120_000,
	reporter: [['list']],
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] },
		},
	],
})
