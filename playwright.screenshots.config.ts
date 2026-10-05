import { defineConfig, devices } from '@playwright/test'

// Captures README screenshots of a demo workspace and renders the cover image
// from them; output goes to the gitignored screenshots/ folder.
export default defineConfig({
	testDir: './tests/screenshots',
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
