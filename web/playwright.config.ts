import { defineConfig, devices } from '@playwright/test';

const ci = Boolean(process.env.CI);
const externalBaseURL = process.env.PLAYWRIGHT_BASE_URL;

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
		: {
			command: `./node_modules/.bin/vite ${ci ? 'preview' : 'dev'} --host 127.0.0.1 --port 5173 --strictPort`,
			cwd: '.', url: 'http://127.0.0.1:5173', reuseExistingServer: !ci
		}
});
