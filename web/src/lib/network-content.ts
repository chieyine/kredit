export const network = [
	{
		key: 'manufacturers',
		label: 'Manufacturers',
		short: 'Make it.',
		title: 'Know what every distributor owes you.',
		description:
			'Put the credit you give distributors on record, take the bank mandate before the goods move, and see the outstanding before it turns into a bad debt.',
		action: 'Set up your business',
		href: '/signin?next=%2Fworkspace%2Ftoday',
		steps: [
			[
				'Bring your distributors on',
				'Import the roster you already have, or invite one business and start there. Each distributor confirms who he is and joins your account.'
			],
			[
				'Put the terms on record',
				'Set a limit for each customer. Record the goods, the amount and the payment dates. He reads the agreement and accepts it before anything moves.'
			],
			[
				'Follow the trade to the end',
				'Record dispatch. Confirm what was received. Watch the repayments, and the ones that are late.'
			]
		],
		next: 'Your distributors can sell to their own customers on the same workspace. What they do downstream is their business, not yours to see.'
	},
	{
		key: 'distributors',
		label: 'Distributors',
		short: 'Move it.',
		title: 'What you owe. What you are owed.',
		description:
			'Handle your supplier purchases and the credit you give retailers from one workspace, without mixing the two.',
		action: 'Open your workspace',
		href: '/signin?next=%2Fstart',
		steps: [
			[
				'Join your supplier',
				'Open the invitation your manufacturer sent. Connect the business you already run, or create it once and keep it.'
			],
			[
				'Stay on top of what you owe',
				'Check the terms, confirm what arrived, and keep each supplier’s payment dates in front of you.'
			],
			[
				'Start selling on credit yourself',
				'Finish your selling setup, invite your retailers, and run their credit separately from what you owe upstream.'
			]
		],
		next: 'Buying and selling are two sides of the same business. Money your customers pay you does not automatically clear what you owe your supplier.'
	},
	{
		key: 'retailers',
		label: 'Retailers',
		short: 'Sell it.',
		title: 'Keep the shop’s credit straight.',
		description: 'Follow what you took from your distributor, and what your own customers still owe you.',
		action: 'Open your workspace',
		href: '/signin?next=%2Fstart',
		steps: [
			[
				'Connect to your supplier',
				'Accept the invitation with your business account and read the terms on each purchase before you agree.'
			],
			[
				'Keep your stock purchases clear',
				'Check what came, report a shortage there and then, and see what is still due to each supplier.'
			],
			[
				'Serve your own customers',
				'Business sales for another business, personal purchases for an individual. Each one keeps its own terms and its own record.'
			]
		],
		next: 'An individual buying for himself needs a personal purchase account, not a business profile. Send him the private link and let him read it.'
	},
	{
		key: 'consumers',
		label: 'Consumers',
		short: 'Enjoy it.',
		title: 'Know the full price before you agree.',
		description:
			'The seller, the total, the payment dates and the delivery are all in front of you before you accept anything.',
		action: 'View my purchases',
		href: '/signin?next=%2Fpersonal%2Fpurchases',
		steps: [
			[
				'Open the link you were sent',
				'Use the private purchase link from the seller. Sign in with your own email or WhatsApp number.'
			],
			[
				'Read it before you accept',
				'The total, any charge on it, and the dates you are to pay. Bank permission is a separate decision you make yourself.'
			],
			[
				'Follow it afterwards',
				'Track what you have paid and what is left, report a problem, or ask to return something from the same record.'
			]
		],
		next: 'Your personal purchases stay separate from any business workspace. You do not need to register a business to use one.'
	}
] as const;
export type NetworkRole = (typeof network)[number]['key'];

export const tradeSteps = [
	[
		'Connect',
		'Invite the business you trade with. He joins with his own account and confirms the company he represents.'
	],
	[
		'Agree',
		'Record the goods, the price and the dates. Both sides accept the terms, and the bank permission the trade needs is taken then.'
	],
	[
		'Deliver',
		'Record what was dispatched and what was received. Keep the evidence on the trade, and report a shortage against it.'
	],
	['Settle', 'Follow the due dates, confirm what has been paid, and reconcile what is left.']
] as const;
