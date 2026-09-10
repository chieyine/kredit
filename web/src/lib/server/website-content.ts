import { decodeWebsitePublication, websiteDefaults, type WebsitePage } from '$lib/website-content';
export async function loadWebsiteCopy(fetcher: typeof fetch, page: Extract<WebsitePage,'home'|'faq'|'pricing'|'contact'>) {
 try {
  const response = await fetcher(`/api/v1/website/${page}`,{credentials:'omit',cache:'no-store',signal:AbortSignal.timeout(5000)});
  if (!response.ok) throw new Error('Publication unavailable');
  const result = await response.json();
  if (result.publication !== null) return decodeWebsitePublication(result.publication).copy;
 } catch { /* The initial release copy remains available during a content-service outage. */ }
 return structuredClone(websiteDefaults[page]);
}
