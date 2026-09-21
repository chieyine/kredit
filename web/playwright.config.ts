import { defineConfig, devices } from '@playwright/test';

const ci = Boolean(process.env.CI);
const externalBaseURL = process.env.PLAYWRIGHT_BASE_URL;
// Never substitute a fixture for an explicitly configured real stack.
const localPublicAPI = !externalBaseURL && !process.env.API_INTERNAL_URL;
const publicFixtureURL = 'http://127.0.0.1:5174';

export default defineConfig({
	testDir: './tests',
	timeout: 60_000,
	workers: 1,
	forbidOnly: ci,
	retries: 0,
	expect: { timeout: 30_000 },
	use: {
		baseURL: externalBaseURL ?? 'http://127.0.0.1:5173',
		serviceWorkers: 'block',
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure',
		launchOptions: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE
			? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE, args: ['--no-sandbox'] }
			: undefined
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
	webServer: externalBaseURL
		? undefined
		: [
			...(localPublicAPI ? [{
				command: 'node --test test-support/public-api.test.mjs && node test-support/public-api.mjs',
				env: { KREDIT_PUBLIC_API_FIXTURE: '1' },
				url: `${publicFixtureURL}/healthz`, reuseExistingServer: false, timeout: 30_000
			}] : []),
			{
			command: `./node_modules/.bin/vite ${ci ? 'preview' : 'dev'} --host 127.0.0.1 --port 5173 --strictPort`,
			cwd: '.', url: 'http://127.0.0.1:5173', reuseExistingServer: !ci && !localPublicAPI, timeout: 180_000,
			env: localPublicAPI ? { API_INTERNAL_URL: publicFixtureURL } : {}
			}
		]
});
