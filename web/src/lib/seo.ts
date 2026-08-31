export type PageSEO = {
	title: string;
	description: string;
	type?: 'website' | 'article';
	published?: string;
	modified?: string;
	wordCount?: number;
	category?: string;
};

export const SITE_URL = 'https://kredit.com.ng';

export const publicSitemapEntries = [
	{ path: '/', priority: '1.0', changeFrequency: 'weekly' },
	{ path: '/demo', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/how-it-works', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/for-suppliers', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/for-buyers', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/pricing', priority: '0.8', changeFrequency: 'monthly' },
	{ path: '/security', priority: '0.7', changeFrequency: 'monthly' },
	{ path: '/faq', priority: '0.7', changeFrequency: 'monthly' },
	{ path: '/glossary', priority: '0.6', changeFrequency: 'monthly' },
	{ path: '/blog', priority: '0.8', changeFrequency: 'weekly' },
	{ path: '/legal/complaints', priority: '0.4', changeFrequency: 'monthly' },
	{ path: '/legal/privacy', priority: '0.3', changeFrequency: 'yearly', requiresLegalApproval: true },
	{ path: '/legal/terms', priority: '0.3', changeFrequency: 'yearly', requiresLegalApproval: true }
] as const;

const defaultSEO: PageSEO = {
	title: 'Kredit — sell goods on credit and get paid',
	description: 'Write down the deal, get your customer to accept it, keep proof of delivery and track every payment.'
};

export const pageSEOByPath: Record<string, PageSEO> = {
	'/': defaultSEO,
	'/demo': {
		title: 'Try Kredit in 60 seconds — interactive credit-sale demo',
		description: 'Try a complete sample credit sale as the seller and customer. Accept the deal, confirm the goods and record a payment without signing in.'
	},
	'/how-it-works': {
		title: 'How Kredit works — from credit sale to final payment',
		description: 'See how to write down a credit sale, let your customer accept it, confirm delivery and track every payment in one clear place.'
	},
	'/for-suppliers': {
		title: 'Sell goods on credit with less chasing — Kredit',
		description: 'Put every credit sale in writing, keep proof that the goods arrived, send simple reminders and see what each customer still owes.'
	},
	'/for-buyers': {
		title: 'Know what you owe and why — Kredit for buyers',
		description: 'Check the goods, amount and payment day before you accept a sale. Confirm delivery, report a problem and see every payment clearly.'
	},
	'/pricing': {
		title: 'Pricing — pay only when you get paid — Kredit',
		description: 'Pay 0.5% when your customer pays or 1% when Kredit collects. No monthly fee, setup fee or hidden charge.'
	},
	'/security': {
		title: 'How Kredit protects your money and records',
		description: 'See how Kredit protects account sign-in, private business information, payment records, staff access and links shared with customers.'
	},
	'/faq': {
		title: 'Frequently asked questions — Kredit',
		description: 'Get simple answers about selling goods on credit, customer payments, Kredit fees, late payment, private information and account safety.'
	},
	'/glossary': {
		title: 'Trade credit glossary — Kredit',
		description: 'Understand the simple meaning of words used for credit sales, customer payments, delivery, late payment and money collection in Nigeria.'
	},
	'/blog': {
		title: 'Helpful guides for selling on credit — Kredit',
		description: 'Read simple guides for Nigerian businesses about credit sales, customer checks, delivery proof, payment records and late payment.'
	},
	'/legal/complaints': {
		title: 'Complaints and support — Kredit',
		description: 'Report a Kredit service, payment, privacy or accessibility concern and understand how evidence, updates and escalation are handled.'
	},
	'/legal/privacy': {
		title: 'Privacy notice — Kredit',
		description: 'Read how Kredit collects, uses, shares, protects and removes information, and how to use your privacy rights in Nigeria.'
	},
	'/legal/terms': {
		title: 'Terms of service — Kredit',
		description: 'Read the clear rules for Kredit accounts, credit sales, delivery, payments, fees, bank-debit permission and complaints.'
	}
};

export const nonIndexablePaths = new Set(['/legal/privacy', '/legal/terms']);

export function seoForPath(pathname: string): PageSEO {
	const normalized = pathname.length > 1 ? pathname.replace(/\/$/, '') : pathname;
	return pageSEOByPath[normalized] ?? defaultSEO;
}

export function jsonLd(value: unknown) {
	return JSON.stringify(value).replaceAll('<', '\\u003c');
}
