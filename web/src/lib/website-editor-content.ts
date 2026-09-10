import {articles} from './blog/articles';
import {websiteDefaults as defaults,websitePages as pages,type WebsitePage,type WebsiteCopy} from './website-content';
export * from './website-content';
const guideDefaults=Object.fromEntries(articles.map(a=>['guide-'+a.slug,{title:a.title,accent:'',introduction:a.intro,sections:a.sections.map(s=>({heading:s.heading,body:s.paragraphs.join('\n\n'),...(s.points?.length?{points:s.points}:{})})),guide:{description:a.description,category:a.category,keyphrase:a.keyphrase,faq:a.faq.map(q=>({heading:q.question,body:q.answer})),sources:a.sources,related:a.related.map(r=>r.slug)}}]));
export const websiteDefaults:Record<WebsitePage,WebsiteCopy>={...defaults,...guideDefaults};
export const websitePages:Record<WebsitePage,string>={...pages,...Object.fromEntries(articles.map(a=>['guide-'+a.slug,'Guide: '+a.title]))};
