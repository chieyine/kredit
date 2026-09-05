import { expect, test } from '@playwright/test';
import { draftGuideSlugs, validateArticleSlugs } from '../src/lib/blog/articles';

const expectedDraftSlugs = [
	'net-7-net-14-net-30-payment-terms',
	'invoice-vs-receipt-vs-statement'
];

test('article slug validation rejects malformed and duplicate slugs', () => {
	expect(() => validateArticleSlugs(['valid-article', 'valid-article'])).toThrow(/Duplicate article slug/);
	expect(() => validateArticleSlugs(['Invalid Article'])).toThrow(/Invalid article slug/);
	expect(() => validateArticleSlugs(['valid-article', 'another-valid-article'])).not.toThrow();
});

test('first-wave editorial drafts stay out of public routes, sitemap and RSS', async ({ request }) => {
	expect([...draftGuideSlugs].sort()).toEqual([...expectedDraftSlugs].sort());

	const sitemap = await (await request.get('/sitemap.xml')).text();
	const rss = await (await request.get('/blog/rss.xml')).text();

	for (const slug of expectedDraftSlugs) {
		const response = await request.get(`/blog/${slug}`);
		expect(response.status(), `${slug} must remain private while draft=true`).toBe(404);
		expect(sitemap).not.toContain(`/blog/${slug}`);
		expect(rss).not.toContain(`/blog/${slug}`);
	}
});
