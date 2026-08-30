import adapter from '@sveltejs/adapter-node';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: adapter({ precompress: true }),
		alias: {
			$features: 'src/lib/features',
			$components: 'src/lib/components'
		}
	}
};

export default config;
