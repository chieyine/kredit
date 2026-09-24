<script lang="ts">
	import { readableDate, readableDateTime } from '$lib/datetime';
	import { nairaInput } from '$lib/money';
	import { onMount, tick } from 'svelte';
	import { purchase, read, Mutation, money, label, actionLabel, eventLabel, kobo, type Purchase } from '$lib/consumer';
	let { id, organization = '', admin = false }: { id: string; organization?: string; admin?: boolean } = $props();
	let sale = $state<Purchase | null>(null),
		error = $state(''),
		message = $state(''),
		busy = $state(false),
		fullName = $state(''),
		address = $state(''),
		consent = $state(false),
		action = $state(''),
		amount = $state(''),
		reference = $state(''),
		related = $state(''),
		note = $state(''),
		occurred = $state('');
	const mutation = new Mutation();
	const endpoint = $derived(
		admin
			? `/api/v1/ops/consumer-sales/${encodeURIComponent(id)}`
			: organization
				? `/api/v1/organizations/${encodeURIComponent(organization)}/consumer-sales/${encodeURIComponent(id)}`
				: `/api/v1/buyer/purchases/${encodeURIComponent(id)}`
	);
	const buyer = $derived(!admin && !organization);
	const pendingClaims = $derived(
		sale?.events.filter(
			(e) =>
				e.action === 'claim' &&
				!sale?.events.some((d) => d.related_id === e.id && ['payment', 'reject_claim'].includes(d.action))
		) ?? []
	);
	const payments = $derived(
		sale?.events.filter(
			(e) =>
				e.action === 'payment' && !sale?.events.some((d) => d.related_id === e.id && d.action === 'reverse_payment')
		) ?? []
	);
	async function load() {
		busy = true;
		error = '';
		try {
			sale = purchase(await read(endpoint));
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not load this purchase.';
		} finally {
			busy = false;
		}
	}
	// What just happened, said to whoever did it.
	function doneMessage(kind: string) {
		const said: Record<string, string> = {
			accept: 'You accepted. The retailer can see your details and the payment dates are set.',
			decline: 'You declined this offer. Nothing is owed.',
			cancel: 'Cancelled. Any confirmed payment is now refundable.',
			claim: 'Payment reported. The retailer checks it arrived before your balance changes.',
			payment: 'Payment confirmed. The balance has been updated.',
			reject_claim: 'Marked as not found. The customer can see this.',
			reverse_payment: 'Payment corrected. The balance has been updated.',
			release: 'Dispatch recorded. The customer is asked to confirm they received the goods.',
			received: 'Thank you. The goods are recorded as received.',
			request_return: 'Your request is with the retailer. You will see their answer here.',
			approve_return: 'Return approved. Record the refund once you have paid it.',
			reject_return: 'Return declined. The customer can ask Kredit support to review it.',
			escalate: 'Sent to Kredit support. They will look at the retailer’s decision.',
			reduce_price: 'Price reduced. Any overpayment is now refundable.',
			refund: 'Refund recorded.'
		};
		return said[kind] ?? 'Saved. The record below has been updated.';
	}
	// A prompt for the evidence box, in the words of whoever is filling it in.
	function hint(kind: string) {
		if (buyer)
			return kind === 'request_return'
				? 'For example: the screen has a crack along the bottom edge. I noticed it when I unpacked it on Friday.'
				: 'Say what happened in your own words.';
		if (kind === 'release') return 'For example: delivered by our van, driver Musa, waybill WB-7781.';
		if (['payment', 'reverse_payment', 'reject_claim', 'refund'].includes(kind))
			return 'What you checked: for example the bank statement line or the signed cash receipt.';
		return 'Describe what happened and the evidence you checked.';
	}
	// The form opens at the end of the page; take the reader to it, or the
	// button seems to do nothing.
	function choose(kind: string) {
		action = kind;
		void tick().then(() => {
			const heading = document.querySelector<HTMLElement>('.action-form h2');
			heading?.scrollIntoView({ block: 'start' });
			heading?.focus({ preventScroll: true });
		});
		amount = '';
		reference = '';
		related = '';
		note = '';
		occurred = '';
		message = '';
	}
	function useClaim(id: string) {
		related = id;
		const e = pendingClaims.find((e) => e.id === id);
		if (e) {
			amount = nairaInput(e.amount_kobo);
			reference = e.reference;
			occurred = e.occurred_at;
		}
	}
	async function submit() {
		if (!sale || busy) return;
		busy = true;
		error = '';
		try {
			let body: Record<string, unknown> = {
				action,
				version: sale.version,
				agreement_hash: sale.agreement_hash,
				full_name: fullName,
				delivery_address: address,
				consent,
				note,
				related_id: related,
				reference
			};
			if (['payment', 'claim', 'refund', 'reduce_price'].includes(action)) body.amount_kobo = kobo(amount);
			if (['payment', 'claim', 'refund'].includes(action)) {
				const found = pendingClaims.find((e) => e.id === related);
				body.occurred_at = found?.occurred_at ?? new Date(occurred).toISOString();
			}
			const data = await mutation.send(endpoint, body);
			sale = purchase(data);
			message = doneMessage(action);
			action = '';
		} catch (e) {
			error = e instanceof Error ? e.message : 'The action was not confirmed.';
		} finally {
			busy = false;
		}
	}
	async function retry() {
		busy = true;
		error = '';
		try {
			sale = purchase(await mutation.retry());
			action = '';
			message = 'Previous action confirmed.';
		} catch (e) {
			error = e instanceof Error ? e.message : 'Could not confirm previous action.';
		} finally {
			busy = false;
		}
	}
	function print() {
		window.print();
	}
	onMount(load);
</script>

<main class="shell workspace feature-page purchase">
	<a
		href={admin
			? '/admin/consumer-sales'
			: organization
				? `/workspace/sales/consumers?organization=${organization}`
				: '/personal/purchases'}>{buyer ? '← Your purchases' : '← Consumer sales'}</a
	>
	{#if error}<p role="alert" class="error">{error}</p>
		{#if mutation.pending}<button onclick={retry} disabled={busy}>Retry the same unconfirmed action</button
			>{/if}{/if}{#if message}<p role="status">{message}</p>{/if}
	{#if !sale}<p>{busy ? 'Opening purchase…' : 'Purchase unavailable.'}</p>
		<button onclick={load} disabled={busy}>Try again</button>{:else}
		<header class="feature-heading purchase-heading">
			<div>
				<p class="eyebrow">
					{buyer ? 'Your purchase' : 'Consumer sale'}
					<span class="status">{buyer && sale.state === 'offered' ? 'Waiting for your answer' : label(sale.state)}</span
					>
				</p>
				<h1>{sale.terms.item}</h1>
				<p>Sold by {sale.terms.seller_name} · Quantity {sale.terms.quantity}</p>
			</div>
			<div class="heading-actions">
				<button class="secondary" onclick={print}>Print agreement and receipts</button>{#if !buyer}<a
						href="/account/security">Verify security for sensitive actions</a
					>{/if}<button onclick={load} disabled={busy}>Refresh</button>
			</div>
		</header>
		<section class="purchase-metrics metric-grid">
			<article>
				<small>Total agreed price</small><strong>{money(sale.terms.total_kobo)}</strong>
				<p>Price reductions: {money(sale.reduction_kobo)}. No added interest or late fees.</p>
			</article>
			<article>
				<small>Confirmed receipts</small><strong>{money(sale.paid_kobo)}</strong>
				<p>Refunds recorded: {money(sale.refunded_kobo)}</p>
			</article>
			<article>
				<small>{sale.state === 'offered' ? 'Price remaining on acceptance' : 'Left to pay'}</small><strong
					>{money(sale.outstanding_kobo)}</strong
				>
				<p>Refund still due: {money(sale.refund_due_kobo)}</p>
			</article>
		</section>
		<section>
			<h2>Your agreement</h2>
			<p>Retailer address: {sale.terms.seller_address}</p>
			<p>
				Customer: {sale.customer_name || sale.target}. Delivery address: {sale.delivery_address ||
					'Customer confirms at acceptance.'}
			</p>
			<p>
				Delivery: {sale.terms.fulfillment === 'immediate'
					? 'after acceptance'
					: sale.terms.fulfillment === 'on_full_payment'
						? 'after full payment'
						: `after ${sale.terms.threshold_percent}% is paid`}, within {sale.terms.delivery_days} days of meeting that condition.
				The retailer has confirmed stock reservation.
			</p>
			<p>{sale.terms.returns_policy}</p>
			<p>
				Before release, cancellation entitles you to a full refund of confirmed payments. After release, report a
				delivery problem or request a return. Your statutory consumer rights still apply. Kredit records the sale; the
				retailer supplies the goods and owes any refund.
			</p>
			<small>Agreement {sale.terms.terms_version} · {sale.agreement_hash}</small>
			{#if sale.state === 'offered' && buyer}<button onclick={() => choose('accept')} disabled={busy}
					>Review and accept</button
				><button onclick={() => choose('decline')} disabled={busy}>Decline offer</button>{/if}
		</section>
		<section>
			<h2>Payment dates</h2>
			<p>Manage optional payment reminders in <a href="/account/notifications">notification settings</a>.</p>
			<p>Payments go directly to this retailer. They do not automatically pay any wholesaler debt.</p>
			<div class="table">
				<table class="schedule">
					<thead><tr><th>Date</th><th>Scheduled</th><th>Paid</th></tr></thead><tbody
						>{#each sale.schedule_progress as due, i (i)}<tr
								><td>{readableDate(due.date)}</td><td>{money(due.amount_kobo)}</td><td>{money(due.paid_kobo)}</td></tr
							>{/each}</tbody
					>
				</table>
			</div>
			{#if sale.state === 'cancelled'}<p>
					This schedule is cancelled. Check the refund balance above.
				</p>{:else if sale.reduction_kobo > 0}<p>
					The remaining balance above includes the price reduction; do not pay more than that balance.
				</p>{/if}
			{#if sale.accepted_at && sale.state !== 'cancelled' && sale.outstanding_kobo > 0}<h3>Pay the retailer</h3>
				<p>{sale.terms.bank_name}<br />{sale.terms.account_name}<br /><strong>{sale.terms.account_number}</strong></p>
				<p>
					Confirm the recipient in your banking app. Report the transfer below after paying. A payment report is not a
					confirmed receipt.
				</p>{/if}
			{#if sale.accepted_at && buyer}<button onclick={() => choose('claim')} disabled={busy}
					>Report a payment I made</button
				>{/if}
			{#if sale.accepted_at && !buyer && sale.state !== 'cancelled'}<button
					onclick={() => choose('payment')}
					disabled={busy}>Confirm money received</button
				>{/if}{#if sale.accepted_at && !buyer}<button
					onclick={() => choose('reverse_payment')}
					disabled={busy || !payments.length}>Correct a recorded receipt</button
				>{/if}
			{#if pendingClaims.length}<h3>Payments awaiting review</h3>
				{#each pendingClaims as claim, i (i)}<p>
						{money(claim.amount_kobo)} · {claim.reference} · {claim.note}
					</p>{/each}{#if !buyer}<button onclick={() => choose('reject_claim')} disabled={busy}
						>Review an unmatched payment</button
					>{/if}{/if}
		</section>
		<section>
			<h2>Delivery and returns</h2>
			<p>
				{sale.received_at
					? 'Receipt confirmed by the customer.'
					: sale.released_at
						? 'Retailer recorded delivery. Waiting for customer confirmation.'
						: sale.release_eligible
							? 'Ready for the retailer to deliver.'
							: 'Goods have not been released.'}
			</p>
			{#if sale.delivery_due_at && !sale.received_at}<p>
					Delivery deadline: {readableDate(sale.delivery_due_at)}
				</p>{/if}{#if sale.case_state}<p>Return case: {label(sale.case_state)}</p>{/if}
			{#if !buyer && sale.release_eligible}<button disabled={busy} onclick={() => choose('release')}
					>Record dispatch or handover</button
				>{/if}
			{#if buyer && sale.released_at && !sale.received_at && sale.state !== 'cancelled'}<button
					disabled={busy}
					onclick={() => choose('received')}>Confirm I received the goods</button
				>{/if}
			{#if !sale.released_at && !['cancelled', 'declined'].includes(sale.state) && !(buyer && sale.state === 'offered')}<button
					disabled={busy}
					onclick={() => choose('cancel')}>{buyer ? 'Cancel purchase' : 'Cancel this sale'}</button
				>{/if}
			{#if buyer && sale.released_at && sale.state !== 'cancelled' && !['requested', 'escalated'].includes(sale.case_state)}<button
					disabled={busy}
					onclick={() => choose('request_return')}>Report a problem or request a return</button
				>{/if}
			{#if buyer && sale.case_state === 'rejected'}<button disabled={busy} onclick={() => choose('escalate')}
					>Ask Kredit to review the decision</button
				>{/if}
			{#if !buyer && (sale.case_state === 'requested' || (admin && sale.case_state === 'escalated'))}<button
					disabled={busy}
					onclick={() => choose('approve_return')}>Approve return and full refund</button
				><button disabled={busy} onclick={() => choose('reject_return')}>Decline return with a reason</button>{/if}
			{#if !buyer && sale.accepted_at && sale.state !== 'cancelled'}<button
					disabled={busy}
					onclick={() => choose('reduce_price')}>Reduce the sale price</button
				>{/if}
			{#if !buyer && sale.refund_due_kobo > 0}<button disabled={busy} onclick={() => choose('refund')}
					>Record refund paid to customer</button
				>{/if}
			<a href="/legal/complaints">Get help with this purchase</a>
		</section>
		{#if action}<form
				class="action-form"
				onsubmit={(e) => {
					e.preventDefault();
					void submit();
				}}
			>
				<h2 tabindex="-1">{actionLabel(action)}</h2>
				<fieldset disabled={busy}>
					{#if action === 'accept'}<label
							>Full name<input bind:value={fullName} required maxlength="200" autocomplete="name" /></label
						><label
							>Delivery address<textarea
								bind:value={address}
								required
								minlength="10"
								maxlength="1000"
								autocomplete="street-address"
							></textarea></label
						><label
							><input type="checkbox" bind:checked={consent} required />I am 18 or older. I accept the price, payment
							dates, delivery and return terms above, and acknowledge the
							<a href="/legal/privacy">privacy notice</a>.</label
						>{/if}
					{#if ['payment', 'reject_claim'].includes(action) && pendingClaims.length}<label
							>Customer payment report<select value={related} onchange={(e) => useClaim(e.currentTarget.value)}
								><option value="">{action === 'payment' ? 'Record a separate receipt' : 'Choose a report'}</option
								>{#each pendingClaims as e (e.id)}<option value={e.id}>{e.reference} · {money(e.amount_kobo)}</option
									>{/each}</select
							></label
						>{/if}
					{#if action === 'reverse_payment'}<label
							>Receipt to reverse<select bind:value={related} required
								><option value="">Choose receipt</option>{#each payments as e (e.id)}<option value={e.id}
										>{e.reference} · {money(e.amount_kobo)}</option
									>{/each}</select
							></label
						>{/if}
					{#if ['payment', 'claim', 'refund', 'reduce_price'].includes(action)}<label
							>Amount (₦)<input bind:value={amount} inputmode="decimal" required readonly={Boolean(related)} /></label
						>{/if}
					{#if ['payment', 'claim', 'refund'].includes(action)}<label
							>Bank transfer or signed cash receipt reference<input
								bind:value={reference}
								required
								minlength="3"
								maxlength="200"
								readonly={Boolean(related)}
							/></label
						>{#if !related}<label>When money moved<input type="datetime-local" bind:value={occurred} required /></label
							>{/if}{/if}
					{#if !['accept', 'received', 'decline'].includes(action)}<label
							>{buyer ? 'What happened' : 'Reason and evidence'}<textarea
								bind:value={note}
								required
								minlength="20"
								maxlength="2000"
								placeholder={hint(action)}
							></textarea></label
						>{/if}
					{#if action === 'refund'}<p>This records a refund you have already paid. It does not send money.</p>{/if}
					<button class="primary" type="submit">Confirm</button>
				</fieldset>
			</form>{/if}
		<section class="history">
			<h2>Purchase history</h2>
			{#each sale.events as event, i (i)}<article>
					<strong>{eventLabel(event.action)}{event.amount_kobo > 0 ? ` · ${money(event.amount_kobo)}` : ''}</strong>
					<p>{event.note}</p>
					<small
						>{readableDateTime(event.occurred_at)}{event.reference && !/^version-\d+$/.test(event.reference)
							? ` · ${event.reference}`
							: ''}</small
					>
				</article>{:else}<p>No actions recorded yet.</p>{/each}
		</section>
	{/if}
</main>

<style>
	/* three short columns fit a phone; the page-wide minimum width would only
	   make this one scroll sideways */
	table.schedule {
		min-width: 0;
	}
	/* history notes and references carry agreement hashes, which must wrap */
	.history p,
	.history small {
		overflow-wrap: anywhere;
	}
</style>
