<script lang="ts">
	import type { KoboValue } from '$lib/money';
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { readableDate } from '$lib/datetime';
	type OverdueSale = {
		id: string;
		buyer_legal_name?: string;
		description?: string;
		due_date?: string;
		state?: string;
		amount_kobo?: KoboValue;
	};
</script>

<WorkspacePage
	eyebrow="Money overdue"
	title="Overdue"
	description="Sales whose payment day has passed. Open one to see the balance and what to do next."
	organizationPath="/overdue"
	collectionKey="overdue"
	emptyTitle="Nothing is overdue"
	emptyCopy="A sale appears here once its payment day has passed."
	searchPlaceholder="Customer or goods"
	rowTitle={(sale: OverdueSale) => sale.buyer_legal_name || 'Customer'}
	rowDetail={(sale) =>
		[sale.description, sale.due_date ? `Payment day was ${readableDate(sale.due_date)}` : '']
			.filter(Boolean)
			.join(' · ')}
	rowStatus={(sale) => sale.state ?? 'OVERDUE'}
	rowAmount={(sale) => sale.amount_kobo ?? null}
	rowAmountLabel="Past due"
	rowHref={(sale, organization) =>
		`/workspace/sales/${encodeURIComponent(sale.id)}?organization=${encodeURIComponent(organization)}`}
/>
