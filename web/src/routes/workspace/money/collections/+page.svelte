<script lang="ts">
	import type { KoboValue } from '$lib/money';
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { readableDateTime } from '$lib/datetime';
	type CollectionAttempt = {
		id: string;
		buyer_legal_name?: string;
		description?: string;
		created_at?: string;
		state?: string;
		amount_kobo?: KoboValue;
	};
</script>

<WorkspacePage
	eyebrow="Bank debits"
	title="Bank debits"
	description="Every time Kredit asked a customer’s bank for a payment, and what the bank said."
	organizationPath="/collections"
	collectionKey="collections"
	emptyTitle="No bank debit yet"
	emptyCopy="A bank debit appears here only after a payment is late and the extra days have ended."
	searchPlaceholder="Customer or result"
	rowTitle={(attempt: CollectionAttempt) => attempt.buyer_legal_name || 'Customer'}
	rowDetail={(attempt) => [attempt.description, readableDateTime(attempt.created_at)].filter(Boolean).join(' · ')}
	rowStatus={(attempt) => attempt.state ?? ''}
	rowAmount={(attempt) => attempt.amount_kobo ?? null}
	rowAmountLabel="Asked for"
	rowHref={(attempt, organization) =>
		`/workspace/sales/${encodeURIComponent(attempt.id)}?organization=${encodeURIComponent(organization)}`}
/>
