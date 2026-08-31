import { error } from '@sveltejs/kit';
import { articleCategoryDetails, articles, categoryForSlug } from '$lib/blog/articles';

export function load({ params }) {
	const category = categoryForSlug(params.topic);
	if (!category) error(404, 'Guide topic not found');
	const details = articleCategoryDetails[category];
	return {
		category,
		details,
		articles: articles.filter(article => article.category === category),
		seo: {
			title: `${details.title} for Nigerian businesses — Kredit`,
			description: details.description
		}
	};
}
