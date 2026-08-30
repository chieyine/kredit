import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
	testDir: './tests',
	timeout: 60_000,
	workers: 1,
	expect: { timeout: 30_000 },
	use: {
		baseURL: process.env.PLAYWRIGHT_BASE_URL ?? 'http://127.0.0.1:5173',
		trace: 'retain-on-failure'
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
	webServer: process.env.PLAYWRIGHT_BASE_URL
		? undefined
		: { command: './node_modules/.bin/vite dev --host 127.0.0.1', cwd: '.', url: 'http://127.0.0.1:5173', reuseExistingServer: true }
});
