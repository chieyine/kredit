export type PageSEO = {
	title: string;
	description: string;
	type?: 'website' | 'article';
	published?: string;
};

const defaultSEO: PageSEO = {
	title: 'Kredit — sell goods on credit and get paid',
	description: 'Write down the deal, get your customer to accept it, keep proof of delivery and track every payment.'
};

const pages: Record<string, PageSEO> = {
	'/': defaultSEO,
	'/how-it-works': {
		title: 'How Kredit works — from credit sale to final payment',
		description: 'Write down the deal, let your customer accept it, confirm delivery and track every payment in one place.'
	},
	'/for-suppliers': {
		title: 'Sell goods on credit with less chasing — Kredit',
		description: 'Put every credit sale in writing, keep delivery proof and see what each customer still owes.'
	},
	'/for-buyers': {
		title: 'Know what you owe and why — Kredit for buyers',
		description: 'Check the full deal before accepting, confirm delivery, report problems and see every payment.'
	},
	'/pricing': {
		title: 'Pricing — pay only when you get paid — Kredit',
		description: 'Pay 0.5% when your customer pays or 1% when Kredit collects. No monthly fee, setup fee or hidden charge.'
	},
	'/security': {
		title: 'How Kredit protects your money and records',
		description: 'See how Kredit protects sign-in, private information, payment records and shared links.'
	},
	'/faq': {
		title: 'Frequently asked questions — Kredit',
		description: 'Simple answers about selling on credit, customer payments, fees, late payment, privacy and safety.'
	},
	'/glossary': {
		title: 'Trade credit glossary — Kredit',
		description: 'Simple meanings for finance words you may see when selling or buying goods on credit.'
	},
	'/blog': {
		title: 'Helpful guides for selling on credit — Kredit',
		description: 'Short, simple guides about clear credit sales, delivery proof, late payment and customer relationships.'
	},
	'/blog/sell-on-credit-safely': {
		title: 'How to sell on credit and protect your money — Kredit',
		description: 'Five things to agree before goods leave your shop: the amount, payment date, extra time, delivery proof and late-payment steps.',
		type: 'article', published: '2026-08-10'
	},
	'/blog/why-credit-agreements-fail': {
		title: 'Why credit sales turn into arguments — Kredit',
		description: 'How a clear record of the goods, date, amount and delivery can prevent payment arguments.',
		type: 'article', published: '2026-08-04'
	},
	'/blog/collections-last-resort': {
		title: 'What to do when a customer pays late — Kredit',
		description: 'A fair order for late payments: reminders, extra time, recording direct payments and automatic debit as the last step.',
		type: 'article', published: '2026-07-28'
	},
	'/blog/trade-credit-vs-loan': {
		title: 'Goods on credit or a loan app? — Kredit',
		description: 'A simple look at two ways a customer can buy business stock now and pay later.',
		type: 'article', published: '2026-07-20'
	},
	'/legal/complaints': {
		title: 'Complaints and support — Kredit',
		description: 'Report a Kredit service, payment, privacy or accessibility concern and understand how evidence, updates and escalation are handled.'
	},
	'/legal/privacy': {
		title: 'Privacy notice — Kredit',
		description: 'Pre-launch information about Kredit privacy controls, data rights, retention and provider disclosures.'
	},
	'/legal/terms': {
		title: 'Terms of service — Kredit',
		description: 'Pre-launch information about the Kredit service terms and supplier-funded trade-credit model.'
	}
};

export const nonIndexablePaths = new Set(['/legal/privacy', '/legal/terms']);

export function seoForPath(pathname: string): PageSEO {
	const normalized = pathname.length > 1 ? pathname.replace(/\/$/, '') : pathname;
	return pages[normalized] ?? defaultSEO;
}

export function jsonLd(value: unknown) {
	return JSON.stringify(value).replaceAll('<', '\\u003c');
}
