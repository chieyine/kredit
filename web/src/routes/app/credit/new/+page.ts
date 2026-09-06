import { redirect } from '@sveltejs/kit';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ url }) => {
	if (url.searchParams.get('advanced') !== '1') {
		const params = new URLSearchParams(url.searchParams);
		params.delete('advanced');
		const query = params.toString();
		redirect(307, `/app/credit/quick${query ? `?${query}` : ''}`);
	}
};
