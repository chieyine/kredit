#!/usr/bin/env node

import { globSync, readFileSync } from 'node:fs';
import { articleCategories, articleCategoryDetails, articles, categorySlug } from '../web/src/lib/blog/articles.ts';
import { pageSEOByPath, publicSitemapEntries } from '../web/src/lib/seo.ts';

const failures=[];
const fail=(slug,message)=>failures.push(`${slug}: ${message}`);
if(!articles.length)fail('blog','no published guides');
const slugs=new Set(),titles=new Set(),paragraphs=new Map();
for(const article of articles){
	if(slugs.has(article.slug))fail(article.slug,'duplicate slug');slugs.add(article.slug);
	if(titles.has(article.title))fail(article.slug,'duplicate title');titles.add(article.title);
	if(!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(article.slug))fail(article.slug,'slug is not search-friendly');
	if(article.description.length<40||article.description.length>200)fail(article.slug,`meta description is ${article.description.length} characters; expected 40–200`);
	if(article.title.length<15||article.title.length>100)fail(article.slug,`title is ${article.title.length} characters; expected 15–100`);
	if(article.related.length<3)fail(article.slug,'fewer than 3 internal links');
	for(const date of [article.modified,article.published].filter(Boolean)){if(!/^\d{4}-\d{2}-\d{2}$/.test(date)||Number.isNaN(Date.parse(date))||date>new Date().toISOString().slice(0,10))fail(article.slug,'invalid or future editorial date');}
	if(article.published && article.modified<article.published)fail(article.slug,'modified date is earlier than published date');
	for(const section of article.sections){
		for(const paragraph of section.paragraphs){
			const earlierSlug=paragraphs.get(paragraph);
			if(earlierSlug)fail(article.slug,`reuses a long-form paragraph from ${earlierSlug}`);
			else paragraphs.set(paragraph,article.slug);
		}
	}
}
// Detect shared passages even when a copied paragraph has new topic text appended.
const passages = [];
for (const article of articles) {
 for (const paragraph of article.sections.flatMap(section => section.paragraphs)) {
  const tokens = paragraph.toLowerCase().replace(/[^\p{L}\p{N}]+/gu, ' ').trim().split(/\s+/);
  const spans = new Set(tokens.slice(0, Math.max(0, tokens.length - 11)).map((_, index) => tokens.slice(index, index + 12).join(' ')));
  for (const previous of passages) {
   if (previous.slug === article.slug) continue;
   const shared = [...spans].filter(span => previous.spans.has(span)).length;
   if (shared >= 8 && shared / Math.min(spans.size, previous.spans.size) >= 0.6) fail(article.slug, `reuses a substantial passage from ${previous.slug}`);
  }
  passages.push({ slug: article.slug, spans });
 }
}
// Compare sentences as well as paragraphs: adding a topic suffix must not hide reused prose.
const sentences = new Map();
for (const article of articles) {
 const text = [article.intro, ...article.sections.flatMap(section => [section.heading, ...section.paragraphs, ...(section.points ?? [])]), ...article.faq.flatMap(item => [item.question, item.answer])].join(' ');
 const count = text.trim().split(/\s+/).filter(Boolean).length;
 if (count !== article.wordCount || article.readingMinutes !== Math.max(1, Math.ceil(count / 220))) fail(article.slug, 'incorrect reading metadata');
 if (!article.intro.trim() || !article.sections.length || article.sections.some(section => !section.heading.trim() || !section.paragraphs.length)) fail(article.slug, 'unfinished guide');
 for (const sentence of text.split(/[.!?](?:\s|$)/)) {
  const normalized = sentence.toLowerCase().replace(/[^\p{L}\p{N}]+/gu, ' ').trim();
  if (normalized.split(' ').length < 18) continue;
  const previous = sentences.get(normalized);
  if (previous && previous !== article.slug) fail(article.slug, `repeated sentence from ${previous}`);
  sentences.set(normalized, article.slug);
 }
 for (const source of article.sources) if (!source.note.trim() || !source.url.startsWith('https://')) fail(article.slug, 'source needs a useful note and secure URL');
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
			inboundLinks.set(target.slug,(inboundLinks.get(target.slug)??0)+1);
		}
	}
}
for(const [slug,count] of inboundLinks){if(count<1)fail(slug,`only ${count} inbound article links; expected at least 1`)}
for(const category of articleCategories){
	const details=articleCategoryDetails[category];
	if(details.slug!==categorySlug(category))fail(category,'topic hub slug does not match its category');
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
		required: ['Information we collect', 'Why we use your information', 'Who may receive your information', 'How long we keep information', 'Your rights and choices', 'How we protect information', 'Questions and complaints']
	},
	{
		path: 'web/src/routes/legal/terms/+page.svelte',
		required: ['What Kredit does', 'Making a credit sale', 'Goods, delivery and problems', 'Payments, balances and Kredit fees', 'Bank-debit permission and late payment', 'Suspension, closure and records after closure', 'Help, complaints and regulators']
	}
];
for (const page of detailedPages) {
	const source = readFileSync(page.path, 'utf8');
	for (const heading of page.required) if (!source.includes(heading)) fail(page.path, `missing required section: ${heading}`);
}

if(failures.length){for(const failure of failures)process.stderr.write(`${failure}\n`);process.stderr.write(`Content audit failed with ${failures.length} issue(s).\n`);process.exit(1)}
process.stdout.write(`Content audit passed: ${articles.length} articles, ${total.toLocaleString('en-NG')} blog words, route discovery and required legal sections checked.\n`);
