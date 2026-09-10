<script lang="ts">
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { productLabel } from '$lib/product-language';
	import { readableDate } from '$lib/datetime';
</script>

<WorkspacePage
	eyebrow="Problems"
	title="Problems"
	description="Say which goods or how much money is affected, add a photo or paper, and see what the seller replies."
	endpoint="/api/v1/buyer/disputes"
	collectionKey="disputes"
	emptyTitle="No open problems"
	emptyCopy="If goods arrive short, damaged or wrong, report it from the sale and it will appear here."
	searchPlaceholder="Reason"
	rowTitle={(problem) => productLabel(problem.reason, 'Problem reported')}
	rowDetail={(problem) =>
		[problem.explanation, readableDate(problem.opened_at)].filter(Boolean).join(' · ')}
	rowStatus={(problem) => problem.state ?? ''}
	rowAmount={(problem) => problem.remaining_disputed_kobo ?? problem.total_disputed_kobo ?? null}
	rowAmountLabel="Amount in question"
	rowHref={(problem) => `/buyer/disputes/${encodeURIComponent(problem.id)}`}
/>
