<script lang="ts">
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { readableDateTime } from '$lib/datetime';
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
	rowTitle={(attempt) => attempt.buyer_legal_name || 'Customer'}
	rowDetail={(attempt) =>
		[attempt.description, readableDateTime(attempt.created_at)].filter(Boolean).join(' · ')}
	rowStatus={(attempt) => attempt.state ?? ''}
	rowAmount={(attempt) => attempt.amount_kobo ?? null}
	rowAmountLabel="Asked for"
	rowHref={(attempt, organization) =>
		`/app/credit/${encodeURIComponent(attempt.id)}?organization=${encodeURIComponent(organization)}`}
/>
