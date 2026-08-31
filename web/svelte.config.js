import adapter from '@sveltejs/adapter-vercel';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter(),
		alias: {
			$features: 'src/lib/features',
			$components: 'src/lib/components'
		}
	}
};

export default config;
