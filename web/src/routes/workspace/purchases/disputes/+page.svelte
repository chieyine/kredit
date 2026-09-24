<script lang="ts">
	import type { KoboValue } from '$lib/money';
	import { buyerEndpoint } from '$lib/buyer-navigation';
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { reasonText } from '$lib/product-language';
	import { readableDate } from '$lib/datetime';
	type Problem = {
		id: string;
		reason?: string;
		explanation?: string;
		opened_at?: string;
		state?: string;
		remaining_disputed_kobo?: KoboValue;
		total_disputed_kobo?: KoboValue;
	};
</script>

<WorkspacePage
	eyebrow="Problems"
	title="Problems"
	description="Say which goods or how much money is affected, add a photo or paper, and see what the seller replies."
	endpoint={buyerEndpoint('/api/v1/buyer/disputes')}
	collectionKey="disputes"
	emptyTitle="No open problems"
	emptyCopy="If goods arrive short, damaged or wrong, report it from the sale and it will appear here."
	searchPlaceholder="Reason"
	rowTitle={(problem: Problem) => reasonText(problem.reason ?? '')}
	rowDetail={(problem) => [problem.explanation, readableDate(problem.opened_at)].filter(Boolean).join(' · ')}
	rowStatus={(problem) => problem.state ?? ''}
	rowAmount={(problem) => problem.remaining_disputed_kobo ?? problem.total_disputed_kobo ?? null}
	rowAmountLabel="Amount in question"
	rowHref={(problem) => `/workspace/purchases/disputes/${encodeURIComponent(problem.id)}`}
/>
