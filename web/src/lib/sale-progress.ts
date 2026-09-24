import type { SaleView } from './records';
export type NextStep = { actor: string; title: string; detail: string; action: string };
// Guidance is based on recorded evidence. It never authorizes a financial action.
export function saleNextStep(view: SaleView): NextStep {
	const state = view.request.state;
	if (view.obligation && String(view.obligation.outstanding_kobo) === '0')
		return {
			actor: 'No action needed',
			title: 'The sale balance is paid',
			detail: 'Keep the agreement and payment records. Seller settlement is recorded separately.',
			action: 'Review payments'
		};
	const steps: Record<string, NextStep> = {
		DRAFT: {
			actor: 'Seller',
			title: 'Check and send the sale',
			detail: 'Confirm the goods, amount, fees and payment dates. Save your changes before sending.',
			action: 'Review draft'
		},
		SENT: {
			actor: 'Customer',
			title: 'Read the agreement and decide',
			detail: 'The customer reads the sale exactly as it was sent, then accepts or declines it.',
			action: 'Review agreement'
		},
		BUYER_REVIEWING: {
			actor: 'Customer',
			title: 'Read the agreement and decide',
			detail: 'Accepting agrees to the sale. Allowing a bank debit is a separate step that comes after.',
			action: 'Review agreement'
		},
		VERIFICATION_PENDING: {
			actor: 'Customer',
			title: 'Complete the account checks',
			detail: 'The customer confirms who they are and that they may buy for the business, then accepts.',
			action: 'Review account checks'
		},
		BUYER_ACCEPTED: {
			actor: 'Customer / bank',
			title: 'Finish bank-debit permission',
			detail: 'The sale is accepted. Next the customer allows the bank debit; the goods go out once the bank confirms.',
			action: 'Review bank permission'
		},
		READY_TO_RELEASE: {
			actor: 'Seller',
			title: 'Arrange and record delivery',
			detail: 'The customer has accepted and the bank has confirmed. Record the dispatch when the goods leave.',
			action: 'Review delivery'
		},
		GOODS_RELEASED: {
			actor: 'Customer',
			title: 'Confirm the goods arrived',
			detail: 'The seller has recorded that the goods left. The customer confirms they arrived, or reports a problem.',
			action: 'Review receipt'
		},
		RECEIPT_CONFIRMATION_PENDING: {
			actor: 'Customer',
			title: 'Confirm the goods arrived',
			detail:
				'The customer confirms the goods arrived, or reports a problem. If nobody answers within the agreed waiting time, the delivery record stands.',
			action: 'Review receipt'
		},
		ACTIVE: {
			actor: 'Customer',
			title: 'Payments are due on the agreed dates',
			detail:
				'Each payment is due on its agreed date. A reported transfer counts once the seller confirms it arrived. A bank debit shows as pending until the bank answers.',
			action: 'Review payments'
		},
		CANCELLED: {
			actor: 'No action needed',
			title: 'This sale was cancelled',
			detail: 'Its history remains available. A new sale requires a new agreement.',
			action: 'Review history'
		},
		DECLINED: {
			actor: 'Seller',
			title: 'The customer declined this sale',
			detail: 'Discuss any correction with the customer before creating a new offer.',
			action: 'Review history'
		}
	};
	return (
		steps[state] ?? {
			actor: 'Support',
			title: 'Check the current sale record',
			detail: 'This status needs review before taking another action.',
			action: 'Review history'
		}
	);
}
