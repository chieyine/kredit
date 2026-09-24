import { optionalText, record, rows, text } from './api/reliable';
import type { KoboValue } from './money';
import { kobo } from './records';

// One decoder for the dispute record, shared by the customer, supplier and
// admin screens so they cannot disagree about what a verified dispute is.

export interface Dispute {
	id: string;
	state: string;
	reason: string;
	explanation: string;
	collection_effect: string;
	total_disputed_kobo: KoboValue;
	remaining_disputed_kobo: KoboValue;
	opened_at: string;
}
export interface DisputeEvidence {
	id: string;
	statement: string;
	document_id: string;
	submitted_at: string;
}
export interface DisputeDecision {
	id: string;
	outcome: string;
	reason: string;
	decided_at: string;
}
export interface DisputeDetail {
	dispute: Dispute;
	evidence: DisputeEvidence[];
	decisions: DisputeDecision[];
}

export function dispute(value: unknown): Dispute {
	const item = record(value);
	return {
		id: text(item.id),
		state: text(item.state),
		reason: optionalText(item.reason),
		explanation: optionalText(item.explanation),
		collection_effect: optionalText(item.collection_effect),
		total_disputed_kobo: kobo(item.total_disputed_kobo),
		remaining_disputed_kobo: kobo(item.remaining_disputed_kobo),
		opened_at: optionalText(item.opened_at)
	};
}

function evidence(value: unknown): DisputeEvidence {
	const item = record(value);
	return {
		id: optionalText(item.id),
		statement: optionalText(item.statement),
		document_id: optionalText(item.document_id),
		submitted_at: optionalText(item.submitted_at)
	};
}

function decision(value: unknown): DisputeDecision {
	const item = record(value);
	return {
		id: optionalText(item.id),
		outcome: text(item.outcome),
		reason: optionalText(item.reason),
		decided_at: optionalText(item.decided_at)
	};
}

export function disputeDetail(value: unknown): DisputeDetail {
	const body = record(value);
	return {
		dispute: dispute(body.dispute),
		evidence: rows('evidence', evidence)(body),
		decisions: rows('decisions', decision)(body)
	};
}
