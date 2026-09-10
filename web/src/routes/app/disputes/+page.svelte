<script lang="ts">
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { productLabel } from '$lib/product-language';
	import { readableDate } from '$lib/datetime';
</script>

<WorkspacePage
	eyebrow="Problems"
	title="Problems"
	description="Problems your customers reported. Open one to see the proof and record your answer."
	organizationPath="/disputes"
	collectionKey="disputes"
	emptyTitle="No open problems"
	emptyCopy="When a customer reports a problem with goods or an amount, it appears here."
	searchPlaceholder="Reason or customer"
	rowTitle={(problem) => productLabel(problem.reason, 'Problem reported')}
	rowDetail={(problem) =>
		[problem.explanation, readableDate(problem.opened_at)].filter(Boolean).join(' · ')}
	rowStatus={(problem) => problem.state ?? ''}
	rowAmount={(problem) => problem.remaining_disputed_kobo ?? problem.total_disputed_kobo ?? null}
	rowAmountLabel="Amount in question"
	rowHref={(problem, organization) =>
		`/app/disputes/${encodeURIComponent(problem.id)}?organization=${encodeURIComponent(organization)}`}
/>
