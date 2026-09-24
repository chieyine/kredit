import type { Slide } from './types';

/*
 * The investor deck, written at launch with no volume behind it yet.
 *
 * A pre-traction deck cannot borrow numbers it has not earned. So it argues
 * from three things that are true now: the problem is visible in published
 * filings, the product is built rather than described, and the model takes
 * no credit risk.
 *
 * There is no ask slide and no milestones slide here. Those numbers are
 * settled in the room, not on a screen.
 */
export const investorDeck: Slide[] = [
	{
		kind: 'cover',
		record: true,
		eyebrow: 'Pre-seed · Nigeria',
		title: 'Ten Nigerian manufacturers are owed ₦515bn<br /><em>by their own distributors.</em>',
		lede: 'Kredit turns that informal trade credit into signed agreements with a bank permission behind them. The product is built and live.',
		meta: 'Kredit Technologies Limited · RC 9834452'
	},
	{
		kind: 'figures',
		eyebrow: 'The problem, in their own filings',
		title: 'Sales flat. Money owed by the trade up 22%.',
		lede: 'Ten listed Nigerian food and drinks manufacturers, first quarter 2026 against the same quarter of 2025.',
		figures: [
			{ label: 'Combined turnover', value: '₦1.78tn', note: '₦1.76tn a year before.' },
			{ label: 'Owed by the trade', value: '₦515.3bn', note: 'Up from ₦423.4bn.', tone: 'alert' },
			{ label: 'Cash from operations', value: '−71%', note: 'BUA Foods. Nestlé Nigeria’s halved.', tone: 'alert' }
		],
		source: 'BusinessDay analysis of Q1 2026 filings.'
	},
	{
		kind: 'bars',
		eyebrow: 'Owed by the trade, a year apart',
		title: 'The profit is booked. The cash is with the distributor.',
		bars: [
			{ label: 'NASCON Allied', before: 17.8, after: 60.2 },
			{ label: 'BUA Foods', before: 20.4, after: 85.1 },
			{ label: 'Nestlé Nigeria', before: 5.4, after: 26.9 }
		],
		barLabels: ['Q1 2025', 'Q1 2026'],
		source: 'BusinessDay analysis of Q1 2026 filings. Naira billions.'
	},
	{
		kind: 'quote',
		eyebrow: 'Why it lasts',
		title: 'They cannot stop giving credit, and they cannot collect it.',
		quote: {
			text: 'Credit sales are now a survival tool for the sector. It helps manufacturers keep products in circulation while giving retailers breathing room.',
			who: 'Uchenna Uzo, Faculty Director, Lagos Business School'
		},
		note: 'Tighten credit and volume falls, in a market where 83% of households are already cutting back. Extend it and the receivable grows with nothing behind it. This is a lasting condition of selling into Nigerian traditional trade, not a bad quarter.',
		source: 'BusinessDay, Q1 2026; BCG household survey.'
	},
	{
		kind: 'chain',
		eyebrow: 'Who ends up carrying it',
		title: 'The chain has no bank in it.',
		tiers: [
			['Manufacturer', 'Carries the credit in the end, as trade debtors.'],
			['Distributor', 'Takes stock on credit and passes credit down.'],
			['Retailer', 'Kiosks, stalls, open markets. No facility.'],
			['Customer', 'Buys from the shop, often on credit.']
		],
		figures: [
			{ label: 'Retail through traditional trade', value: '9 in 10', note: 'Naira of Nigerian retail.' },
			{ label: 'Small businesses in Nigeria', value: '39.6m', note: '44% of informal ones are in retail and trade.' },
			{ label: 'Unmet small-business credit', value: '$32.2bn', note: 'About ₦13tn.' }
		],
		source: 'FieldAssist; SMEDAN; IFC “Market Bite Nigeria” (2023); Moniepoint informal economy survey.'
	},
	{
		kind: 'compare',
		eyebrow: 'The wedge',
		title: 'Others try to fund the gap. We make the existing credit hold.',
		compare: [
			{
				heading: 'Lending to traders',
				items: [
					'Needs statements, collateral and a credit file the trader does not have.',
					'The lender carries the credit risk and needs a balance sheet to grow.',
					'Adds a new creditor to a relationship that already works.'
				]
			},
			{
				heading: 'Kredit',
				items: [
					'The supplier already knows the customer and has made the credit decision.',
					'We add the agreement, the bank permission and a record that survives a dispute.',
					'No loan book and no credit risk. We are paid for the rails.'
				]
			}
		]
	},
	{
		kind: 'split',
		eyebrow: 'Where we are',
		title: 'Built and live now.',
		lede: 'The whole system is written and running: sealed agreements, bank permission through licensed providers, a double-entry ledger in kobo, disputes that hold only the contested amount, settlement, reconciliation, admin controls and an audit trail.',
		points: [
			[
				'The build risk is already retired',
				'Most rounds at this stage pay to find out if the team can build it. You can open it yourself.'
			],
			[
				'It is not quick to copy',
				'Mandates, disputes, reversals and reconciliation are where this gets hard. A demo takes a weekend; this did not.'
			],
			['What is not here yet', 'Volume. We are at the start, and we will not dress that up.']
		],
		note: 'Live at kredit.ng. The demo runs without signing in.'
	},
	{
		kind: 'steps',
		eyebrow: 'The product',
		title: 'Four steps for every sale.',
		points: [
			['Connect', 'The supplier invites a customer, who joins with their own account and confirms their business.'],
			['Agree', 'Goods, price and dates go on record. The customer accepts and gives the bank permission.'],
			['Deliver', 'Dispatch and receipt are recorded against the sale, with the evidence attached.'],
			['Settle', 'Collection runs on the agreed dates. A dispute holds only the amount in question.']
		]
	},
	{
		kind: 'split',
		eyebrow: 'The model',
		title: 'We are paid out of the trade, not out of a loan.',
		points: [
			['A base fee when a sale goes live', 'On the value of the sale, with a minimum, never more than the sale.'],
			[
				'A collection fee on money actually collected',
				'A failed debit earns nothing, so revenue follows money that really moves.'
			],
			[
				'No monthly fee, no joining fee',
				'Nothing to negotiate before the first sale, which matters when the buyer is a commercial director.'
			]
		],
		note: 'Rates are published on the site. The seller pays; it is never added to what the buyer owes.'
	},
	{
		kind: 'steps',
		eyebrow: 'Distribution',
		title: 'One manufacturer brings its whole network.',
		lede: 'We do not win traders one at a time. Each tier has a commercial reason to bring on the tier below it.',
		points: [
			[
				'Land the manufacturer',
				'It has the receivables problem and the leverage to make Kredit a condition of credit.'
			],
			[
				'Its distributors follow',
				'A distributor paying its supplier on Kredit is one step from getting paid by its shops on it.'
			],
			['Then their retailers', 'Every tier uses the same product. There is no separate build for each segment.']
		]
	},
	{
		kind: 'split',
		eyebrow: 'The risks, stated plainly',
		title: 'What has to go right.',
		points: [
			[
				'Bank debits must succeed often enough',
				'A debit can fail, and we do not guarantee repayment. This is the core execution risk.'
			],
			[
				'Manufacturers buy slowly',
				'A listed company does not sign in a week. We ask for one distributor and one cycle, not the whole network.'
			],
			[
				'Payment rules can change',
				'Direct debit in Nigeria is actively regulated. We work through licensed providers rather than holding permissions ourselves.'
			]
		]
	},
	{
		kind: 'cover',
		eyebrow: 'Next step',
		title: 'The product is built.<br /><em>Come and use it.</em>',
		lede: 'The demo runs without signing in. Judge the thing itself, not the description of it.',
		links: [
			['Try the demo', '/demo'],
			['How it works', '/how-it-works'],
			['The manufacturer deck', '/deck'],
			['Talk to us', '/contact']
		],
		meta: 'kredit.ng · hello@kredit.ng'
	}
];
