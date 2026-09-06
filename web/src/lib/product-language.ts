const labels: Record<string, string> = {
	DRAFT: 'Draft — not sent', BUYER_REVIEWING: 'Waiting for customer', PENDING_BUYER_CONFIRMATION: 'Waiting for customer',
	BUYER_CONFIRMED: 'Customer accepted', READY_TO_RELEASE: 'Ready to release goods',
	GOODS_RELEASED: 'Waiting for delivery confirmation', RECEIPT_ISSUE_REPORTED: 'Delivery problem reported',
	ACTIVATED: 'Payment tracking started', ACTIVE: 'Active', SUSPENDED: 'Paused', COMPLETED: 'Paid in full', PAID: 'Paid',
	OVERDUE: 'Payment overdue', PENDING: 'Waiting', RECOGNIZED: 'Payment received', CONFIRMED: 'Confirmed',
	REJECTED: 'Not approved', CANCELLED: 'Cancelled', EXPIRED: 'Expired', FAILED: 'Unsuccessful', OPEN: 'Open',
	RESOLVED: 'Resolved', WITHDRAWN: 'Withdrawn', NO_ISSUE: 'Goods received', ISSUE_REPORTED: 'Problem reported',
	NOT_STARTED: 'Not started', IN_PROGRESS: 'In progress', APPROVED: 'Approved', VERIFIED: 'Verified', CONFIGURED: 'Set up',
	CONTESTED_ONLY: 'Only the disputed amount is paused', FULL_BLOCK: 'All bank debits are paused',
	NO_AUTOMATIC_BLOCK: 'Bank debit continues', UPHELD: 'Problem accepted', PARTIALLY_UPHELD: 'Partly accepted',
	INTEGRATED_VOLUNTARY: 'Paid online', SUPPLIER_RECORDED_TRANSFER: 'Bank transfer recorded', BUYER_PAYMENT_CLAIM: 'Transfer reported',
	CASH_RECORDED: 'Cash recorded', KREDIT_COLLECTION: 'Collected through Kredit', ADJUSTMENT: 'Account adjustment',
	OWNER: 'Owner', SALES: 'Create and manage sales', FINANCE: 'Manage payments', COLLECTIONS: 'Follow up overdue payments',
	ADMINISTRATOR: 'Manage the account', VIEWER: 'View only', REMOVED: 'Access removed',
	ACCESS: 'Access my information', CORRECTION: 'Correct my information', DELETION: 'Delete my information',
	RESTRICTION: 'Restrict use of my information', OBJECTION: 'Object to a use of my information',
	CONSENT_WITHDRAWAL: 'Withdraw my permission', PORTABILITY: 'Download my information',
	PENDING_VERIFICATION: 'Verification pending', PENDING_REVIEW: 'Under review',
	CLARIFICATION_REQUIRED: 'More information needed', PARTIALLY_APPROVED: 'Partly approved',
	COOLING_OFF: 'Safety waiting period'
};

export function productLabel(value: unknown, fallback = 'Not available') {
	if (value === null || value === undefined || value === '') return fallback;
	const raw = String(value);
	return labels[raw.toUpperCase()] ?? raw.replaceAll('_', ' ').toLowerCase().replace(/^./, (letter) => letter.toUpperCase());
}
