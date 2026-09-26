<script lang="ts">
	import type { KoboValue } from '$lib/money';
	import { buyerEndpoint } from '$lib/buyer-navigation';
	import WorkspacePage from '$lib/components/WorkspacePage.svelte';

	// Every row opens the credit's own page: accept it, set up payment, pay, or report a transfer there.
	type PurchaseView = {
		request?: {
			id?: string;
			state?: string;
			supplier_legal_name?: string;
			supplier_trading_name?: string;
			goods_description?: string;
			due_date?: string;
			principal_kobo?: KoboValue;
		};
		obligation?: { id?: string; payment_status?: string; outstanding_kobo?: KoboValue } | null;
	};
	const closed = ['CANCELLED', 'DECLINED', 'EXPIRED', 'CLOSED', 'PAID', 'COMPLETED', 'WRITTEN_OFF', 'DRAFT'];
	const waiting = (view: PurchaseView) => ['SENT', 'BUYER_REVIEWING'].includes(view.request?.state ?? '');
	const creditHref = (view: PurchaseView) =>
		view.request?.id ? `/workspace/purchases/orders/${encodeURIComponent(view.request.id)}` : '';
</script>

<WorkspacePage
	eyebrow=""
	title="What I owe"
	description="Each supplier, what you took, how much is left and when to pay."
	endpoint={buyerEndpoint('/api/v1/buyer/credit-requests')}
	collectionKey="requests"
	emptyTitle="You owe nothing right now"
	emptyCopy="When a supplier gives you goods on credit, it shows here for you to accept."
	searchPlaceholder="Supplier or goods"
	keep={(view: PurchaseView) => !closed.includes(view.request?.state ?? '')}
	rowTitle={(view) => view.request?.supplier_trading_name || view.request?.supplier_legal_name || 'Supplier'}
	rowDetail={(view) =>
		[view.request?.goods_description, waiting(view) ? 'Waiting for you to accept' : ''].filter(Boolean).join(' · ')}
	rowStatus={(view) => (waiting(view) ? '' : (view.obligation?.payment_status ?? view.request?.state ?? ''))}
	rowAmount={(view) => view.obligation?.outstanding_kobo ?? view.request?.principal_kobo ?? null}
	rowAmountLabel="Left to pay"
	rowHref={creditHref}
/>
