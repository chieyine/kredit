import type { Slide } from './types';

/*
 * The investor deck, written at launch with no volume behind it yet.
 *
 * A pre-traction deck cannot borrow numbers it has not earned. So it argues
 * from three things that are true today: the problem is visible in published
 * filings, the product is built rather than described, and the founder can
 * both read the trade and ship the software.
 *
 * There is no ask slide and no milestones slide here. Those numbers are
 * settled in the room, not on a screen.
 */
export const investorDeck: Slide[] = [
	{
		kind: 'cover',
		eyebrow: 'Pre-seed · Nigeria',
		title: 'Nigerian manufacturers are lending<br /><em>₦515bn they never agreed to lend.</em>',
		lede: 'Kredit turns informal distributor credit into an agreement with a bank mandate behind it. Live from today.',
		meta: 'Kredit Technologies Limited · RC 9834452'
	},
	{
		kind: 'data',
		eyebrow: 'The problem, in their own filings',
		title: 'Receivables up 22%. Turnover flat.',
		lede: 'Ten listed Nigerian food and drinks manufacturers, Q1 2026 against Q1 2025.',
		figures: [
			{ label: 'Combined turnover', value: '₦1.78tn', note: 'Against ₦1.76tn. Flat.' },
			{ label: 'Owed by the trade', value: '₦515.3bn', note: 'From ₦423.4bn.', tone: 'alert' },
			{ label: 'Worst single case', value: '5.0×', note: 'Nestlé Nigeria: ₦5.4bn to ₦26.9bn.', tone: 'alert' }
		],
		note: 'BUA Foods’ cash from operations fell 71%. Nestlé’s halved. The profit is booked and the cash is with the distributor.',
		source: 'BusinessDay analysis of Q1 2026 filings.'
	},
	{
		kind: 'statement',
		eyebrow: 'Why it is structural',
		title: 'They cannot stop, and they cannot collect.',
		quote: {
			text: 'Credit sales are now a survival tool for the sector. It helps manufacturers keep products in circulation while giving retailers breathing room.',
			who: 'Uchenna Uzo, Faculty Director, Lagos Business School'
		},
		note: 'Tighten credit and volume falls in a market where 83% of households are already cutting back. Extend it and the receivable grows with nothing behind it. That is not a cycle. It is a permanent condition of selling into Nigerian traditional trade.',
		source: 'BusinessDay, Q1 2026; BCG household survey.'
	},
	{
		kind: 'data',
		eyebrow: 'Who ends up carrying it',
		title: 'The chain has no bank in it.',
		lede: 'Roughly nine in ten naira of Nigerian retail moves through kiosks, stalls and open markets. Those traders cannot get a facility, so credit is pushed up the chain until a manufacturer is carrying it.',
		figures: [
			{ label: 'Retail through traditional trade', value: '~90%' },
			{ label: 'MSMEs in Nigeria', value: '39.6m', note: '44% of informal businesses are in retail and trade.' },
			{ label: 'Unmet MSME credit demand', value: '$32.2bn', note: 'About ₦13tn. IFC.' }
		],
		source: 'FieldAssist; SMEDAN; IFC “Market Bite Nigeria” (2023); Moniepoint informal economy survey.'
	},
	{
		kind: 'statement',
		eyebrow: 'The wedge',
		title: 'Everyone is funding the gap. Nobody is papering it.',
		lede: 'The lending answer has been tried repeatedly in this market and keeps running into the same wall: you cannot underwrite a trader with no statements, no collateral and no credit file. Kredit does not underwrite anybody.',
		points: [
			[
				'The supplier already made the credit decision',
				'The seller knows this customer, often for years. They do not need a score. They need the agreement to hold.'
			],
			[
				'The missing piece is the instrument, not the money',
				'An accepted record, a bank mandate taken before the goods move, and a trail that survives a dispute.'
			],
			[
				'That makes us infrastructure, not a balance sheet',
				'We take no credit risk and hold no loan book. We charge for the rails.'
			]
		]
	},
	{
		kind: 'statement',
		eyebrow: 'Where we are',
		title: 'Built, and live from today.',
		lede: 'Not a prototype and not a design file. The full system is written and running: agreements with cryptographic sealing, bank mandate capture through licensed providers, a double-entry ledger in kobo, disputes with proportionate holds, settlement, reconciliation, an admin and compliance surface, and an audit trail.',
		points: [
			[
				'The usual pre-seed risk is already retired',
				'Most rounds at this stage are funding the question of whether the team can build it. That question is answered. You can open it yourself.'
			],
			[
				'It is not quick to copy',
				'Mandates, disputes, reversals and reconciliation are where this gets hard. A demo takes a weekend. None of this did.'
			],
			[
				'What you are not seeing yet',
				'No volume, and no settled cycles behind us. We are at day one, and we are not going to dress that up.'
			]
		],
		note: 'Live at kredit.ng. The working demo runs without sign-in, so judge the product rather than the description.'
	},
	{
		kind: 'steps',
		eyebrow: 'The product',
		title: 'Four steps for every sale.',
		points: [
			[
				'Connect',
				'The supplier invites their customer, who joins with their own account and confirms the business they represent.'
			],
			[
				'Agree',
				'Goods, price and dates go on record. The customer accepts, and gives the bank permission the trade requires.'
			],
			['Deliver', 'Dispatch and receipt are logged against the trade, with the evidence attached.'],
			['Settle', 'Collection runs on the due date against the mandate. Disputes hold only the contested amount.']
		]
	},
	{
		kind: 'statement',
		eyebrow: 'The model',
		title: 'We are paid out of the trade, not out of a loan.',
		points: [
			[
				'A base fee when a sale goes live',
				'Charged on the activated sale value, subject to a minimum and capped at the sale amount.'
			],
			[
				'A collection fee on money actually collected',
				'A failed debit earns nothing. Our revenue tracks money that genuinely moves.'
			],
			[
				'No monthly fee, no joining fee',
				'Nothing to negotiate at the top of the funnel, which matters when the buyer is a commercial director rather than a CTO.'
			]
		],
		note: 'Both rates are set in the platform and published live. The fee is paid by the seller and is not added to the buyer’s principal.'
	},
	{
		kind: 'statement',
		eyebrow: 'Distribution',
		title: 'One manufacturer brings its whole book.',
		lede: 'We do not acquire traders one by one. A manufacturer who adopts Kredit has a commercial reason to put its distributors on it, and each distributor has the same reason to put its retailers on it.',
		points: [
			[
				'Land upstream',
				'The manufacturer has the receivables pain, the leverage over its network, and the ability to make it a condition of credit.'
			],
			[
				'The network pulls itself down the chain',
				'A distributor already using it to pay its supplier is one step from using it to get paid by its retailers.'
			],
			['Each tier is the same product', 'Four tiers, one workspace. No separate build for each segment.']
		]
	},
	{
		kind: 'statement',
		eyebrow: 'The risks, stated plainly',
		title: 'What would have to go right.',
		points: [
			[
				'Mandate reliability',
				'Collection depends on bank debit succeeding at acceptable rates. A debit can fail, and we do not guarantee repayment. This is the core execution risk and we do not hide it.'
			],
			[
				'Manufacturer sales cycles are long',
				'A commercial director at a listed company does not sign in a week. Our answer is one distributor, one cycle, rather than a network-wide decision.'
			],
			[
				'Regulatory movement',
				'Direct debit and payment rails in Nigeria are actively regulated. We run mandates through licensed providers rather than holding the permission ourselves.'
			]
		]
	},
	{
		kind: 'cover',
		eyebrow: 'Next step',
		title: 'The product is built.<br /><em>Come and use it.</em>',
		lede: 'The demo runs without sign-in. Judge the thing itself rather than the description of it.',
		links: [
			['Try the demo', '/demo'],
			['How it works', '/how-it-works'],
			['The manufacturer deck', '/deck'],
			['Talk to us', '/contact']
		],
		meta: 'kredit.ng · hello@kredit.ng'
	}
];
