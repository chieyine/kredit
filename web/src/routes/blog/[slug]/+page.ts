import { error } from '@sveltejs/kit';
import { articleForSlug } from '$lib/blog/articles';

export function load({ params }) {
	const article = articleForSlug(params.slug);
	if (!article) error(404, 'Guide not found');
	return { article };
}
