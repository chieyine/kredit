import { guideDrafts } from './guides.js';

export type ArticleCategory = 'Credit sales' | 'Customer checks' | 'Agreements' | 'Payments' | 'Late payment' | 'Cash flow' | 'Business records' | 'Safe payments' | 'Industry guides' | 'Business growth';

export type ArticleSection = { heading: string; paragraphs: string[]; points?: string[] };
export type ArticleSource = { name: string; url: string; note: string };
export type Article = {
	slug: string; title: string; description: string; category: ArticleCategory; keyphrase: string;
	published?: string; modified: string; readingMinutes: number; wordCount: number; intro: string;
	sections: ArticleSection[]; faq: { question: string; answer: string }[]; sources: ArticleSource[];
	related: { slug: string; title: string }[];
};

const words = (text: string) => text.trim().split(/\s+/).filter(Boolean).length;
export const articles: Article[] = guideDrafts.map(draft => {
 const { relatedSlugs, ...content } = draft;
 const wordCount = words([draft.intro, ...draft.sections.flatMap(section => [section.heading, ...section.paragraphs, ...('points' in section ? section.points ?? [] : [])]), ...draft.faq.flatMap(item => [item.question, item.answer])].join(' '));
 return { ...content, category: draft.category as ArticleCategory, keyphrase: draft.title, modified: '2026-09-04', wordCount, readingMinutes: Math.max(1, Math.ceil(wordCount / 220)), related: relatedSlugs.map(slug => {
  const target = guideDrafts.find(candidate => candidate.slug === slug);
  if (!target) throw new Error(`Missing related guide: ${slug}`);
  return { slug, title: target.title };
 }) };
});
export const articleCategories = [...new Set(articles.map(article => article.category))];
export const articlesBySlug = new Map(articles.map(article => [article.slug, article]));
export function articleForSlug(slug: string) { return articlesBySlug.get(slug); }

export function categorySlug(category: ArticleCategory) {
	return category.toLowerCase().replaceAll(' ', '-');
}

export const articleCategoryDetails: Record<ArticleCategory, { slug: string; title: string; description: string }> = {
	'Credit sales': { slug: 'credit-sales', title: 'Credit sales guides', description: 'Learn how to choose customers, set a safe limit and agree clear payment terms before goods leave your business.' },
	'Customer checks': { slug: 'customer-checks', title: 'Customer check guides', description: 'Twelve questions to discuss with a customer before agreeing a credit order, from delivery details to the source of repayment.' },
	'Agreements': { slug: 'agreements', title: 'Credit agreement guides', description: 'Learn what to write down, how customers accept a sale and which delivery and payment records both sides should keep.' },
	'Payments': { slug: 'payments', title: 'Customer payment guides', description: 'Learn how to confirm transfers, record cash and part-payments, issue receipts and keep every customer balance correct.' },
	'Late payment': { slug: 'late-payment', title: 'Late payment guides', description: 'Payment reminder examples for before the due date, on the day and after a missed payment, with room for the customer to respond.' },
	'Cash flow': { slug: 'cash-flow', title: 'Cash-flow guides', description: 'Learn how unpaid sales affect stock and bills, and how to plan around the money customers still owe your business.' },
	'Business records': { slug: 'business-records', title: 'Business record guides', description: 'Decide what belongs in a customer record, control staff access and prepare for a lost phone or a change in your team.' },
	'Safe payments': { slug: 'safe-payments', title: 'Safe payment guides', description: 'Check incoming money in your own account and handle transfer screenshots, delayed payments and suspicious refund requests.' },
	'Industry guides': { slug: 'industry-guides', title: 'Credit guides for different businesses', description: 'A building-material sale involves the buyer, the delivery site and whoever signs for the goods. Keep those responsibilities clear.' },
	'Business growth': { slug: 'business-growth', title: 'Business growth guides', description: 'Write a credit policy your team can use: who approves an order, when supply pauses and how exceptions are recorded.' }
};

export function categoryForSlug(slug: string) {
	return articleCategories.find(category => articleCategoryDetails[category].slug === slug);
}
