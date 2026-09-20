<script lang="ts">
 import { buyerEndpoint } from '$lib/buyer-navigation';
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';

	// /workspace/purchases/obligations/[id] reads an obligation id, not the credit request's.
	// Linking the request id sent every row to a sale that does not exist.
	const obligationHref = (view: Record<string, any>) =>
		view.obligation?.id ? `/workspace/purchases/obligations/${encodeURIComponent(view.obligation.id)}` : '';
</script>

<WorkspacePage
	eyebrow="Money I owe"
	title="What you owe"
	description="For each sale: the goods, the payment day and what is left to pay."
	endpoint={buyerEndpoint('/api/v1/buyer/credit-requests')}
	collectionKey="requests"
	emptyTitle="You owe nothing right now"
	emptyCopy="An obligation appears after the agreement’s acceptance, bank-permission and delivery conditions have been met."
	searchPlaceholder="Seller or goods"
	keep={(view) => Boolean(view.obligation)}
	rowTitle={(view) =>
		view.request?.supplier_trading_name || view.request?.supplier_legal_name || 'Seller'}
	rowDetail={(view) =>
		[
			view.request?.goods_description,
			'Open for current payment days'
		]
			.filter(Boolean)
			.join(' · ')}
	rowStatus={(view) => view.obligation?.payment_status ?? view.request?.state ?? ''}
	rowAmount={(view) => view.obligation?.outstanding_kobo ?? null}
	rowAmountLabel="Left to pay"
	rowHref={obligationHref}
/>
