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
		publicError,
		RequestError,
		rows,
		text,
		type Resource
	} from '$lib/api/reliable';
	import { MutationIntent } from '$lib/api/mutation';
	import { customer, organization, saleView, dateLabel, type Customer, type Organization } from '$lib/records';
	import { parseNaira, verbalizeNaira } from '$lib/money';
	import { loadSaleDefaults, paymentDateAfter } from '$lib/sale-defaults';
	import { Mutation, purchase, kobo, type Purchase } from '$lib/consumer';
	import PersonalSaleOptions from '$lib/components/PersonalSaleOptions.svelte';
	import { defaultPersonalOptions } from '$lib/personal-sale-options';
	import SaleOptions from '$lib/components/SaleOptions.svelte';
	import { defaultSaleOptions, scheduleTerms, InvoiceUpload } from '$lib/sale-options';
	import Money from '$lib/components/Money.svelte';
	import ResourceNotice from '$lib/components/ResourceNotice.svelte';
	import ShareActions from '$lib/components/ShareActions.svelte';
	import VerifyIdentity from '$lib/components/VerifyIdentity.svelte';

	const account = getContext<AccountContext>(ACCOUNT_CONTEXT);
	const RETURNS =
		'Before release, cancellation receives a full refund. Contact the seller about incorrect, defective or undelivered goods. Other returns are reviewed individually. Your statutory rights are unaffected.';

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
	let personPhone = $state(''),
		stockReserved = $state(false);
	// The credit
	let goods = $state(''),
		amountText = $state(''),
		dueDate = $state(''),
		graceHours = $state(24);
	let busy = $state(false),
		error = $state(''),
		personLink = $state(''),
		savedSale = $state(''),
		settingsReady = $state(false),
		settingsError = $state('');
	let creation: MutationIntent | null = null,
		invitation: MutationIntent | null = null;
	const personMutation = new Mutation();
	const reads = new LatestRequest();

	let options = $state(defaultSaleOptions());
	let personal = $state(defaultPersonalOptions(RETURNS));
	const invoiceUpload = new InvoiceUpload();

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
			((kind === 'business' && settingsReady && !addingNew && Boolean(buyer)) ||
				(kind === 'person' &&
					stockReserved &&
					amount <= 2_500_000_000 &&
					(personal.channel === 'phone'
						? personPhone.trim().length >= 10
						: /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(personPhone))))
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
		settingsError = '';
		settingsReady = false;
		try {
			const defaults = await loadSaleDefaults(organizationID, request.signal);
			if (!request.current()) return;
			graceHours = defaults.graceHours;
			if (!dueDate) dueDate = defaults.dueDate;
			settingsReady = true;
		} catch (cause) {
			if (!request.current()) return;
			settingsError = publicError(cause, 'your saved payment settings');
		}
		const customerResult = await readResource(
			organizationID,
			`/api/v1/organizations/${encodeURIComponent(organizationID)}/customers`,
			rows('customers', customer),
			request.signal,
			'your customers'
		);
		if (!request.current()) return;
		customers = customerResult;
		// Coming from a customer's page, or "give again" on an old credit.
		const params = new URLSearchParams(location.search);
		if (params.has('goods')) goods = (params.get('goods') ?? '').slice(0, 500);
		if (params.has('amount')) amountText = (params.get('amount') ?? '').slice(0, 40);
		if (params.get('kind') === 'person') kind = 'person';
		if (!dueDate) dueDate = paymentDateAfter(30);
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
		if (!buyer || !settingsReady) throw new Error('Choose a customer and confirm your payment settings.');
		const org = organizationID;
		const schedule = scheduleTerms(options, amount, dueDate);
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
				body: JSON.stringify({ due_date: dueDate, grace_hours: graceHours, principal_kobo: amount, ...schedule })
			}
		);
		const base = `/api/v1/organizations/${encodeURIComponent(org)}/credit-requests`;
		creation ??= new MutationIntent(`${account.userID}:${org}`, base);
		const invoice = await invoiceUpload.upload(account.userID, org, options.invoiceFile);
		const id = await creation.run(
			{
				buyer_user_id: buyer.buyer_user_id,
				buyer_business_id: buyer.buyer_business_id,
				buyer_legal_name: buyer.legal_name,
				buyer_trading_name: buyer.trading_name,
				principal_kobo: amount,
				goods_description: goods.trim(),
				invoice_reference: options.invoiceReference.trim(),
				invoice_document_hash: invoice.hash,
				invoice_document_id: invoice.id,
				due_date: dueDate,
				grace_hours: graceHours,
				collection_at: collectionAt,
				timing_mode: 'lagos_end_of_day',
				...schedule
			},
			(value) => {
				const view = saleView(value);
				if (!view.request.id || view.request.state !== 'DRAFT') throw new Error('The saved draft was not confirmed.');
				return view.request.id;
			}
		);
		savedSale = `/workspace/sales/${encodeURIComponent(id)}?organization=${encodeURIComponent(org)}&created=1`;
		// The saved record displays exact fees and terms before its Send action.
		await goto(savedSale);
	}

	async function giveToPerson() {
		const data = (await personMutation.send(
			`/api/v1/organizations/${encodeURIComponent(organizationID)}/consumer-sales`,
			{
				target_type: personal.channel,
				target: personal.channel === 'phone' ? normalizeNigerianPhone(personPhone) : personPhone.trim(),
				terms: {
					item: goods.trim(),
					quantity: personal.quantity,
					total_kobo: amount,
					deposit_kobo: kobo(personal.deposit),
					deposit_date: personal.depositDate,
					first_date: dueDate,
					count: personal.count,
					cadence: personal.cadence,
					fulfillment: personal.fulfillment,
					threshold_percent: personal.threshold,
					delivery_days: personal.days,
					stock_reference: 'Stock reserved for the agreed delivery terms',
					stock_reserved: stockReserved,
					returns_policy: personal.returns
				}
			},
			purchase
		)) as Purchase;
		personLink = `${location.origin}/personal/purchases/${data.id}`;
	}

	async function give(event: SubmitEvent) {
		event.preventDefault();
		if (busy || !ready || savedSale || personLink) return;
		busy = true;
		error = '';
		try {
			if (kind === 'business') await giveToBusiness();
			else await giveToPerson();
		} catch (cause) {
			error =
				cause instanceof RequestError && cause.code === 'invalid_phone'
					? cause.message
					: publicError(cause, 'We could not confirm this credit. Check your list before trying again.');
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
	{:else if savedSale}
		<section class="card done" aria-live="polite">
			<h2>Your draft is saved</h2>
			<p>Review the exact terms and fees before sending it to your customer.</p>
			<a class="primary" href={savedSale}>Review saved sale</a>
		</section>
	{:else if personLink}
		<section class="card done" aria-live="polite">
			<h2>Purchase saved. Share the link.</h2>
			<p>
				Your customer opens this link, signs in with the contact you invited, and reviews the terms. You will see it on
				<a href="/workspace/today{scope}">Who owes me</a>.
			</p>
			<input value={personLink} readonly aria-label="Customer link" />
			<ShareActions
				title="Goods on credit"
				text={`Hello, please review this offer for goods on credit (${goods}). Please open the link to confirm.`}
				url={personLink}
			/>
			<p>
				<button
					type="button"
					class="secondary"
					onclick={() => {
						personLink = '';
						personPhone = '';
						personal = defaultPersonalOptions(RETURNS);
						goods = '';
						amountText = '';
						stockReserved = false;
						error = '';
					}}>Give another credit</button
				>
			</p>
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
							>Buying for themselves or their home</small
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
								This customer has overdue payments. Review them before giving more credit.
							</p>{/if}
						<button
							type="button"
							class="link"
							disabled={busy}
							onclick={() => {
								addingNew = true;
								selected = '';
								inviteLink = '';
							}}>Add a new customer</button
						>
					{:else}
						<div class="new-customer">
							{#if inviteLink}
								<h2>Share this customer link first</h2>
								<p>They open it and confirm their business. Then refresh your customers to find them on the list.</p>
								<input value={inviteLink} readonly aria-label="Customer link" />
								<ShareActions
									title="Your Kredit customer link"
									text={`Hello ${newName}, please open this link to confirm your business so I can give you goods on credit.`}
									url={inviteLink}
								/>
								<button
									type="button"
									class="secondary"
									disabled={busy}
									onclick={() => {
										addingNew = false;
										void load();
									}}>Refresh customers</button
								>
							{:else}
								<h2>New business customer</h2>
								<p>Create a link and share it with your customer to confirm their business.</p>
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
									>What do they sell?<input
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
									onclick={inviteBusiness}>Create customer link</button
								>
							{/if}
							{#if customers.data.length}<button
									type="button"
									class="link"
									disabled={busy}
									onclick={() => (addingNew = false)}>Back to my customer list</button
								>{/if}
						</div>
					{/if}
				{/if}
			{:else if kind === 'person'}
				<label
					>{personal.channel === 'phone' ? 'Customer’s WhatsApp number' : 'Customer’s email'}<input
						bind:value={personPhone}
						type={personal.channel === 'phone' ? 'tel' : 'email'}
						inputmode={personal.channel === 'phone' ? 'tel' : 'email'}
						placeholder={personal.channel === 'phone' ? '0803 000 0000' : 'customer@example.com'}
						disabled={busy}
					/></label
				>
			{/if}

			{#if (kind === 'business' && buyer && !addingNew) || kind === 'person'}
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
				{#if kind === 'business'}
					<SaleOptions bind:options disabled={busy} />
					{#if settingsError}<p role="alert">
							{settingsError} <button type="button" disabled={busy} onclick={load}>Check again</button>
						</p>{/if}
					<p>You will review the exact fees and terms on the next page before sending the sale.</p>
				{:else}
					<PersonalSaleOptions bind:options={personal} disabled={busy} />
					<p>
						Goods are reserved for handover within {personal.days}
						{personal.days === 1 ? 'day' : 'days'}
						after {personal.fulfillment === 'immediate'
							? 'acceptance'
							: personal.fulfillment === 'on_full_payment'
								? 'full payment'
								: `${personal.threshold}% is paid`}. {personal.returns}
					</p>
					<label
						><input type="checkbox" bind:checked={stockReserved} disabled={busy} />I have reserved these goods and can
						meet these delivery and return terms.</label
					>
				{/if}
				{#if ready}<p class="summary">
						<strong>{kind === 'business' ? buyer?.trading_name || buyer?.legal_name : personPhone}</strong> will be
						asked to pay
						<strong><Money amountKobo={amount} /></strong>
						{(
							kind === 'business'
								? options.scheduleType !== 'one_time'
								: personal.count > 1 || parseNaira(personal.deposit) > 0
						)
							? 'starting'
							: 'by'}
						<strong
							>{dateLabel(
								kind === 'person' && parseNaira(personal.deposit) > 0 ? personal.depositDate : dueDate
							)}</strong
						>.
					</p>{/if}
				{#if error}<p class="error" role="alert">{error}</p>{/if}
				<button class="primary big" disabled={busy || !ready}
					>{busy ? 'Saving…' : kind === 'business' ? 'Review credit sale' : 'Create purchase link'}</button
				>
			{:else if error}<p class="error" role="alert">{error}</p>{/if}
		</form>
	{/if}
	{#if organizationID && !personLink && !savedSale}<VerifyIdentity />{/if}
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
		border-radius: var(--radius);
		background: var(--color-surface);
		width: 100%;
		box-sizing: border-box;
	}
	input[type='checkbox'] {
		width: auto;
		justify-self: start;
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
		border-radius: var(--radius);
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
		border-radius: var(--radius);
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
		border-radius: var(--radius);
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
