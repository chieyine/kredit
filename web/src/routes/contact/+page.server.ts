import {loadWebsiteCopy} from '$lib/server/website-content';
import {pageSEOByPath} from '$lib/seo';
import type {PageServerLoad} from './$types';
export const load:PageServerLoad=async({fetch,setHeaders})=>{setHeaders({'cache-control':'no-store'});return {copy:await loadWebsiteCopy(fetch,'contact'),seo:pageSEOByPath['/contact']}};
