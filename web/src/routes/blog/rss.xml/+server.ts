import { articles } from '$lib/blog/articles';
import { SITE_URL } from '$lib/seo';

const escape = (value:string) => value.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;');
export function GET(){
	const newest = [...articles].sort((a, b) => b.published.localeCompare(a.published));
	const items=newest.slice(0,50).map(article=>`<item><title>${escape(article.title)}</title><link>${SITE_URL}/blog/${article.slug}</link><guid isPermaLink="true">${SITE_URL}/blog/${article.slug}</guid><description>${escape(article.description)}</description><category>${escape(article.category)}</category><pubDate>${new Date(`${article.published}T08:00:00+01:00`).toUTCString()}</pubDate></item>`).join('');
	const lastBuildDate = new Date(`${newest[0]?.modified ?? '2026-08-31'}T08:00:00+01:00`).toUTCString();
	return new Response(`<?xml version="1.0" encoding="UTF-8"?><rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom"><channel><title>Kredit helpful guides</title><link>${SITE_URL}/blog</link><atom:link href="${SITE_URL}/blog/rss.xml" rel="self" type="application/rss+xml"/><description>Simple guides for Nigerian businesses that sell goods on credit.</description><language>en-NG</language><lastBuildDate>${lastBuildDate}</lastBuildDate><generator>Kredit</generator>${items}</channel></rss>`,{headers:{'Content-Type':'application/rss+xml; charset=utf-8','Cache-Control':'public, max-age=3600'}});
}
