import vercelAdapter from '@sveltejs/adapter-vercel';
import nodeAdapter from '@sveltejs/adapter-node';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: process.env.KREDIT_WEB_ADAPTER === 'node' ? nodeAdapter() : vercelAdapter(),
		alias: {
			$components: 'src/lib/components'
		},
		// SvelteKit augments these directives with hashes/nonces for framework
		// generated inline scripts. This removes script-src 'unsafe-inline' while
		// keeping style-src compatible with Svelte transition-generated styles.
		csp: {
			mode: 'auto',
			directives: {
				'default-src': ['self'],
				'script-src': ['self'],
				'style-src': ['self', 'unsafe-inline'],
				'img-src': ['self', 'data:'],
				'font-src': ['self', 'data:'],
				'connect-src': ['self'],
				'frame-ancestors': ['none'],
				'base-uri': ['self'],
				'form-action': ['self']
			}
		}
	}
};

export default config;
