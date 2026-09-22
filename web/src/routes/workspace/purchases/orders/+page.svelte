<script lang="ts">
	import { buyerEndpoint } from '$lib/buyer-navigation';
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';
	import { readableDate } from '$lib/datetime';
</script>

<WorkspacePage
	eyebrow="Your business · Purchases"
	title="Purchase offers"
	description="Review the goods, price, payment dates and delivery terms before accepting a supplier’s offer."
	endpoint={buyerEndpoint('/api/v1/buyer/credit-requests')}
	collectionKey="requests"
	emptyTitle="No purchase offers to review"
	emptyCopy="When a supplier sends you goods on credit, the offer appears here before you accept it."
	searchPlaceholder="Supplier or goods"
	keep={(view) => view.request?.state === 'SENT' || view.request?.state === 'BUYER_REVIEWING'}
	rowTitle={(view) => view.request?.supplier_trading_name || view.request?.supplier_legal_name || 'Seller'}
	rowDetail={(view) =>
		[view.request?.goods_description, view.request?.due_date ? `Pay by ${readableDate(view.request.due_date)}` : '']
			.filter(Boolean)
			.join(' · ')}
	rowStatus={(view) => view.request?.state ?? ''}
	rowAmount={(view) => view.request?.principal_kobo ?? null}
	rowAmountLabel="Value of goods"
	rowHref={(view) => `/workspace/purchases/orders/${encodeURIComponent(view.request?.id ?? '')}`}
/>
