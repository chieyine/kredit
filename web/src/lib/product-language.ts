const labels: Record<string, string> = {
	DRAFT: 'Not sent yet', BUYER_REVIEWING: 'Waiting for customer', PENDING_BUYER_CONFIRMATION: 'Waiting for customer',
	BUYER_CONFIRMED: 'Customer accepted', READY_TO_RELEASE: 'You can send the goods',
	GOODS_RELEASED: 'Waiting for customer to confirm goods', RECEIPT_ISSUE_REPORTED: 'Problem with the delivery',
	ACTIVATED: 'Payment has started', ACTIVE: 'Active', SUSPENDED: 'Paused', COMPLETED: 'Paid in full', PAID: 'Paid',
	OVERDUE: 'Payment is late', PENDING: 'Waiting', RECOGNIZED: 'Money confirmed', CONFIRMED: 'Confirmed',
	REJECTED: 'Turned down', CANCELLED: 'Cancelled', EXPIRED: 'Expired', FAILED: 'Did not work', OPEN: 'Open',
	RESOLVED: 'Resolved', WITHDRAWN: 'Withdrawn', NO_ISSUE: 'Goods received', ISSUE_REPORTED: 'Problem reported',
	NOT_STARTED: 'Not started', IN_PROGRESS: 'In progress', APPROVED: 'Approved', VERIFIED: 'Verified', CONFIGURED: 'Set up',
	CONTESTED_ONLY: 'Only the money in question is on hold', FULL_BLOCK: 'All bank debits are on hold',
	NO_AUTOMATIC_BLOCK: 'Bank debit still continues', UPHELD: 'Customer was right', PARTIALLY_UPHELD: 'Customer was partly right',
	INTEGRATED_VOLUNTARY: 'Paid online', SUPPLIER_RECORDED_TRANSFER: 'Bank transfer recorded', BUYER_PAYMENT_CLAIM: 'Customer says they paid',
	CASH_RECORDED: 'Cash recorded', KREDIT_COLLECTION: 'Debited from their bank', ADJUSTMENT: 'Correction',
	OWNER: 'Owner', SALES: 'Add and manage sales', FINANCE: 'Manage payments', COLLECTIONS: 'Chase late payments',
	ADMINISTRATOR: 'Manage the account', VIEWER: 'Can look, cannot change', REMOVED: 'Access removed',
	ACCESS: 'Show me my information', CORRECTION: 'Correct my information', DELETION: 'Delete my information',
	RESTRICTION: 'Stop using some of my information', OBJECTION: 'Object to how it is being used',
	CONSENT_WITHDRAWAL: 'Withdraw my permission', PORTABILITY: 'Give me a copy to download',
	PENDING_VERIFICATION: 'Still being checked', PENDING_REVIEW: 'Somebody is looking at it',
	CLARIFICATION_REQUIRED: 'We need more information', PARTIALLY_APPROVED: 'Partly approved',
	COOLING_OFF: 'Short safety wait'
};

export function productLabel(value: unknown, fallback = 'Not available') {
	if (value === null || value === undefined || value === '') return fallback;
	const raw = String(value);
	return labels[raw.toUpperCase()] ?? raw.replaceAll('_', ' ').toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}
