import type { Slide } from './types';

/*
 * The manufacturer deck.
 *
 * Voice: Nigerian commercial English. Accounting vocabulary used plainly —
 * trade debtors, turnover, default, recovery — broken up by short flat
 * sentences. No em-dash reversals, no matched triads, no epigrams.
 *
 * Every figure is published and cited on the slide that carries it. Nothing
 * here describes Kredit's own volumes or customers. Where the product has a
 * limit, the slide states it in the words the product already uses.
 */
export const manufacturerDeck: Slide[] = [
  {
    kind: 'cover',
    eyebrow: 'For manufacturers',
    title: 'Your trade debtors are growing.<br /><em>Your cash is not.</em>',
    lede: 'Kredit puts an agreement, a bank mandate and a record behind credit you are already extending to your distributors.',
    meta: 'Kredit Technologies Limited · RC 9834452 · Maiduguri, Nigeria'
  },
  {
    kind: 'data',
    eyebrow: 'First quarter 2026',
    title: 'Receivables up 22%. Turnover flat.',
    lede: 'Ten listed Nigerian food and drinks manufacturers, first quarter 2026 against the same quarter last year.',
    figures: [
      { label: 'Combined turnover', value: '₦1.78tn', note: '₦1.76tn a year before. Nothing moved.' },
      { label: 'Owed by the trade', value: '₦515.3bn', note: 'Up from ₦423.4bn.', tone: 'alert' },
      { label: 'Combined profit', value: '₦307.5bn', note: 'Up 19.6%, on paper.' }
    ],
    source: 'BusinessDay analysis of Q1 2026 filings: BUA Foods, Nestlé Nigeria, Unilever Nigeria, NASCON Allied, Dangote Sugar, Nigerian Breweries, International Breweries, Guinness Nigeria, Cadbury Nigeria, Champion Breweries.'
  },
  {
    kind: 'data',
    eyebrow: 'First quarter 2026',
    title: 'For some of them it tripled.',
    rows: {
      columns: ['Manufacturer', 'Q1 2025', 'Q1 2026', 'Change'],
      data: [
        ['NASCON Allied', '₦17.8bn', '₦60.2bn', '3.4×'],
        ['BUA Foods', '₦20.4bn', '₦85.1bn', '4.2×'],
        ['Nestlé Nigeria', '₦5.4bn', '₦26.9bn', '5.0×']
      ]
    },
    note: 'NASCON is owed more than one and a half times what it sells in a quarter. International Breweries stands at 60.7% of turnover, Champion at 46.7%. That money is with distributors and retailers who have already collected the goods.',
    source: 'BusinessDay analysis of Q1 2026 filings.'
  },
  {
    kind: 'data',
    eyebrow: 'What it costs you',
    title: 'The sale is booked. The money is with the distributor.',
    figures: [
      { label: 'BUA Foods, cash from operations', value: '−71%', note: '₦29.1bn down to ₦8.4bn.', tone: 'alert' },
      { label: 'Nestlé Nigeria', value: '−50%', note: '₦114.3bn down to ₦56.8bn.', tone: 'alert' },
      { label: 'NASCON Allied', value: '−58%', note: 'With profit up.', tone: 'alert' }
    ],
    source: 'BusinessDay analysis of Q1 2026 filings.'
  },
  {
    kind: 'statement',
    eyebrow: 'Why it continues',
    title: 'Nobody can afford to stop.',
    quote: {
      text: 'Credit sales are now a survival tool for the sector. It helps manufacturers keep products in circulation while giving retailers breathing room.',
      who: 'Uchenna Uzo, Faculty Director, Lagos Business School'
    },
    note: 'A distribution executive in the same report said it more directly: tighten credit and the volume goes with it. So the argument is not whether to give credit. It is what you are holding when the customer stops picking up.',
    source: 'BusinessDay, Q1 2026 market intelligence.'
  },
  {
    kind: 'data',
    eyebrow: 'Further down',
    title: 'The shop at the end of the line has no bank.',
    lede: 'Close to nine in ten naira of Nigerian retail passes through kiosks, stalls and open markets. No bank is giving those traders a facility. So the credit comes from the man above them, and eventually it comes from you.',
    figures: [
      { label: 'Retail through traditional trade', value: '~90%' },
      { label: 'Retailers who say finance is the problem', value: '74%', note: '18% have ever had a formal loan.' },
      { label: 'Unmet MSME credit demand', value: '$32.2bn', note: 'About ₦13tn across some 39.6m businesses.' }
    ],
    source: 'FieldAssist on Nigeria’s FMCG open market; BusinessDay retailer survey; IFC “Market Bite Nigeria” (2023); SMEDAN.'
  },
  {
    kind: 'statement',
    eyebrow: 'The gap',
    title: 'The credit is already there. The paperwork is not.',
    lede: 'Most of this trade runs on a phone call and a long memory. He will pay. He is a regular. That holds until the month it does not.',
    points: [
      ['The terms are in a chat', 'When the balance is disputed there is nothing neutral to open.'],
      ['Payment waits on goodwill', 'Somebody has to call, then travel down, then call again. Usually the sales rep, who should be opening new accounts.'],
      ['You see it late', 'It surfaces as trade debtors at the end of the quarter, months after it mattered.']
    ]
  },
  {
    kind: 'statement',
    eyebrow: 'What Kredit does',
    title: 'We do not lend you or your customer anything.',
    lede: 'You supply the goods. You decide who gets credit, how much, and on what terms. What we add is the agreement around it and the means to collect.',
    points: [
      ['The credit decision stays yours', 'Limits, terms, who qualifies. Kredit never fronts goods or cash, and never guarantees repayment.'],
      ['One agreement, accepted by both parties', 'Sealed on acceptance, so it is not edited afterwards.'],
      ['Bank permission taken up front', 'Collection runs on the due date against a mandate, instead of recovery after the account has gone bad.']
    ]
  },
  {
    kind: 'steps',
    eyebrow: 'How a trade runs',
    title: 'Connect. Agree. Deliver. Settle.',
    points: [
      ['Connect', 'You invite a distributor. He joins with his own account and confirms the business he represents.'],
      ['Agree', 'Goods, price and payment dates go on record. Your customer reads it, accepts it, and gives the bank permission the trade requires.'],
      ['Deliver', 'Dispatch and receipt are logged against that trade. Short delivery is reported there, not argued about three weeks later.'],
      ['Settle', 'Due dates followed, payments confirmed, balance reconciled.']
    ]
  },
  {
    kind: 'statement',
    eyebrow: 'The agreement',
    title: 'Both parties are looking at the same page.',
    points: [
      ['No second version', 'Your distributor sees the terms he accepted. You see the same ones.'],
      ['Sealed', 'Each accepted agreement carries a cryptographic hash and the terms version in force the day it was signed.'],
      ['The evidence stays attached', 'Invoice reference, dispatch, delivery confirmation, and any problem reported against the trade.']
    ],
    note: 'Six months later, when a distributor disputes what he collected in March, the record answers it. Not whoever kept better notes.'
  },
  {
    kind: 'statement',
    eyebrow: 'Collection',
    title: 'The bank permission comes before the goods move.',
    lede: 'Authorising the debit is part of accepting the trade. It is not a favour you go back and ask for once the account is overdue.',
    note: 'We state this inside the product and we will state it here. A bank debit needs valid permission, the agreed collection time, and the payment and dispute checks to pass. A debit can still fail. Kredit does not guarantee repayment. What it removes is the delay, the ambiguity and the excuse.',
    source: 'Bank authorisation runs through licensed providers. A debit already submitted to a bank may still complete.'
  },
  {
    kind: 'statement',
    eyebrow: 'When there is a problem',
    title: 'A complaint holds the amount in question. Not the whole account.',
    points: [
      ['Reported against the trade', 'With evidence, instead of the customer simply going silent on your rep.'],
      ['Only the disputed amount is held', 'Undisputed balances can still be collected while it is reviewed.'],
      ['The decision is on file', 'Whoever handles it next can see what was claimed and what was decided.']
    ]
  },
  {
    kind: 'statement',
    eyebrow: 'Your network',
    title: 'Your customer list stays your customer list.',
    lede: 'A distributor can run his own credit to retailers on Kredit. That does not open his book to you, and it does not open yours to anybody upstream.',
    points: [
      ['A trade is visible to the two businesses in it', 'Nobody else, ourselves included. We do not sell around you.'],
      ['Balances stay separate', 'What a distributor collects downstream does not automatically clear what he owes you.']
    ]
  },
  {
    kind: 'statement',
    eyebrow: 'What it costs',
    title: 'You pay when it works.',
    points: [
      ['A base fee when a sale goes live', 'On the value of the activated sale, with a minimum, capped at the sale amount.'],
      ['A collection fee on what actually lands', 'A failed debit earns us nothing.'],
      ['No monthly fee, no joining fee', 'A sale already accepted keeps its fee terms even if rates change later.']
    ],
    note: 'The seller pays. It is not added to your customer’s balance. Current rates are published at kredit.ng/pricing.'
  },
  {
    kind: 'steps',
    eyebrow: 'Getting started',
    title: 'One distributor. Not the whole network.',
    points: [
      ['Start with one', 'Pick the customer whose account already gives you trouble.'],
      ['Run it beside what you do now', 'Same trades, recorded both ways. Compare the two at month end.'],
      ['Widen it when you are satisfied', 'Bring the others on after that first account has been through a full payment cycle.']
    ],
    note: 'It runs in a browser. Your distributors have nothing to install.'
  },
  {
    kind: 'cover',
    eyebrow: 'Where to go next',
    title: 'One distributor.<br /><em>One payment cycle.</em>',
    lede: 'Give us one credit customer and one cycle. Judge it on what you can see at the end of it.',
    links: [
      ['Try the demo', '/demo'],
      ['How it works', '/how-it-works'],
      ['What it costs', '/pricing'],
      ['Talk to us', '/contact']
    ],
    meta: 'kredit.ng · hello@kredit.ng'
  }
];
