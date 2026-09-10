<script lang="ts">
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { readableDate } from '$lib/datetime';
</script>

<WorkspacePage
	eyebrow="Sales waiting for me"
	title="Sales waiting for you"
	description="The goods, the money, the payment day and what happens if you pay late. Open one to accept or decline it."
	endpoint="/api/v1/buyer/credit-requests"
	collectionKey="requests"
	emptyTitle="No sale is waiting for you"
	emptyCopy="When a seller sends you goods on credit, the offer appears here before you accept it."
	searchPlaceholder="Seller or goods"
	keep={(view) => view.request?.state === 'SENT' || view.request?.state === 'BUYER_REVIEWING'}
	rowTitle={(view) =>
		view.request?.supplier_trading_name || view.request?.supplier_legal_name || 'Seller'}
	rowDetail={(view) =>
		[
			view.request?.goods_description,
			view.request?.due_date ? `Pay by ${readableDate(view.request.due_date)}` : ''
		]
			.filter(Boolean)
			.join(' · ')}
	rowStatus={(view) => view.request?.state ?? ''}
	rowAmount={(view) => view.request?.principal_kobo ?? null}
	rowAmountLabel="Value of goods"
	rowHref={(view) => `/buyer/credit-requests/${encodeURIComponent(view.request?.id ?? '')}`}
/>
