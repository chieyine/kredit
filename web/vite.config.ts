import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': {
				target: (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env?.API_INTERNAL_URL ?? 'http://localhost:8080',
				changeOrigin: false
			}
		}
	}
});
