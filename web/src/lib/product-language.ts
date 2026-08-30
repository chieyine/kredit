const labels: Record<string, string> = {
	DRAFT: 'Not sent yet', BUYER_REVIEWING: 'Waiting for customer', PENDING_BUYER_CONFIRMATION: 'Waiting for customer',
	BUYER_CONFIRMED: 'Customer said yes', READY_TO_RELEASE: 'Ready to give the goods',
	GOODS_RELEASED: 'Waiting for customer to confirm the goods', RECEIPT_ISSUE_REPORTED: 'Customer reported a problem',
	ACTIVATED: 'Payment has started', ACTIVE: 'Active', SUSPENDED: 'Paused', COMPLETED: 'Paid in full', PAID: 'Paid',
	OVERDUE: 'Payment is late', PENDING: 'Waiting', RECOGNIZED: 'Money received', CONFIRMED: 'Confirmed',
	REJECTED: 'Not approved', CANCELLED: 'Cancelled', EXPIRED: 'Time ended', FAILED: 'Did not work', OPEN: 'Open',
	RESOLVED: 'Solved', WITHDRAWN: 'Withdrawn', NO_ISSUE: 'Goods received', ISSUE_REPORTED: 'Problem reported',
	NOT_STARTED: 'Not started', IN_PROGRESS: 'In progress', APPROVED: 'Approved', VERIFIED: 'Checked', CONFIGURED: 'Set up',
	CONTESTED_ONLY: 'Only the money in question is paused', FULL_BLOCK: 'All bank debit is paused',
	NO_AUTOMATIC_BLOCK: 'Bank debit continues', UPHELD: 'Problem accepted', PARTIALLY_UPHELD: 'Partly accepted',
	INTEGRATED_VOLUNTARY: 'Paid online', SUPPLIER_RECORDED_TRANSFER: 'Bank transfer', BUYER_PAYMENT_CLAIM: 'Transfer reported',
	CASH_RECORDED: 'Cash', KREDIT_COLLECTION: 'Collected by Kredit', ADJUSTMENT: 'Account correction',
	OWNER: 'Owner', SALES: 'Add sales', FINANCE: 'Handle money', COLLECTIONS: 'Follow up payments',
	ADMINISTRATOR: 'Manage the account', VIEWER: 'View only', REMOVED: 'Access removed',
	ACCESS: 'See my information', CORRECTION: 'Correct my information', DELETION: 'Remove my information',
	RESTRICTION: 'Limit how my information is used', OBJECTION: 'Stop a use of my information',
	CONSENT_WITHDRAWAL: 'Take back my permission', PORTABILITY: 'Download my information',
	PENDING_VERIFICATION: 'Waiting for identity check', PENDING_REVIEW: 'Being checked',
	CLARIFICATION_REQUIRED: 'More information needed', PARTIALLY_APPROVED: 'Partly approved',
	COOLING_OFF: 'Safety waiting period'
};

export function productLabel(value: unknown, fallback = 'Not available') {
	if (value === null || value === undefined || value === '') return fallback;
	const raw = String(value);
	return labels[raw.toUpperCase()] ?? raw.replaceAll('_', ' ').toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}
