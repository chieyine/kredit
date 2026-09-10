import {env} from '$env/dynamic/private';
import {dev} from '$app/environment';
import {error} from '@sveltejs/kit';
import {legalPublication} from './legal-publication';
import {decodeWebsitePublication} from '$lib/website-content';
export type LegalPage='terms'|'privacy'|'complaints';
const initialVersions={terms:legalPublication.termsVersion,privacy:legalPublication.privacyVersion,complaints:legalPublication.complaintsVersion};
export async function loadLegalContent(fetcher:typeof fetch,page:LegalPage,version:string|null){
 if(version===initialVersions[page])return {legal:legalPublication,publication:null};
 if(dev&&!env.API_INTERNAL_URL){if(version)error(404,'That published document was not found.');return {legal:legalPublication,publication:null};}
 let response:Response;
 try {response=await fetcher(`/api/v1/website/${page}${version?`?version=${encodeURIComponent(version)}`:''}`,{credentials:'omit',cache:'no-store',signal:AbortSignal.timeout(5000)});}
 catch {error(503,'This legal document could not be verified. Please try again.');}
 if(response.status===404)error(404,'That published document was not found.');
 if(!response.ok)error(503,'This legal document could not be verified. Please try again.');
 try {
  const result=await response.json();
  if(result.publication===null&&!version)return {legal:legalPublication,publication:null};
  const publication=decodeWebsitePublication(result.publication);
  if(publication.document_version!==`legal-${page}-v${publication.version}`||!publication.effective_date||!/^\d{4}-\d{2}-\d{2}$/.test(publication.effective_date)||new Date(`${publication.effective_date}T00:00:00Z`).toISOString().slice(0,10)!==publication.effective_date||(version&&publication.document_version!==version))throw new Error('Unverified publication');
  return {legal:legalPublication,publication};
 }catch{error(503,'This legal document could not be verified. Please try again.');}
}
