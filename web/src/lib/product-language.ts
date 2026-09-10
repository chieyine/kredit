/**
 * One name for every state, used everywhere it appears.
 *
 * Two rules the earlier version broke:
 *  - A label is a label. "Only the money in question is on hold" is a sentence;
 *    it belongs in the explanation beside a status, not inside the pill.
 *  - A financial state means one thing. "Did not work" covers a bank refusing a
 *    debit, a network timeout and an expired mandate, and a seller cannot act on
 *    it. Say which one happened.
 *
 * Plain does not mean childish. These read the way a trader would say them.
 */
const labels: Record<string, string> = {
	// Sale lifecycle
	DRAFT: 'Not sent',
	SENT: 'Sent to customer',
	BUYER_REVIEWING: 'With customer',
	PENDING_BUYER_CONFIRMATION: 'With customer',
	BUYER_CONFIRMED: 'Accepted',
	BUYER_ACCEPTED: 'Accepted',
	VERIFICATION_PENDING: 'Being checked',
	READY_TO_RELEASE: 'Ready to send goods',
	GOODS_RELEASED: 'Goods sent',
	RECEIPT_CONFIRMATION_PENDING: 'Waiting for delivery confirmation',
	RECEIPT_ISSUE_REPORTED: 'Delivery problem reported',
	ACTIVATED: 'Repayment started',
	ACTIVE: 'Active',
	SUSPENDED: 'Paused',
	COMPLETED: 'Completed',
	PAID: 'Paid in full',
	OVERDUE: 'Overdue',
	DECLINED: 'Declined by customer',
	CANCELLED: 'Cancelled',
	EXPIRED: 'Expired',

	// Payments and collection
	PENDING: 'Awaiting confirmation',
	RECOGNIZED: 'Confirmed',
	CONFIRMED: 'Confirmed',
	REVERSED: 'Reversed',
	REJECTED: 'Not confirmed',
	SUBMITTED: 'Sent to the bank',
	UNKNOWN: 'Result not yet known',
	FAILED: 'Failed',
	PARTIAL: 'Part of the amount collected',
	SUCCEEDED: 'Collected',
	INTEGRATED_VOLUNTARY: 'Paid online',
	SUPPLIER_RECORDED_TRANSFER: 'Bank transfer',
	BUYER_PAYMENT_CLAIM: 'Transfer reported by customer',
	CASH_RECORDED: 'Cash',
	KREDIT_COLLECTION: 'Collected by Kredit',
	ADJUSTMENT: 'Correction',

	// Disputes
	OPEN: 'Open',
	UNDER_REVIEW: 'Being reviewed',
	PARTIALLY_RESOLVED: 'Partly resolved',
	RESOLVED: 'Resolved',
	WITHDRAWN: 'Withdrawn',
	NO_ISSUE: 'Goods received',
	ISSUE_REPORTED: 'Problem reported',
	UPHELD: 'Report accepted',
	PARTIALLY_UPHELD: 'Report partly accepted',
	VALID_AMOUNT_CONFIRMED: 'Amount confirmed',
	PARTIAL_ADJUSTMENT: 'Balance partly reduced',
	FULL_ADJUSTMENT: 'Disputed amount removed',
	CONTESTED_ONLY: 'Disputed amount on hold',
	FULL_BLOCK: 'All debits on hold',
	NO_AUTOMATIC_BLOCK: 'Debits continue',

	// Setup and verification
	NOT_STARTED: 'Not started',
	IN_PROGRESS: 'In progress',
	APPROVED: 'Approved',
	VERIFIED: 'Verified',
	CONFIGURED: 'Set up',
	PENDING_VERIFICATION: 'Being checked',
	PENDING_REVIEW: 'Awaiting review',
	CLARIFICATION_REQUIRED: 'More information needed',
	PARTIALLY_APPROVED: 'Partly approved',
	COOLING_OFF: 'Waiting period',

	// Roles. A role is a name, not a description of the job.
	OWNER: 'Owner',
	ADMINISTRATOR: 'Administrator',
	FINANCE: 'Finance',
	SALES: 'Sales',
	COLLECTIONS: 'Collections',
	VIEWER: 'View only',
	REMOVED: 'Removed',

	// Privacy requests, named from the operator's side of the desk.
	ACCESS: 'Copy of their information',
	CORRECTION: 'Correction',
	DELETION: 'Deletion',
	RESTRICTION: 'Restrict use',
	OBJECTION: 'Objection to use',
	CONSENT_WITHDRAWAL: 'Consent withdrawn',
	PORTABILITY: 'Portable copy'
};

/**
 * What a role or permission lets someone do. Shown beside the role name where
 * there is room to explain, never inside the pill that names it.
 */
export const roleDescriptions: Record<string, string> = {
	OWNER: 'Everything, including staff and settings',
	ADMINISTRATOR: 'Staff, customers and business settings',
	FINANCE: 'Payments, balances and corrections',
	SALES: 'Add customers and record sales',
	COLLECTIONS: 'Follow up late payments',
	VIEWER: 'Can look at records but change nothing'
};

/**
 * The same seven requests, in the words of the person making them.
 *
 * `labels` above names these from the operator's side of the desk — "Copy of
 * their information" is right in a queue of other people's requests and wrong on
 * the screen where you ask for your own. Both wordings are needed; what is not
 * needed is the form offering one and the list beside it printing the other,
 * which is what happened. The person's page reads its choices and its list from
 * this one array, so they cannot drift apart again.
 */
export const privacyRequestChoices: { value: string; label: string }[] = [
	{ value: 'ACCESS', label: 'Show me everything you hold about me' },
	{ value: 'CORRECTION', label: 'Correct something that is wrong' },
	{ value: 'DELETION', label: 'Delete whatever you are allowed to delete' },
	{ value: 'RESTRICTION', label: 'Stop using some of my information' },
	{ value: 'OBJECTION', label: 'Object to how you are using it' },
	{ value: 'CONSENT_WITHDRAWAL', label: 'Take back a permission I gave' },
	{ value: 'PORTABILITY', label: 'Give me a copy I can download' }
];

/** What the person asked for, in the words they chose. */
export function privacyRequestLabel(value: unknown, fallback = 'Your request') {
	const raw = String(value ?? '').toUpperCase();
	return privacyRequestChoices.find((choice) => choice.value === raw)?.label ?? fallback;
}

export function productLabel(value: unknown, fallback = 'Not available') {
	if (value === null || value === undefined || value === '') return fallback;
	const raw = String(value);
	return (
		labels[raw.toUpperCase()] ??
		raw.replaceAll('_', ' ').toLowerCase().replace(/^./, (letter) => letter.toUpperCase())
	);
}
