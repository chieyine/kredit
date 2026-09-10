import {loadLegalContent} from '$lib/server/legal-content';
export async function load({fetch,url}:{fetch:typeof globalThis.fetch;url:URL}){return loadLegalContent(fetch,'terms',url.searchParams.get('version'));}
