import {error} from '@sveltejs/kit';
import {loadGuides} from '$lib/server/guides';
import type {PageServerLoad} from './$types';
export const load:PageServerLoad=async({params,fetch,setHeaders})=>{setHeaders({'cache-control':'no-store'});const article=(await loadGuides(fetch)).find(a=>a.slug===params.slug);if(!article)error(404,'Guide not found');return {article}};
