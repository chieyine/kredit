import type { PageServerLoad } from './$types';
import { loadWebsiteCopy } from '$lib/server/website-content';
export const load: PageServerLoad = async ({fetch, setHeaders}) => {
 setHeaders({'cache-control':'no-store'});
 return {copy:await loadWebsiteCopy(fetch,'home')};
};
