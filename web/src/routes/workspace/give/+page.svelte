<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { ACCOUNT_CONTEXT, type AccountContext } from '$lib/account-context';
	import { requestedWorkspace } from '$lib/workspace-context';
	import {
		checkedJSON,
		csrfHeader,
		LatestRequest,
		normalizeNigerianPhone,
		readResource,
		record,
		RequestError,
		rows,
		text,
		type Resource
	} from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { customer, organization, dateLabel, type Customer, type Organization } from '$lib/records';
	import { parseNaira, verbalizeNaira } from '$lib/money';
	import { loadSaleDefaults, paymentDateAfter } from '$lib/sale-defaults';
	import { Mutation, purchase, type Purchase } from '$lib/consumer';
	import Money from '$lib/components/Money.svelte';
	import ResourceNotice from '$lib/components/ResourceNotice.svelte';
	import ShareActions from '$lib/components/ShareActions.svelte';

	const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
	const RETURNS =
		'The goods were handed over at the time of sale. Report any fault or shortage to the seller straight away. Your statutory rights are unaffected.';

	let organizations = $state<Resource<Organization[]>>({ state: 'loading', scope: '' });
	let customers = $state<Resource<Customer[]>>({ state: 'loading', scope: '' });
	let organizationID = $state('');
	let kind = $state<'business' | 'person' | ''>('');
	let selected = $state(''),
		addingNew = $state(false);
	// New business customer
	let newName = $state(''),
		newPhone = $state(''),
		newAddress = $state(''),
		newIndustry = $state(''),
		newType = $state('unregistered_business'),
		inviteLink = $state('');
	// Person
	let personPhone = $state('');
	// The credit
	let goods = $state(''),
		amountText = $state(''),
		dueDate = $state(paymentDateAfter(30)),
		graceHours = $state(24);
	let busy = $state(false),
		error = $state(''),
		personLink = $state('');
	let creation: MutationIntent | null = null,
		sending: MutationIntent | null = null,
		invitation: MutationIntent | null = null;
	const personMutation = new Mutation();
	const reads = new LatestRequest();

	const amount = $derived(parseNaira(amountText));
	const amountWords = $derived(verbalizeNaira(amount));
	const key = (item: Customer) => `${item.buyer_user_id}:${item.buyer_business_id}`;
	const buyer = $derived(
		customers.state === 'ready' ? customers.data.find((item) => key(item) === selected) : undefined
	);
	const today = paymentDateAfter(0);
	const scope = $derived(`?organization=${encodeURIComponent(organizationID)}`);
	const ready = $derived(
		goods.trim().length >= 3 &&
			amount >= 100 &&
			/^\d{4}-\d{2}-\d{2}$/.test(dueDate) &&
			dueDate >= today &&
			((kind === 'business' && Boolean(buyer)) || (kind === 'person' && personPhone.trim().length >= 10))
	);

	async function load() {
		const request = reads.begin();
		organizations = { state: 'loading', scope: account.userID };
		const result = await readResource(
			account.userID,
			'/api/v1/organizations',
			rows('organizations', organization),
			request.signal,
			'your businesses'
		);
		if (!request.current()) return;
		organizations = result;
		if (result.state !== 'ready') return;
		try {
			organizationID = requestedWorkspace(result.data);
		} catch {
			organizations = {
				state: 'error',
				scope: account.userID,
				status: 404,
				message: 'This business is not available in your account.'
			};
			return;
		}
		if (!organizationID) return;
		try {
			const defaults = await loadSaleDefaults(organizationID, request.signal);
			graceHours = defaults.graceHours;
			dueDate = defaults.dueDate;
		} catch {
			// The 30-day default above stands; the server checks the date again.
		}
		customers = await readResource(
			organizationID,
			`/api/v1/organizations/${encodeURIComponent(organizationID)}/customers`,
			rows('customers', customer),
			request.signal,
			'your customers'
		);
		if (!request.current()) return;
		// Coming from a customer's page, or "give again" on an old credit.
		const params = new URLSearchParams(location.search);
		if (params.has('goods')) goods = (params.get('goods') ?? '').slice(0, 500);
		if (params.has('amount')) amountText = (params.get('amount') ?? '').slice(0, 40);
		if (params.get('kind') === 'person') kind = 'person';
		if (customers.state === 'ready' && params.get('customer')) {
			const match = customers.data.filter(
				(item) =>
					item.buyer_user_id === params.get('customer') &&
					(!params.get('customer_business') || item.buyer_business_id === params.get('customer_business'))
			);
			if (match.length === 1) {
				kind = 'business';
				selected = key(match[0]);
			}
		}
	}

	async function inviteBusiness() {
		if (busy) return;
		busy = true;
		error = '';
		try {
			const path = `/api/v1/organizations/${encodeURIComponent(organizationID)}/buyer-invitations`;
			invitation ??= new MutationIntent('give-invite', path);
			inviteLink = await invitation.run(
				{
					target: normalizeNigerianPhone(newPhone),
					target_type: 'phone',
					legal_name: newName.trim(),
					trading_name: '',
					business_type: newType,
					business_address: newAddress.trim(),
					industry: newIndustry.trim()
				},
				(value) => {
					const url = new URL(text(record(value).invitation_url), location.origin);
					if (!url.pathname.startsWith('/buyer-invitations/'))
						throw new Error('The customer link could not be checked.');
					return url.href;
				}
			);
		} catch (cause) {
			error = cause instanceof Error ? cause.message : 'We could not confirm the customer link. Try again.';
		} finally {
			busy = false;
		}
	}

	async function giveToBusiness() {
		if (!buyer) throw new Error('Choose the customer.');
		const org = organizationID;
		const collectionAt = await checkedJSON(
			`/api/v1/organizations/${encodeURIComponent(org)}/credit-terms/preview`,
			(value) => {
				const result = record(value),
					at = text(result.collection_at);
				if (result.due_date !== dueDate || !Number.isFinite(Date.parse(at)))
					throw new Error('The payment date could not be checked.');
				return at;
			},
			{
				method: 'POST',
				headers: { 'Content-Type': 'application/json', ...csrfHeader() },
				body: JSON.stringify({ due_date: dueDate, grace_hours: graceHours })
			}
		);
		const base = `/api/v1/organizations/${encodeURIComponent(org)}/credit-requests`;
		creation ??= new MutationIntent(`${account.userID}:${org}`, base);
		const id = await creation.run(
			{
				buyer_user_id: buyer.buyer_user_id,
				buyer_business_id: buyer.buyer_business_id,
				buyer_legal_name: buyer.legal_name,
				buyer_trading_name: buyer.trading_name,
				principal_kobo: amount,
				goods_description: goods.trim(),
				invoice_reference: '',
				invoice_document_hash: '',
				due_date: dueDate,
				grace_hours: graceHours,
				collection_at: collectionAt,
				timing_mode: 'lagos_end_of_day',
				schedule_type: 'one_time',
				schedule_count: 1,
				schedule_cadence: 'custom',
				month_end_policy: 'last_day',
				custom_schedule_items: []
			},
			(value) => {
				const id = text(record(record(value).request).id);
				if (!id) throw new Error('The credit was not confirmed.');
				return id;
			}
		);
		sending ??= new MutationIntent(`${account.userID}:${org}`, `${base}/${encodeURIComponent(id)}/send`);
		try {
			await sending.run(undefined, (value) => record(record(value).request));
		} catch {
			// Saved but not sent: the credit page shows the Send button.
		}
		await goto(`/workspace/sales/${encodeURIComponent(id)}?organization=${encodeURIComponent(org)}&given=1`);
	}

	async function giveToPerson() {
		const data = (await personMutation.send(
			`/api/v1/organizations/${encodeURIComponent(organizationID)}/consumer-sales`,
			{
				target_type: 'phone',
				target: normalizeNigerianPhone(personPhone),
				terms: {
					item: goods.trim(),
					quantity: 1,
					total_kobo: amount,
					deposit_kobo: 0,
					deposit_date: '',
					first_date: dueDate,
					count: 1,
					cadence: 'monthly',
					fulfillment: 'immediate',
					threshold_percent: 0,
					delivery_days: 1,
					stock_reference: 'Handed over at the shop',
					stock_reserved: true,
					returns_policy: RETURNS
				}
			},
			purchase
		)) as Purchase;
		personLink = `${location.origin}/personal/purchases/${data.id}`;
	}

	async function give(event: SubmitEvent) {
		event.preventDefault();
		if (busy || !ready) return;
		busy = true;
		error = '';
		try {
			if (kind === 'business') await giveToBusiness();
			else await giveToPerson();
		} catch (cause) {
			error =
				cause instanceof RequestError && [401, 403, 423].includes(cause.status)
					? cause.message
					: cause instanceof Error
						? cause.message
						: 'We could not confirm this credit. Check your list before trying again.';
		} finally {
			busy = false;
		}
	}

	onMount(() => {
		void load();
		return () => reads.cancel();
	});
</script>

<svelte:head><title>Give goods on credit — Kredit</title></svelte:head>
<main class="shell give">
	<p><a href="/workspace/today{scope}">← Who owes me</a></p>
	<h1>Give goods on credit</h1>
	<ResourceNotice resource={organizations} label="Your business" retry={load} />

	{#if organizations.state === 'ready' && !organizations.data.length}
		<section class="card">
			<h2>Add your business first</h2>
			<p>We need your business name and the bank account your money goes to.</p>
			<a class="primary" href="/workspace/today">Add my business</a>
		</section>
	{:else if personLink}
		<section class="card done" aria-live="polite">
			<h2>Done. Send him the link.</h2>
			<p>
				Your customer opens this link, confirms his phone number, and says yes to the credit. You will see it on
				<a href="/workspace/today{scope}">Who owes me</a>.
			</p>
			<input value={personLink} readonly aria-label="Customer link" />
			<ShareActions
				title="Goods on credit"
				text={`Hello, this is the record of the goods you took on credit (${goods}). Please open the link to confirm.`}
				url={personLink}
			/>
			<p><a href="/workspace/give{scope}">Give another credit</a></p>
		</section>
	{:else if organizationID}
		<form class="card" onsubmit={give} aria-busy={busy}>
			<fieldset class="who" disabled={busy}>
				<legend>Who is taking the goods?</legend>
				<div class="choice">
					<label class:on={kind === 'business'}
						><input type="radio" bind:group={kind} value="business" /><strong>A business</strong><small
							>Shop, supermarket, distributor</small
						></label
					>
					<label class:on={kind === 'person'}
						><input type="radio" bind:group={kind} value="person" /><strong>A person</strong><small
							>Buying for himself or his home</small
						></label
					>
				</div>
			</fieldset>

			{#if kind === 'business'}
				<ResourceNotice resource={customers} label="Your customers" retry={load} />
				{#if customers.state === 'ready'}
					{#if customers.data.length && !addingNew}
						<label
							>Customer<select bind:value={selected} disabled={busy}
								><option value="">Choose the customer</option>{#each customers.data as item (key(item))}<option
										value={key(item)}>{item.trading_name || item.legal_name}</option
									>{/each}</select
							></label
						>
						{#if buyer?.overdue}<p class="warn">
								This customer is owing past his due date. Think before giving more.
							</p>{/if}
						<button type="button" class="link" onclick={() => (addingNew = true)}>New customer? Add him here</button>
					{:else}
						<div class="new-customer">
							{#if inviteLink}
								<h2>Send him this link first</h2>
								<p>
									He opens it and confirms his business. Once he has done that, come back here and his name will be on
									the list.
								</p>
								<input value={inviteLink} readonly aria-label="Customer link" />
								<ShareActions
									title="Your Kredit customer link"
									text={`Hello ${newName}, please open this link to confirm your business so I can give you goods on credit.`}
									url={inviteLink}
								/>
							{:else}
								<h2>New business customer</h2>
								<p>He gets a link on WhatsApp to confirm his business. This is done once.</p>
								<label>Business name<input bind:value={newName} autocomplete="off" disabled={busy} /></label>
								<label
									>WhatsApp number<input
										bind:value={newPhone}
										type="tel"
										inputmode="tel"
										placeholder="0803 000 0000"
										disabled={busy}
									/></label
								>
								<label
									>Shop address<input
										bind:value={newAddress}
										placeholder="Shop number, street, town"
										disabled={busy}
									/></label
								>
								<label
									>What does he sell?<input
										bind:value={newIndustry}
										placeholder="Provisions, drinks, building materials…"
										disabled={busy}
									/></label
								>
								<label
									>Is the business registered?<select bind:value={newType} disabled={busy}
										><option value="unregistered_business">Not registered</option><option value="registered_business"
											>Business name (CAC)</option
										><option value="limited_company">Limited company</option></select
									></label
								>
								<button
									type="button"
									class="secondary"
									disabled={busy || !newName.trim() || !newPhone.trim() || !newAddress.trim() || !newIndustry.trim()}
									onclick={inviteBusiness}>Send him the link</button
								>
							{/if}
							{#if customers.data.length}<button type="button" class="link" onclick={() => (addingNew = false)}
									>Back to my customer list</button
								>{/if}
						</div>
					{/if}
				{/if}
			{:else if kind === 'person'}
				<label
					>His WhatsApp number<input
						bind:value={personPhone}
						type="tel"
						inputmode="tel"
						placeholder="0803 000 0000"
						disabled={busy}
					/></label
				>
			{/if}

			{#if (kind === 'business' && buyer) || kind === 'person'}
				<fieldset class="credit" disabled={busy}>
					<legend>The goods</legend>
					<label
						>What goods?<textarea
							bind:value={goods}
							rows="2"
							maxlength="500"
							placeholder="For example: 20 cartons of Indomie, 5 bags of rice"
						></textarea></label
					>
					<label
						>How much? (₦)<input
							bind:value={amountText}
							inputmode="decimal"
							placeholder="150,000"
						/>{#if amountWords}<small>{amountWords}</small>{/if}</label
					>
					<label>Pay by<input type="date" bind:value={dueDate} min={today} /></label>
				</fieldset>
				{#if ready}<p class="summary">
						<strong>{kind === 'business' ? buyer?.trading_name || buyer?.legal_name : personPhone}</strong> owes you
						<strong><Money amountKobo={amount} /></strong> by <strong>{dateLabel(dueDate)}</strong>.
					</p>{/if}
				{#if error}<p class="error" role="alert">{error}</p>{/if}
				<button class="primary big" disabled={busy || !ready}>{busy ? 'Sending…' : 'Send to customer'}</button>
			{:else if error}<p class="error" role="alert">{error}</p>{/if}
		</form>
	{/if}
</main>

<style>
	.give {
		max-width: 38rem;
		padding-bottom: 3rem;
	}
	h1 {
		margin: 0.5rem 0 1.25rem;
		font-size: clamp(1.8rem, 5vw, 2.4rem);
		line-height: 1.15;
	}
	form,
	.done {
		display: grid;
		gap: 1.1rem;
	}
	fieldset {
		display: grid;
		gap: 1rem;
		margin: 0;
		padding: 0;
		border: 0;
		min-width: 0;
	}
	legend {
		margin-bottom: 0.6rem;
		font-weight: 700;
		font-size: 1.1rem;
	}
	label {
		display: grid;
		gap: 0.4rem;
		font-weight: 600;
	}
	label small {
		color: var(--color-muted);
		font-weight: 400;
	}
	input,
	select,
	textarea {
		font: inherit;
		font-weight: 400;
		padding: 0.75rem;
		border: 1px solid var(--color-border);
		border-radius: 0.4rem;
		background: var(--color-surface);
		width: 100%;
		box-sizing: border-box;
	}
	.choice {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.75rem;
	}
	.choice label {
		gap: 0.2rem;
		padding: 1rem;
		border: 2px solid var(--color-border);
		border-radius: 0.6rem;
		cursor: pointer;
	}
	.choice label.on {
		border-color: var(--color-primary);
	}
	.choice input {
		position: absolute;
		opacity: 0;
		width: 1px;
	}
	.choice label:focus-within {
		outline: 2px solid var(--focus-ring);
		outline-offset: 2px;
	}
	.new-customer {
		display: grid;
		gap: 0.9rem;
		padding: 1rem;
		border: 1px dashed var(--color-border);
		border-radius: 0.6rem;
	}
	.new-customer h2 {
		margin: 0;
		font-size: 1.1rem;
	}
	.new-customer p {
		margin: 0;
		color: var(--color-muted);
	}
	.link {
		justify-self: start;
		padding: 0.4rem 0;
		border: 0;
		background: none;
		color: var(--color-primary);
		font: inherit;
		text-decoration: underline;
		cursor: pointer;
	}
	.warn {
		margin: 0;
		color: var(--color-accent);
	}
	.summary {
		margin: 0;
		padding: 1rem;
		background: var(--color-background);
		border-radius: 0.5rem;
		line-height: 1.6;
	}
	.big {
		min-height: 3.25rem;
		font-size: 1.05rem;
	}
	@media (max-width: 420px) {
		.choice {
			grid-template-columns: 1fr;
		}
	}
</style>
