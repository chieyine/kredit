#!/usr/bin/env node

import { globSync, readFileSync } from 'node:fs';
import { articleCategories, articleCategoryDetails, articles, categorySlug } from '../web/src/lib/blog/articles.ts';
import { pageSEOByPath, publicSitemapEntries } from '../web/src/lib/seo.ts';

const failures=[];
const fail=(slug,message)=>failures.push(`${slug}: ${message}`);
if(articles.length<100)fail('blog',`only ${articles.length} long-form articles found; at least 100 are required`);
const slugs=new Set(),titles=new Set(),paragraphs=new Map();
for(const article of articles){
	if(slugs.has(article.slug))fail(article.slug,'duplicate slug');slugs.add(article.slug);
	if(titles.has(article.title))fail(article.slug,'duplicate title');titles.add(article.title);
	if(!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(article.slug))fail(article.slug,'slug is not search-friendly');
	if(article.wordCount<1500)fail(article.slug,`only ${article.wordCount} words; at least 1,500 are required`);
	if(article.description.length<120||article.description.length>160)fail(article.slug,`meta description is ${article.description.length} characters; expected 120–160`);
	if(article.title.length<25||article.title.length>68)fail(article.slug,`title is ${article.title.length} characters; expected 25–68`);
	if(article.sections.length<10)fail(article.slug,'fewer than 10 useful sections');
	if(article.faq.length<4)fail(article.slug,'fewer than 4 search questions');
	if(article.sources.length<3)fail(article.slug,'fewer than 3 authoritative sources');
	if(article.related.length<3)fail(article.slug,'fewer than 3 internal links');
	if(!/^\d{4}-\d{2}-\d{2}$/.test(article.published)||!/^\d{4}-\d{2}-\d{2}$/.test(article.modified))fail(article.slug,'published and modified dates must use YYYY-MM-DD');
	if(article.modified<article.published)fail(article.slug,'modified date is earlier than published date');
	for(const section of article.sections){
		for(const paragraph of section.paragraphs){
			const earlierSlug=paragraphs.get(paragraph);
			if(earlierSlug)fail(article.slug,`reuses a long-form paragraph from ${earlierSlug}`);
			else paragraphs.set(paragraph,article.slug);
		}
	}
}
const inboundLinks=new Map(articles.map(article=>[article.slug,0]));
for(const article of articles){
	if(new Set(article.related.map(item=>item.slug)).size!==article.related.length)fail(article.slug,'duplicate related guide');
	for(const related of article.related){
		const target=articles.find(candidate=>candidate.slug===related.slug);
		if(!target)fail(article.slug,`related guide does not exist: ${related.slug}`);
		else{
			if(target.slug===article.slug)fail(article.slug,'links to itself as a related guide');
			if(target.title!==related.title)fail(article.slug,`related title does not match ${related.slug}`);
			if(target.category!==article.category)fail(article.slug,`related guide leaves its topic cluster: ${related.slug}`);
			inboundLinks.set(target.slug,(inboundLinks.get(target.slug)??0)+1);
		}
	}
}
for(const [slug,count] of inboundLinks){if(count<3)fail(slug,`only ${count} inbound article links; expected at least 3`)}
for(const category of articleCategories){
	const details=articleCategoryDetails[category];
	if(details.slug!==categorySlug(category))fail(category,'topic hub slug does not match its category');
	if(articles.filter(article=>article.category===category).length<5)fail(category,'topic hub has fewer than 5 guides');
	if(details.description.length<100||details.description.length>160)fail(category,`topic description is ${details.description.length} characters; expected 100–160`);
}
const articleTemplate=readFileSync('web/src/routes/blog/[slug]/+page.svelte','utf8');
for(const evidence of ['class="inside-link"','class="guide-links"','/blog/topic/','/glossary','/how-it-works','/faq'])if(!articleTemplate.includes(evidence))fail('article template',`missing internal-link evidence: ${evidence}`);

const pageTitles=new Set(),pageDescriptions=new Set();
for(const entry of publicSitemapEntries){
	const seo=pageSEOByPath[entry.path];
	if(!seo){fail(entry.path,'sitemap page has no SEO record');continue}
	if(seo.title.length<20||seo.title.length>68)fail(entry.path,`SEO title is ${seo.title.length} characters; expected 20–68`);
	if(seo.description.length<100||seo.description.length>160)fail(entry.path,`meta description is ${seo.description.length} characters; expected 100–160`);
	if(pageTitles.has(seo.title))fail(entry.path,'duplicate public-page title');pageTitles.add(seo.title);
	if(pageDescriptions.has(seo.description))fail(entry.path,'duplicate public-page description');pageDescriptions.add(seo.description);
}
for(const path of Object.keys(pageSEOByPath)){if(!publicSitemapEntries.some(entry=>entry.path===path))fail(path,'SEO record is missing from the public sitemap registry')}
const privateRouteRoots=new Set(['admin','app','buyer','buyer-invitations','c','pay','receipt','recover','secure']);
for(const route of globSync('web/src/routes/**/+page.svelte')){
	const path=`/${route.replace(/^web\/src\/routes\//,'').replace(/(?:^|\/)\+page\.svelte$/,'')}`.replace(/\/$/,'')||'/';
	const root=path.split('/').filter(Boolean)[0]??'';
	if(path.includes('[')||privateRouteRoots.has(root))continue;
	if(!publicSitemapEntries.some(entry=>entry.path===path))fail(path,'public page is missing from the shared SEO and sitemap registry');
}
const discoveryPaths=[...publicSitemapEntries.map(entry=>entry.path),...articles.map(article=>`/blog/${article.slug}`)];
if(new Set(discoveryPaths).size!==discoveryPaths.length)fail('sitemap','duplicate public URL in discovery registry');
for(const failure of failures)process.stderr.write(`${failure}\n`);
if(failures.length){process.stderr.write(`Content audit failed with ${failures.length} issue(s).\n`);process.exit(1)}
const total=articles.reduce((sum,article)=>sum+article.wordCount,0);

const unfinishedPhrases = [
	'coming soon',
	'under construction',
	'lorem ipsum',
	'placeholder content',
	'draft for review',
	'the final terms will explain',
	'what the final notice still needs'
];
for (const route of globSync('web/src/routes/**/+page.svelte')) {
	const source = readFileSync(route, 'utf8').toLowerCase();
	for (const phrase of unfinishedPhrases) {
		if (source.includes(phrase)) fail(route, `unfinished page wording: “${phrase}”`);
	}
}

const detailedPages = [
	{
		path: 'web/src/routes/legal/privacy/+page.svelte',
		minimumWords: 1500,
		required: ['Information we collect', 'Why we use your information', 'Who may receive your information', 'How long we keep information', 'Your rights and choices', 'How we protect information', 'Questions and complaints']
	},
	{
		path: 'web/src/routes/legal/terms/+page.svelte',
		minimumWords: 2000,
		required: ['What Kredit does', 'Making a credit sale', 'Goods, delivery and problems', 'Payments, balances and Kredit fees', 'Bank-debit permission and late payment', 'Suspension, closure and records after closure', 'Help, complaints and regulators']
	}
];
for (const page of detailedPages) {
	const source = readFileSync(page.path, 'utf8');
	const main = source.match(/<main[\s\S]*?<\/main>/)?.[0] ?? '';
	const wordCount = main.replace(/<[^>]+>/g, ' ').replace(/[{}]/g, ' ').trim().split(/\s+/).filter(Boolean).length;
	if (wordCount < page.minimumWords) fail(page.path, `only ${wordCount} words; expected at least ${page.minimumWords}`);
	for (const heading of page.required) if (!source.includes(heading)) fail(page.path, `missing required section: ${heading}`);
}

if(failures.length){for(const failure of failures)process.stderr.write(`${failure}\n`);process.stderr.write(`Content audit failed with ${failures.length} issue(s).\n`);process.exit(1)}
process.stdout.write(`Content audit passed: ${articles.length} articles, ${total.toLocaleString('en-NG')} blog words, complete route copy and detailed legal pages.\n`);
