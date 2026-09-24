export type PageSEO = {
	title: string;
	description: string;
	type?: 'website' | 'article';
	published?: string;
	modified?: string;
	wordCount?: number;
	category?: string;
};

export const SITE_URL = 'https://kredit.ng';

export const publicSitemapEntries = [
	{ path: '/', priority: '1.0', changeFrequency: 'weekly' },
	{ path: '/demo', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/demo/consumer', priority: '0.7', changeFrequency: 'monthly' },
	{ path: '/how-it-works', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/manufacturers', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/distributors', priority: '0.9', changeFrequency: 'monthly' },
	{ path: '/retailers', priority: '0.8', changeFrequency: 'monthly' },
	{ path: '/consumers', priority: '0.8', changeFrequency: 'monthly' },
	{ path: '/pricing', priority: '0.8', changeFrequency: 'monthly' },
	{ path: '/security', priority: '0.7', changeFrequency: 'monthly' },
	{ path: '/contact', priority: '0.5', changeFrequency: 'monthly' },
	{ path: '/faq', priority: '0.7', changeFrequency: 'monthly' },
	{ path: '/glossary', priority: '0.6', changeFrequency: 'monthly' },
	{ path: '/blog', priority: '0.8', changeFrequency: 'weekly' },
	{ path: '/legal/complaints', priority: '0.4', changeFrequency: 'monthly' },
	{ path: '/legal/privacy', priority: '0.3', changeFrequency: 'yearly', requiresLegalApproval: true },
	{ path: '/legal/terms', priority: '0.3', changeFrequency: 'yearly', requiresLegalApproval: true }
] as const;

const defaultSEO: PageSEO = {
	title: 'Kredit — get paid for the goods you gave on credit',
	description:
		'Put your distributor credit on record, take the bank mandate before the goods move, and know what is outstanding. Built for Nigerian trade.'
};

export const pageSEOByPath: Record<string, PageSEO> = {
	'/': defaultSEO,
	'/demo': {
		title: 'Try a sample sale. No sign-in needed',
		description:
			'Play both sides of a sample sale. Accept the terms, confirm the goods, then record a payment. Nothing here is real and no money moves.'
	},
	'/demo/consumer': {
		title: 'Try a personal purchase — Kredit demo',
		description:
			'Explore a sample consumer purchase, from accepted terms through payment and delivery. No sign-in or real money.'
	},
	'/how-it-works': {
		title: 'How Kredit works, from handshake to the last naira',
		description:
			'Four steps. Write the sale down. Let your customer accept it. Confirm the goods arrived. Then follow the money until the balance is cleared.'
	},
	'/manufacturers': {
		title: 'Know what every distributor owes you — Kredit for manufacturers',
		description:
			'Every sale in writing. Proof the goods were collected. Reminders that go out without your rep chasing. And the outstanding, before it goes bad.'
	},
	'/distributors': {
		title: 'What you owe and what you are owed — Kredit for distributors',
		description:
			'Read the goods, the amount and the payment day before you agree. Confirm what arrived. Report a shortage. See every naira you have paid.'
	},
	'/retailers': {
		title: 'Keep the shop’s credit straight — Kredit for retailers',
		description:
			'Follow what you took from your distributor and what your own customers still owe you, from one workspace.'
	},
	'/consumers': {
		title: 'Know the full price before you agree — Kredit for consumers',
		description:
			'The seller, the total and the dates you are to pay, all in front of you before you accept. You do not need to register a business.'
	},
	'/pricing': {
		title: 'Kredit pricing. No monthly fee, free to start',
		description:
			'Writing a sale down is free. A base fee once the sale goes live. A collection fee only on money Kredit actually collects. No monthly charge.'
	},
	'/security': {
		title: 'Is Kredit safe? What we do to protect your money',
		description:
			'How we protect your sign-in and your private records, what your staff can and cannot see, and what happens to a link once you have shared it.'
	},
	'/faq': {
		title: 'The questions people ask — Kredit',
		description:
			'Plain answers on giving goods on credit, what it costs, what happens when a customer will not pay, and who can see your business.'
	},
	'/contact': {
		title: 'Contact Kredit — help with your account',
		description:
			'Write to us about your account, a sale that has gone wrong, or a privacy request. Our address and the complaints process are here too.'
	},
	'/glossary': {
		title: 'What the words mean on Kredit',
		description:
			'Mandate, grace period, principal, drawdown, dispute. Every term you will meet on Kredit, explained in one line without the finance talk.'
	},
	'/blog': {
		title: 'Guides for people who sell on credit — Kredit',
		description:
			'Written for Nigerian traders. How to check a new customer before you load their van, what proof to keep, and what to do when the money is late.'
	},
	'/legal/complaints': {
		title: 'Something went wrong — Kredit support and complaints',
		description:
			'Tell us what happened with a sale, a payment or your privacy. What to send us, what you must never send us, and how soon we reply.'
	},
	'/legal/privacy': {
		title: 'Privacy notice — Kredit',
		description:
			'What information Kredit keeps, why we need it, who else can see it, how long we hold it and the rights you have under Nigerian law.'
	},
	'/legal/terms': {
		title: 'Terms of service — Kredit',
		description:
			'The rules for using Kredit: your account, sales on credit, delivery, payments, fees, bank-debit permission and how to complain.'
	}
};

// Account and token-bearing flows are not public discovery pages.
export const privateRouteRoots = [
	'/account',
	'/start',
	'/signin',
	'/workspace',
	'/personal',
	'/admin',
	'/agents',
	'/c',
	'/pay',
	'/receipt',
	'/secure',
	'/recover',
	'/buyer-invitations'
];
export function isPrivateRoute(pathname: string): boolean {
	return privateRouteRoots.some((root) => pathname === root || pathname.startsWith(root + '/'));
}

// These public links are deliberately shared directly, not advertised in search.
export const unlistedRouteRoots = ['/deck', '/join'];
export function isUnlistedRoute(pathname: string): boolean {
	return unlistedRouteRoots.some((root) => pathname === root || pathname.startsWith(root + '/'));
}

export const nonIndexablePaths = new Set(['/legal/privacy', '/legal/terms', '/deck']);

export function seoForPath(pathname: string): PageSEO {
	const normalized = pathname.length > 1 ? pathname.replace(/\/$/, '') : pathname;
	return pageSEOByPath[normalized] ?? defaultSEO;
}

export function jsonLd(value: unknown) {
	return JSON.stringify(value).replaceAll('<', '\\u003c');
}
