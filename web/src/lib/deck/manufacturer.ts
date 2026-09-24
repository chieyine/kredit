import type { Slide } from './types';

/*
 * The manufacturer deck.
 *
 * Written for a commercial director or finance director at a manufacturer.
 * Plain Nigerian commercial English: trade debtors, turnover, cash, recovery.
 * One point per slide, said once.
 *
 * Every figure is published and cited on the slide that carries it. Nothing
 * here describes Kredit's own volumes or customers. Where the product has a
 * limit, the slide states it in the words the product already uses.
 */
export const manufacturerDeck: Slide[] = [
	{
		kind: 'cover',
		record: true,
		eyebrow: 'For manufacturers',
		title: 'Your distributors owe you more every quarter.<br /><em>Your cash is not keeping up.</em>',
		lede: 'Kredit puts a signed agreement, a bank permission and one shared record behind the credit you already give your distributors.',
		meta: 'Kredit Technologies Limited · RC 9834452 · Maiduguri, Nigeria'
	},
	{
		kind: 'figures',
		eyebrow: 'First quarter 2026',
		title: 'Sales flat. Money owed by the trade up 22%.',
		lede: 'Ten listed Nigerian food and drinks manufacturers, first quarter 2026 against the same quarter of 2025.',
		figures: [
			{ label: 'Combined turnover', value: '₦1.78tn', note: '₦1.76tn a year before.' },
			{ label: 'Owed by the trade', value: '₦515.3bn', note: 'Up from ₦423.4bn.', tone: 'alert' },
			{ label: 'Combined profit', value: '₦307.5bn', note: 'Up 19.6%, on paper.' }
		],
		source:
			'BusinessDay analysis of Q1 2026 filings: BUA Foods, Nestlé Nigeria, Unilever Nigeria, NASCON Allied, Dangote Sugar, Nigerian Breweries, International Breweries, Guinness Nigeria, Cadbury Nigeria, Champion Breweries.'
	},
	{
		kind: 'bars',
		eyebrow: 'Owed by the trade, a year apart',
		title: 'For three of them it more than tripled.',
		bars: [
			{ label: 'NASCON Allied', before: 17.8, after: 60.2 },
			{ label: 'BUA Foods', before: 20.4, after: 85.1 },
			{ label: 'Nestlé Nigeria', before: 5.4, after: 26.9 }
		],
		barLabels: ['Q1 2025', 'Q1 2026'],
		note: 'NASCON is now owed more than one and a half times what it sells in a quarter. At International Breweries it is 60.7% of turnover; at Champion, 46.7%.',
		source: 'BusinessDay analysis of Q1 2026 filings. Naira billions.'
	},
	{
		kind: 'figures',
		eyebrow: 'Cash from operations, first quarter 2026',
		title: 'The sale is booked. The cash is still with the distributor.',
		figures: [
			{ label: 'BUA Foods', value: '−71%', note: '₦29.1bn down to ₦8.4bn.', tone: 'alert' },
			{ label: 'Nestlé Nigeria', value: '−50%', note: '₦114.3bn down to ₦56.8bn.', tone: 'alert' },
			{ label: 'NASCON Allied', value: '−58%', note: 'While profit went up.', tone: 'alert' }
		],
		source: 'BusinessDay analysis of Q1 2026 filings.'
	},
	{
		kind: 'quote',
		eyebrow: 'Why it continues',
		title: 'Nobody can afford to stop giving credit.',
		quote: {
			text: 'Credit sales are now a survival tool for the sector. It helps manufacturers keep products in circulation while giving retailers breathing room.',
			who: 'Uchenna Uzo, Faculty Director, Lagos Business School'
		},
		note: 'A distribution executive in the same report put it more simply: tighten credit and the volume goes with it. So the question is not whether to give credit. It is what you are holding when a customer stops paying.',
		source: 'BusinessDay, Q1 2026 market intelligence.'
	},
	{
		kind: 'chain',
		eyebrow: 'Where the credit comes from',
		title: 'The shop at the end of the line has no bank. So the credit comes from you.',
		tiers: [
			['Manufacturer', 'Ends up carrying it, as trade debtors.'],
			['Distributor', 'Takes stock on credit, and gives credit to the shops.'],
			['Retailer', 'Kiosks, stalls and open markets. No bank facility.'],
			['Customer', 'Buys from the shop, often on credit too.']
		],
		figures: [
			{ label: 'Retail through traditional trade', value: '9 in 10', note: 'Naira of Nigerian retail.' },
			{ label: 'Retailers who say finance is the problem', value: '74%', note: '18% have ever had a formal loan.' },
			{ label: 'Unmet small-business credit', value: '$32.2bn', note: 'About ₦13tn, across some 39.6m businesses.' }
		],
		source:
			'FieldAssist on Nigeria’s FMCG open market; BusinessDay retailer survey; IFC “Market Bite Nigeria” (2023); SMEDAN.'
	},
	{
		kind: 'compare',
		eyebrow: 'The gap',
		title: 'The credit is already there. The paperwork is not.',
		compare: [
			{
				heading: 'How it runs today',
				items: [
					'The terms live in a chat, and each side remembers them differently.',
					'Payment waits on goodwill and your sales rep’s phone calls.',
					'You find out at quarter end, as a trade debtors line.'
				]
			},
			{
				heading: 'On Kredit',
				items: [
					'One agreement, accepted by both sides and sealed on acceptance.',
					'The bank permission is given before the goods move.',
					'Every sale’s balance is visible the day it changes.'
				]
			}
		]
	},
	{
		kind: 'split',
		eyebrow: 'What Kredit does',
		title: 'We do not lend. You stay in charge of credit.',
		lede: 'You supply the goods and decide who gets credit, how much and on what terms. Kredit adds the agreement around it and the means to collect.',
		points: [
			['The credit decision stays yours', 'Limits, terms, who qualifies. Kredit never fronts goods or cash.'],
			['One agreement, the same on both screens', 'Your distributor sees exactly the terms they accepted. So do you.'],
			[
				'Bank permission comes first',
				'Collection runs on the due date against it, not as recovery after the account has gone bad.'
			]
		]
	},
	{
		kind: 'steps',
		eyebrow: 'How a sale runs',
		title: 'Four steps, the same for every sale.',
		points: [
			['Connect', 'You invite a distributor. They join with their own account and confirm their business.'],
			['Agree', 'Goods, price and payment dates go on record. They accept, and give the bank permission.'],
			['Deliver', 'Dispatch and receipt are recorded against the sale. A short delivery is reported there.'],
			['Settle', 'Payments are collected on the agreed dates and confirmed against the balance.']
		]
	},
	{
		kind: 'split',
		eyebrow: 'When something goes wrong',
		title: 'The record answers the argument.',
		lede: 'Six months on, when a distributor disputes what they collected in March, you open the sale instead of searching old messages.',
		points: [
			['Sealed on acceptance', 'Each agreement carries a fingerprint and the terms version in force that day.'],
			['Evidence stays with the sale', 'Invoice, dispatch, delivery confirmation and any problem reported.'],
			[
				'A complaint holds only the amount in question',
				'The rest of the balance can still be collected while it is looked at.'
			]
		],
		note: 'A bank debit needs valid permission, the agreed collection time and the payment and dispute checks to pass. It can still fail. Kredit does not guarantee repayment; it removes the delay and the excuses.'
	},
	{
		kind: 'split',
		eyebrow: 'Your network',
		title: 'Your customer list stays yours.',
		lede: 'A distributor can give credit to their own retailers on Kredit. That does not open their book to you, or yours to anyone above you.',
		points: [
			['A sale is visible to the two businesses in it', 'Nobody else. We do not sell around you.'],
			[
				'Balances stay separate',
				'What a distributor collects from its shops does not automatically clear what it owes you.'
			]
		]
	},
	{
		kind: 'split',
		eyebrow: 'What it costs',
		title: 'You pay when it works.',
		points: [
			['A base fee when a sale goes live', 'On the value of the sale, with a minimum, never more than the sale.'],
			['A collection fee on money that arrives', 'A failed debit costs you nothing.'],
			['No monthly fee, no joining fee', 'A sale keeps the fee terms it was agreed on, even if rates change.']
		],
		note: 'The seller pays. It is never added to what your customer owes. Current rates are at kredit.ng/pricing.'
	},
	{
		kind: 'steps',
		eyebrow: 'Getting started',
		title: 'Start with one distributor.',
		points: [
			['Pick one account', 'The customer whose balance already gives you the most trouble.'],
			['Run it alongside', 'Record the same sales both ways, and compare them at month end.'],
			['Widen it', 'Bring the next distributors on once that first account has been through a payment cycle.']
		],
		note: 'It runs in a browser. Your distributors have nothing to install.'
	},
	{
		kind: 'cover',
		eyebrow: 'Next step',
		title: 'One distributor.<br /><em>One payment cycle.</em>',
		lede: 'Give us one credit customer and one cycle, and judge Kredit on what you can see at the end of it.',
		links: [
			['Try the demo', '/demo'],
			['How it works', '/how-it-works'],
			['What it costs', '/pricing'],
			['Talk to us', '/contact']
		],
		meta: 'kredit.ng · hello@kredit.ng'
	}
];
