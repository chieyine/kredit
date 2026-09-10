import {loadLegalContent} from '$lib/server/legal-content';
export async function load({fetch,url}:{fetch:typeof globalThis.fetch;url:URL}){return loadLegalContent(fetch,'complaints',url.searchParams.get('version'));}
