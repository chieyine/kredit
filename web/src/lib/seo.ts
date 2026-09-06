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
	title: 'Kredit — give goods now, get your money later',
	description: 'Write the sale down, let your customer accept it, keep the delivery proof and follow every payment. Built for Nigerian businesses. Free to start.'
};

export const pageSEOByPath: Record<string, PageSEO> = {
	'/': defaultSEO,
	'/demo': {
		title: 'Try Kredit free in 60 seconds — no sign-in',
		description: 'Play both sides of a sample sale. Accept the deal, confirm the goods and enter a payment. No sign-in, no real money, about 60 seconds.'
	},
	'/how-it-works': {
		title: 'How Kredit works — from handshake to last naira',
		description: 'Four steps: write the sale down, let your customer accept it, confirm the goods arrived, then follow every payment in one clear record.'
	},
	'/for-suppliers': {
		title: 'Sell on credit and stay in control — Kredit for sellers',
		description: 'Put every sale in writing, keep proof the goods arrived, let Kredit send the reminders and always know what each customer still owes you.'
	},
	'/for-buyers': {
		title: 'Know exactly what you owe — Kredit for customers',
		description: 'Read the goods, the money and the payment day before you say yes. Confirm delivery, report a problem and see every payment you have made.'
	},
	'/pricing': {
		title: 'Kredit pricing — no monthly fee, free to start',
		description: 'Writing a sale down is free. A small base fee only when the sale starts, and a collection fee only on money Kredit actually collects for you.'
	},
	'/security': {
		title: 'Is Kredit safe? How we protect your money',
		description: 'How Kredit protects your sign-in, your private business records, every payment, your staff permissions and the links you share with customers.'
	},
	'/faq': {
		title: 'Common questions — Kredit',
		description: 'Straight answers about selling goods on credit, getting paid, what Kredit costs, late customers, your privacy and keeping your account safe.'
	},
	'/glossary': {
		title: 'What the words mean — Kredit glossary',
		description: 'Plain meanings for the business words you will see on Kredit: mandate, grace period, principal, dispute, drawdown and more. No big grammar.'
	},
	'/blog': {
		title: 'Guides for selling on credit — Kredit',
		description: 'Simple guides for Nigerian businesses: selling on credit, checking a new customer, keeping delivery proof and what to do when payment is late.'
	},
	'/legal/complaints': {
		title: 'Get help — Kredit support and complaints',
		description: 'Tell us what went wrong with a sale, a payment, your privacy or the app. See what to send us, what never to send and how we reply.'
	},
	'/legal/privacy': {
		title: 'Privacy notice — Kredit',
		description: 'What information Kredit keeps, why we need it, who else can see it, how long we hold it and the rights you have under Nigerian law.'
	},
	'/legal/terms': {
		title: 'Terms of service — Kredit',
		description: 'The rules for using Kredit: your account, sales on credit, delivery, payments, fees, bank-debit permission and how to complain.'
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
