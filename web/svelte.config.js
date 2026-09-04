import vercelAdapter from '@sveltejs/adapter-vercel';
import nodeAdapter from '@sveltejs/adapter-node';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
		adapter: process.env.KREDIT_WEB_ADAPTER === 'node' ? nodeAdapter() : vercelAdapter(),
		alias: {
			$features: 'src/lib/features',
			$components: 'src/lib/components'
		}
	}
};

export default config;
