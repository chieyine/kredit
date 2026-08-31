import { articleCategories, articleCategoryDetails, articles } from '$lib/blog/articles';

import { publicSitemapEntries, SITE_URL } from '$lib/seo';
import { loadLegalConfig } from '$lib/server/legal-config';

export function GET() {
	const legalActive = loadLegalConfig().active;
	const fixed = publicSitemapEntries.filter((entry) => !('requiresLegalApproval' in entry) || legalActive);
	const urls = [
		...fixed.map(({ path, priority, changeFrequency })=>`<url><loc>${SITE_URL}${path}</loc><lastmod>2026-08-31</lastmod><changefreq>${changeFrequency}</changefreq><priority>${priority}</priority></url>`),
		...articleCategories.map(category=>`<url><loc>${SITE_URL}/blog/topic/${articleCategoryDetails[category].slug}</loc><lastmod>2026-08-31</lastmod><changefreq>monthly</changefreq><priority>0.7</priority></url>`),
		...articles.map(article=>`<url><loc>${SITE_URL}/blog/${article.slug}</loc><lastmod>${article.modified}</lastmod><changefreq>monthly</changefreq><priority>0.7</priority></url>`)
	].join('');
	return new Response(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls}</urlset>`,{headers:{'Content-Type':'application/xml; charset=utf-8','Cache-Control':'public, max-age=3600'}});
}
