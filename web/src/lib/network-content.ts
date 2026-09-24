export const network = [
	{
		key: 'manufacturers',
		label: 'Manufacturers',
		short: 'You make the goods and give distributors time to pay.',
		title: 'Know what every distributor owes you.',
		description:
			'Record the credit you give each distributor, get the bank mandate before the goods leave your warehouse, and see a late payment while there is still time to act on it.',
		action: 'Set up your business',
		href: '/signin?next=%2Fworkspace%2Ftoday',
		panel: [
			['Distributors', 'Each one with their own limit and terms'],
			['Outstanding', 'What is due this week, and what is already late'],
			['Mandates', 'Which bank permissions are in place before dispatch']
		],
		steps: [
			[
				'Bring your distributors on',
				'Import the list you already keep, or invite one business first and see how it goes. Each distributor confirms who they are before they join.'
			],
			[
				'Put the terms on record',
				'Set a limit for each customer, then record the goods, the amount and the payment dates. The distributor reads the agreement and accepts it before anything is loaded.'
			],
			[
				'Follow each sale to the end',
				'Record the dispatch, confirm what was received, and watch the repayments come in. Late ones are flagged on the day they fall due.'
			]
		],
		next: 'Your distributors can sell on credit to their own customers from the same account. Those sales belong to them, and you cannot see them.'
	},
	{
		key: 'distributors',
		label: 'Distributors',
		short: 'You buy from manufacturers and sell on to shops.',
		title: 'What you owe and what you are owed, kept apart.',
		description:
			'Track what you owe your suppliers and the credit you give retailers in one account, without the two ever getting mixed up.',
		action: 'Open your account',
		href: '/signin?next=%2Fstart',
		panel: [
			['You owe', 'Each supplier, with the next payment date'],
			['Owed to you', 'Each retailer, with what is still outstanding'],
			['Deliveries', 'What came in, and any shortage you reported']
		],
		steps: [
			[
				'Accept your supplier’s invitation',
				'Open the link your manufacturer sent. Connect the business you already run, or set it up once and keep it.'
			],
			[
				'Keep track of what you owe',
				'Read the terms on each purchase, confirm what arrived, and see every supplier’s payment dates in one place.'
			],
			[
				'Start giving credit yourself',
				'Finish the selling checks, invite your retailers, and manage their credit separately from what you owe upstream.'
			]
		],
		next: 'When a retailer pays you, that money is not used to pay your supplier unless you pay it yourself. The two accounts never net off.'
	},
	{
		key: 'retailers',
		label: 'Retailers',
		short: 'You take stock on credit and sell to the people on your street.',
		title: 'Keep the shop’s credit straight.',
		description:
			'See what you took from your distributor and when it is due, alongside what your own customers still owe you.',
		action: 'Open your account',
		href: '/signin?next=%2Fstart',
		panel: [
			['Stock on credit', 'What you took, and the day each payment is due'],
			['Your customers', 'Who still owes you, and how much'],
			['Receipts', 'A record of every payment on both sides']
		],
		steps: [
			[
				'Connect to your supplier',
				'Accept the invitation with your business account, and read the terms on each purchase before you agree to it.'
			],
			[
				'Check every delivery',
				'Confirm what came, report a shortage straight away, and see what you still owe each supplier.'
			],
			[
				'Sell to your own customers',
				'Record a credit sale to another business, or send a private purchase link to a person buying for themselves. Each sale keeps its own terms.'
			]
		],
		next: 'A person buying for themselves does not need a business account. Send them the private purchase link and they can read the terms on their phone.'
	},
	{
		key: 'consumers',
		label: 'Consumers',
		short: 'You buy for yourself and pay over time.',
		title: 'See the full price before you agree.',
		description:
			'The seller, the total, the payment dates and the delivery are all shown to you before you accept anything.',
		action: 'View my purchases',
		href: '/signin?next=%2Fpersonal%2Fpurchases',
		panel: [
			['Total to pay', 'Including any charge, shown before you accept'],
			['Payment dates', 'Each instalment and the day it is due'],
			['Paid so far', 'Updated when the seller confirms your payment']
		],
		steps: [
			[
				'Open the link you were sent',
				'Use the private purchase link from the seller, and sign in with your email address or WhatsApp number.'
			],
			[
				'Read it before you accept',
				'Check the total, any charge on it, and the dates you have to pay. Giving bank permission is a separate choice, and it is yours to make.'
			],
			[
				'Keep track afterwards',
				'See what you have paid and what is left, report a problem, or ask to return something, all from the same page.'
			]
		],
		next: 'Your personal purchases are kept apart from every business account. You do not have to register a business to use Kredit.'
	}
] as const;
export type NetworkRole = (typeof network)[number]['key'];

export const tradeSteps = [
	[
		'Connect',
		'Invite a business you already trade with. They join with their own account and confirm the company they represent.'
	],
	[
		'Agree',
		'Record the goods, the price and the payment dates. Both sides accept the terms, and any bank permission the sale needs is given at the same time.'
	],
	[
		'Deliver',
		'Record what was sent and what arrived. Keep the waybill or photo on the sale, and report a shortage against it.'
	],
	['Get paid', 'Watch the due dates, confirm each payment as it lands, and see exactly what is left to collect.']
] as const;
