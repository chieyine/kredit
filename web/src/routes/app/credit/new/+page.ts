import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ url }) => {
	if (url.searchParams.get('advanced') !== '1') {
		redirect(307, '/app/credit/quick');
	}
};
