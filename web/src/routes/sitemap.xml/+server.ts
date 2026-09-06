import { articleCategories, articleCategoryDetails, articles, type ArticleCategory } from '$lib/blog/articles';

import { publicSitemapEntries, SITE_URL } from '$lib/seo';
import { loadLegalConfig } from '$lib/server/legal-config';

function topicLastmod(category: ArticleCategory) {
	return articles
		.filter((article) => article.category === category)
		.map((article) => article.modified)
		.sort()
		.at(-1);
}

export function GET() {
	const legalActive = loadLegalConfig().active;
	const fixed = publicSitemapEntries.filter((entry) => !('requiresLegalApproval' in entry) || legalActive);
	const urls = [
		...fixed.map(({ path, priority, changeFrequency }) => `<url><loc>${SITE_URL}${path}</loc><changefreq>${changeFrequency}</changefreq><priority>${priority}</priority></url>`),
		...articleCategories.map((category) => {
			const lastmod = topicLastmod(category);
			return `<url><loc>${SITE_URL}/blog/topic/${articleCategoryDetails[category].slug}</loc>${lastmod ? `<lastmod>${lastmod}</lastmod>` : ''}<changefreq>monthly</changefreq><priority>0.7</priority></url>`;
		}),
		...articles.map((article) => `<url><loc>${SITE_URL}/blog/${article.slug}</loc><lastmod>${article.modified}</lastmod><changefreq>monthly</changefreq><priority>0.7</priority></url>`)
	].join('');
	return new Response(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${urls}</urlset>`, { headers: { 'Content-Type': 'application/xml; charset=utf-8', 'Cache-Control': 'public, max-age=3600' } });
}
