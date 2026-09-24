import { productLabel } from './product-language';

// The database records every change to a customer-facing table as
// `record.insert`, `record.update` or `record.delete` against the table's name,
// and the in-app notice for it names the table too. Those are storage words.
// This turns them into what a person would call the thing that changed.

const nouns: Record<string, string> = {
	organizations: 'business',
	memberships: 'team member',
	organization_invitations: 'team invitation',
	businesses: 'business profile',
	business_representatives: 'business representative',
	buyer_invitations: 'customer invitation',
	relationship_consents: 'trading permission',
	credit_requests: 'sale',
	agreement_acceptances: 'agreement acceptance',
	goods_releases: 'goods dispatch',
	receipt_confirmations: 'delivery confirmation',
	system_acceptances: 'terms acceptance',
	obligations: 'amount owed',
	trade_lines: 'customer limit',
	drawdowns: 'sale on a limit',
	payments: 'payment',
	payment_claims: 'reported transfer',
	disputes: 'reported problem',
	dispute_decisions: 'decision on a problem',
	correction_requests: 'correction request',
	correction_decisions: 'correction decision',
	support_cases: 'help request',
	support_case_events: 'reply to a help request',
	privacy_requests: 'privacy request',
	processing_restrictions: 'data restriction',
	supplier_onboarding_profiles: 'selling setup',
	platform_role_assignments: 'admin access',
	platform_suspensions: 'account suspension',
	platform_settings: 'platform setting',
	business_policy_changes: 'business policy',
	operations_commands: 'operations action',
	documents: 'document'
};

const verbs: Record<string, string> = { insert: 'added', update: 'updated', delete: 'removed' };

function noun(table: string) {
	return nouns[table] ?? table.replaceAll('_', ' ').replace(/s$/, '');
}

function sentence(value: string) {
	return value.replace(/^./, (letter) => letter.toUpperCase());
}

/** One line for an activity entry, such as "Sale updated" or "Credit approval requested". */
export function activityLabel(action: string, resourceType = ''): string {
	const change = /^record\.(insert|update|delete)$/.exec(action);
	if (change) return sentence(`${noun(resourceType)} ${verbs[change[1]]}`);
	return productLabel(action.replaceAll('.', ' '));
}

const recordNoticeBody = /^A (.+?) record was updated\./;

/** The title and text for an automatic "something changed" notice. */
export function recordNotice(template: string, body: string): { title: string; body: string } | null {
	if (template !== 'RecordUpdated') return null;
	const table = recordNoticeBody.exec(body)?.[1]?.replaceAll(' ', '_');
	const thing = table ? noun(table) : 'record';
	return {
		title: sentence(`${thing} updated`),
		body: `Open the ${thing} in Kredit to see what changed.`
	};
}
