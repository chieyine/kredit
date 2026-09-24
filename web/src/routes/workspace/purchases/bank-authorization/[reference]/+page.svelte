<script lang="ts">
	import { workspaceHref } from '$lib/workspace-navigation';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { checkedJSON, optionalNumber, optionalText, record, text, publicError } from '$lib/api/reliable';
	import { kobo } from '$lib/records';
	import type { KoboValue } from '$lib/money';
	import { MutationIntent } from '$lib/api/mutation';
	import Money from '$lib/components/Money.svelte';
	type Transfer = {
		amount_kobo: KoboValue;
		bank_name: string;
		account_name: string;
		account_number: string;
		expires_at: string;
	};
	type Enrollment = {
		state: 'DRAFT' | 'STARTED' | 'CONFIRMED' | 'CANCELLED';
		version: number;
		result: { transfer: Transfer | null; authorization_url: string };
	};
	type Mandate = { id: string; status: string; amount_ceiling_kobo: KoboValue };
	const enrollmentStates = ['DRAFT', 'STARTED', 'CONFIRMED', 'CANCELLED'] as const;
	let enrollment: Enrollment | null = $state(null),
		mandate: Mandate | null = $state(null),
		busy = $state(false),
		error = $state('');
	let banks: { code: string; name: string }[] = $state([]);
	let name = $state(''),
		email = $state(''),
		phone = $state(''),
		address = $state(''),
		bankCode = $state(''),
		accountNumber = $state(''),
		consent = $state(false);
	const path = $derived(`/api/v1/buyer/bank-authorization/${encodeURIComponent(page.params.reference ?? '')}`);
	function transfer(value: unknown): Transfer | null {
		if (value == null) return null;
		// These are the bank details a customer sends money to, so every field
		// must be present exactly; a partial instruction is never shown.
		const t = record(value);
		const expires = text(t.expires_at);
		if (!Number.isFinite(Date.parse(expires))) throw new Error('Incomplete transfer instructions');
		return {
			amount_kobo: kobo(t.amount_kobo),
			bank_name: text(t.bank_name),
			account_name: text(t.account_name),
			account_number: text(t.account_number),
			expires_at: expires
		};
	}
	function decode(value: unknown) {
		const data = record(value),
			e = record(data.enrollment),
			m = record(data.mandate);
		const state = enrollmentStates.find((candidate) => candidate === text(e.state));
		if (!state || !text(m.id)) throw new Error('Incomplete bank permission');
		const result = e.result && typeof e.result === 'object' ? record(e.result) : {};
		const verified: { enrollment: Enrollment; mandate: Mandate } = {
			enrollment: {
				state,
				version: optionalNumber(e.version),
				result: { transfer: transfer(result.transfer), authorization_url: optionalText(result.authorization_url) }
			},
			mandate: {
				id: text(m.id),
				status: optionalText(m.status),
				amount_ceiling_kobo: m.amount_ceiling_kobo == null ? null : kobo(m.amount_ceiling_kobo)
			}
		};
		return {
			...verified,
			banks: Array.isArray(data.banks)
				? data.banks.map((value) => {
						const b = record(value);
						return { code: text(b.code), name: text(b.name) };
					})
				: []
		};
	}
	async function load() {
		busy = true;
		error = '';
		try {
			const data = await checkedJSON(path, decode);
			enrollment = data.enrollment;
			mandate = data.mandate;
			banks = data.banks;
		} catch (cause) {
			error = publicError(cause, 'your bank permission');
		} finally {
			busy = false;
		}
	}
	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (busy) return;
		busy = true;
		error = '';
		try {
			const data = await new MutationIntent(`bank-authorization-${enrollment?.version ?? 0}`, path).run(
				{ name, email, phone, address, bank_code: bankCode, account_number: accountNumber, consent },
				decode
			);
			enrollment = data.enrollment;
			mandate = data.mandate;
			banks = data.banks;
			accountNumber = '';
		} catch (cause) {
			error = publicError(cause, 'your bank permission');
			await load();
		} finally {
			busy = false;
		}
	}
	function bankURL(value: unknown) {
		try {
			const u = new URL(String(value));
			return u.protocol === 'https:' &&
				!u.username &&
				!u.password &&
				!u.port &&
				(u.hostname === 'paylink.monnify.com' || u.hostname === 'sandbox.monnify.com')
				? u.href
				: null;
		} catch {
			return null;
		}
	}
	onMount(() => {
		void load();
	});
</script>

<svelte:head><title>Set up bank permission — Kredit</title></svelte:head>
<main class="shell workspace">
	<a class="back" href={workspaceHref('/workspace/purchases/mandates', page.url)}>← Bank permissions</a>
	<p class="eyebrow">Bank permission</p>
	<h1>Review bank authorisation</h1>
	<p class="lede">
		Authorize scheduled payments for your sale. You can stop future debit requests from your bank permissions page.
	</p>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
	{#if mandate}<section class="summary">
			<span>Permission limit</span><strong><Money amountKobo={mandate.amount_ceiling_kobo} /></strong>
			<p>This permission does not increase what you owe. Kredit checks your outstanding balance before each debit.</p>
		</section>{/if}
	{#if !enrollment}<p role="status">
			{busy ? 'Opening your bank permission…' : 'Your bank permission could not be loaded.'}
		</p>
		<button disabled={busy} onclick={load}>Try again</button>
	{:else if mandate?.status === 'ACTIVE'}<section class="panel">
			<h2>Your bank permission is active</h2>
			<p>You can return to your sale. Payments will follow its agreed schedule.</p>
			<a href={workspaceHref('/workspace/purchases/orders', page.url)}>Your sales →</a>
		</section>
	{:else if enrollment.state === 'DRAFT'}
		<form onsubmit={submit}>
			<h2>Your account details</h2>
			<p>Use an account in your own name. Your bank details are encrypted and sent to the selected payment provider.</p>
			<div class="fields">
				<label>Full name<input bind:value={name} required minlength="2" maxlength="100" autocomplete="name" /></label
				><label>Email<input type="email" bind:value={email} required autocomplete="email" /></label><label
					>Phone number<input
						type="tel"
						bind:value={phone}
						required
						pattern={String.raw`\+?[0-9]{10,15}`}
						autocomplete="tel"
					/></label
				><label
					>Home address<input
						bind:value={address}
						required
						minlength="5"
						maxlength="100"
						autocomplete="street-address"
					/></label
				><label
					>Bank<select bind:value={bankCode} required
						><option value="" disabled>Choose your bank</option>{#each banks as bank (bank.code)}<option
								value={bank.code}>{bank.name}</option
							>{/each}</select
					></label
				><label
					>Account number<input
						bind:value={accountNumber}
						required
						pattern={String.raw`[0-9]{10}`}
						maxlength="10"
						inputmode="numeric"
						autocomplete="off"
					/></label
				>
			</div>
			<label class="consent"
				><input type="checkbox" bind:checked={consent} required /><span
					>I authorize bank verification and agree to set up scheduled debits for this sale, within the permission limit
					shown above. My bank will ask me to complete authorization.</span
				></label
			>
			<button disabled={busy}>{busy ? 'Setting up your permission…' : 'Continue to bank authorization'}</button>
		</form>
	{:else if enrollment.state === 'STARTED'}<section class="panel">
			<h2>We’re checking the bank response</h2>
			<p>
				Your request may have reached the bank. Please do not submit another one. Support can reconcile this saved
				request if confirmation is delayed.
			</p>
			<button disabled={busy} onclick={load}>Check again</button><a href="/contact">Contact support</a>
		</section>
	{:else if enrollment.state === 'CANCELLED'}<section class="panel">
			<h2>This permission is closed</h2>
			<p>Open your sale to give fresh bank permission.</p>
			<a href={workspaceHref('/workspace/purchases/orders', page.url)}>Your sales →</a>
		</section>
	{:else}<section class="panel">
			<h2>Complete your bank’s authorization</h2>
			{#if enrollment.result?.transfer}{@const transfer = enrollment.result.transfer}
				<p>
					Transfer <strong><Money amountKobo={transfer.amount_kobo} /></strong> from the account you just entered. This is
					the bank’s activation fee, not a repayment for your sale.
				</p>
				<dl>
					<dt>Bank</dt>
					<dd>{transfer.bank_name}</dd>
					<dt>Account name</dt>
					<dd>{transfer.account_name}</dd>
					<dt>Account number</dt>
					<dd>{transfer.account_number}</dd>
					<dt>Complete before</dt>
					<dd>{new Date(transfer.expires_at).toLocaleString('en-NG', { timeZone: 'Africa/Lagos' })} (Lagos time)</dd>
				</dl>
				<p>Do not transfer after this time. Contact support if the activation window has expired.</p>
			{:else if bankURL(enrollment.result?.authorization_url)}<a
					class="primary"
					href={bankURL(enrollment.result.authorization_url)}>Continue securely with your bank →</a
				>
			{:else}<p>
					Your bank’s authorization link may take a moment to arrive. Check your email and refresh these instructions.
				</p>
				<button disabled={busy} onclick={load}>Refresh instructions</button>{/if}
			<p>
				After authorizing, open your bank permissions and select “Check bank confirmation”. Bank activation may take
				some time.
			</p>
			<a href={workspaceHref('/workspace/purchases/mandates', page.url)}>Check bank permission →</a>
		</section>{/if}
</main>

<style>
	main {
		max-width: 58rem;
	}
	.back {
		display: inline-block;
		margin-bottom: 1.5rem;
	}
	.summary,
	.panel,
	form {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 1rem;
		padding: 1.5rem;
		margin-top: 1.5rem;
	}
	.summary > span {
		display: block;
		color: var(--color-muted);
		font-size: 0.875rem;
	}
	.summary > strong {
		display: block;
		font-size: 1.8rem;
		margin-top: 0.4rem;
	}
	.summary p,
	.panel p,
	form > p {
		color: var(--color-muted);
		line-height: 1.6;
	}
	.fields {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 1rem;
		margin: 1.5rem 0;
	}
	label {
		display: grid;
		gap: 0.4rem;
	}
	select,
	input:not([type='checkbox']) {
		width: 100%;
		padding: 0.8rem;
		border: 1px solid var(--color-border);
		border-radius: 0.5rem;
		background: var(--color-surface);
		color: inherit;
		font: inherit;
	}
	.consent {
		display: flex;
		align-items: flex-start;
		gap: 0.7rem;
		margin: 1.5rem 0;
		line-height: 1.5;
	}
	.consent input {
		margin-top: 0.3rem;
	}
	dl {
		display: grid;
		grid-template-columns: 9rem 1fr;
		gap: 0.8rem;
		padding: 1rem;
		background: var(--color-background);
		border-radius: 0.6rem;
	}
	dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
	dt {
		color: var(--color-muted);
	}
	.error {
		color: var(--color-destructive);
	}
	.panel > a {
		display: inline-block;
		margin: 1rem 1rem 0 0;
	}
	@media (max-width: 600px) {
		.fields {
			grid-template-columns: 1fr;
		}
		form,
		.panel,
		.summary {
			padding: 1.1rem;
		}
		dl {
			grid-template-columns: 1fr;
			gap: 0.35rem;
		}
		dd {
			margin-bottom: 0.7rem;
		}
	}
</style>
