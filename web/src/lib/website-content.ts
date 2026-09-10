import marketingDefaults from './website-defaults.json' with { type: 'json' };
import legalDefaults from './legal-defaults.json' with { type: 'json' };
const defaults={...marketingDefaults,...legalDefaults,contact:{title:'Contact Kredit',accent:'',introduction:'Tell us how we can help. Include your sale reference, but never send a sign-in code or bank password.',sections:[],contact:{support_email:'hello@kredit.com.ng',privacy_email:'hello@kredit.com.ng',phone:'',address:'House No. 348, Jamaina Road, Pompomari Bypass, Maiduguri, Borno State, Nigeria'}}};
export type WebsitePage = keyof typeof marketingDefaults | keyof typeof legalDefaults | 'contact' | `guide-${string}`;
export type WebsiteCopy = { title: string; accent: string; introduction: string; sections: { heading: string; body: string; points?:string[] }[]; guide?:{description:string;category:string;keyphrase:string;faq:{heading:string;body:string}[];sources:{name:string;url:string;note:string}[];related:string[]}; contact?:{support_email:string;privacy_email:string;phone:string;address:string} };
export type WebsitePublication = {document_version?:string;effective_date?:string;copy: WebsiteCopy; version: number; published_at: string; content_hash: string};
export type WebsiteRecord = {version: number; content: {draft: WebsiteCopy; published?: WebsitePublication} | null};
export const websiteDefaults: Record<WebsitePage, WebsiteCopy> = defaults as Record<WebsitePage,WebsiteCopy>;
export const websitePages: Record<WebsitePage, string> = {home:'Homepage introduction',faq:'Common questions',pricing:'Pricing introduction',terms:'Terms of service',privacy:'Privacy notice',complaints:'Complaints policy',contact:'Contact details'};
export function decodeWebsiteCopy(value: unknown): WebsiteCopy {
 const v = value as WebsiteCopy;
 const text = (s: unknown, max: number, required = true) => typeof s === 'string' && [...s].length <= max && (!required || s.trim().length > 0) && !/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/.test(s);
 if (!v || typeof v !== 'object' || !text(v.title,160) || !text(v.accent,160,false) || !text(v.introduction,2000) || !Array.isArray(v.sections) || v.sections.length>40 || !v.sections.every(s=>s && text(s.heading,200) && text(s.body,6000))) throw new Error('The page content was incomplete. Reload before editing.');
 for(const section of v.sections) if(section.points && (!Array.isArray(section.points)||section.points.length>40||!section.points.every(p=>text(p,2000)))) throw new Error('Check the guide bullet points.');
 if(v.guide){const g=v.guide;if(!text(g.description,2000)||!text(g.keyphrase,200)||!['Credit sales','Customer checks','Agreements','Payments','Late payment','Cash flow','Business records','Safe payments','Industry guides','Business growth'].includes(g.category)||!Array.isArray(g.faq)||g.faq.length>40||!g.faq.every(q=>q&&text(q.heading,200)&&text(q.body,6000))||!Array.isArray(g.sources)||g.sources.length>40||!g.sources.every(s=>{try{const u=new URL(s.url);return text(s.name,200)&&text(s.note,2000,false)&&s.url.length<=2000&&u.protocol==='https:'&&!u.username&&!u.password}catch{return false}})||!Array.isArray(g.related)||g.related.length>12||new Set(g.related).size!==g.related.length||!g.related.every(slug=>typeof slug==='string'&&/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)))throw new Error('Check the guide details, questions, sources and related guides.');}
 if(v.contact){const c=v.contact;if(![c.support_email,c.privacy_email].every(e=>text(e,254)&&/^[^\s@<>]+@[^\s@<>]+$/.test(e))||!text(c.address,2000)||!text(c.phone,40,false)||!/^[-+0-9 ()]*$/.test(c.phone))throw new Error('Check the contact email, phone and address.');}
 return JSON.parse(JSON.stringify(v));
}
export function decodeWebsitePublication(value: unknown): WebsitePublication {
 const v=value as WebsitePublication;
 if (!v || !Number.isSafeInteger(v.version) || v.version<1 || typeof v.content_hash!=='string' || !/^[a-f0-9]{64}$/.test(v.content_hash) || typeof v.published_at!=='string' || !Number.isFinite(Date.parse(v.published_at))) throw new Error('The publication could not be verified.');
 return {...v,copy:decodeWebsiteCopy(v.copy)};
}
export function decodeWebsiteRecord(value: unknown): WebsiteRecord {
 const v=value as WebsiteRecord;
 if (!v || !Number.isSafeInteger(v.version) || v.version<0 || (v.version===0 ? v.content!==null : !v.content)) throw new Error('The page version was incomplete. Reload before editing.');
 return {version:v.version,content:v.content ? {draft:decodeWebsiteCopy(v.content.draft),published:v.content.published ? decodeWebsitePublication(v.content.published):undefined}:null};
}

export const isLegalPage=(page:WebsitePage)=>["terms","privacy","complaints"].includes(page);

export const isGuidePage=(page:WebsitePage)=>page.startsWith("guide-");
