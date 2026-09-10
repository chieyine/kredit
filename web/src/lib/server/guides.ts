import { articles, type Article } from '$lib/blog/articles';
import { decodeWebsitePublication } from '$lib/website-content';

export async function loadGuides(fetcher:typeof fetch):Promise<Article[]> {
 let result=structuredClone(articles);
 try {
  const response=await fetcher('/api/v1/website-guides',{credentials:'omit',cache:'no-store',signal:AbortSignal.timeout(5000)});
  if(!response.ok)throw new Error('Guide publications unavailable');
  const data=await response.json();
  if(!data.publications||typeof data.publications!=='object'||Array.isArray(data.publications))throw new Error('Incomplete guide publications');
  result=result.map(article=>{
   const value=data.publications[article.slug];if(value===undefined)return article;
   const publication=decodeWebsitePublication(value),copy=publication.copy,g=copy.guide;
   if(!g||copy.sections.length===0)throw new Error('Incomplete guide');
   const sections=copy.sections.map(s=>({heading:s.heading,paragraphs:s.body.split(/\n\s*\n/),points:s.points}));
   const faq=g.faq.map(q=>({question:q.heading,answer:q.body}));
   const wordCount=[copy.introduction,...sections.flatMap(s=>[s.heading,...s.paragraphs,...(s.points??[])]),...faq.flatMap(q=>[q.question,q.answer])].join(' ').trim().split(/\s+/).length;
   return {...article,title:copy.title,intro:copy.introduction,description:g.description,category:g.category as Article['category'],keyphrase:g.keyphrase,sections,faq,sources:g.sources,related:g.related.map(slug=>({slug,title:''})),wordCount,readingMinutes:Math.max(1,Math.ceil(wordCount/220)),modified:publication.published_at.slice(0,10)};
  });
 } catch { /* Public educational copy remains available during a content outage. */ }
 const published=new Map(result.map(a=>[a.slug,a]));
 return result.map(a=>({...a,related:a.related.map(r=>({...r,title:published.get(r.slug)?.title??r.title,category:published.get(r.slug)?.category}))}));
}
