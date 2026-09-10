import {loadGuides} from '$lib/server/guides';
import type {PageServerLoad} from './$types';
import { error } from '@sveltejs/kit';
import { articleCategoryDetails, categoryForSlug } from '$lib/blog/articles';

export const load:PageServerLoad=async({params,fetch,setHeaders})=>{
 setHeaders({"cache-control":"no-store"});
 const articles=await loadGuides(fetch);
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
