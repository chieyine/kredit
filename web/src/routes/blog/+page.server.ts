import {loadGuides} from '$lib/server/guides';
import type {PageServerLoad} from './$types';
export const load:PageServerLoad=async({fetch,setHeaders})=>{setHeaders({'cache-control':'no-store'});return {articles:await loadGuides(fetch)}};
